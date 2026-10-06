#!/usr/bin/env bash
# Pushover credentials must never be in curl's argv (ops #4348). Any local user
# can read argv via `ps` or /proc/<pid>/cmdline; stdin (curl -K /dev/stdin) is
# private to the process.
#
# Part 1: a repo-wide scan of shipped shell files, with a positive control and a
#         canary planted INSIDE the walked tree (so it proves coverage, not just
#         that the regex works).
# Part 2: each changed sender's real send block is run against a stub curl on
#         PATH that records argv and stdin, using FAKE credentials.
#
# Run: bash deploy/tests/test_pushover_argv.sh
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

fail=0
ok()  { printf 'ok   - %s\n' "$1"; }
bad() { printf 'FAIL - %s\n' "$1"; fail=1; }

# ---------------------------------------------------------------- Part 1: scan
# curl flag followed by a token=/user= field whose value is NOT routed through
# the cfg_escape heredoc form.
PATTERN='(--data-urlencode|--data-raw|--data-binary|--data|--form-string|--form|-d|-F)\s+"?(token|user)=\$(?!\(cfg_escape )'

# Shipped shell files under $1, excluding test dirs and vendored/VCS trees.
list_shell_files() {
  find "$1" \( -name .git -o -name node_modules -o -name .worktrees -o -name vendor \
               -o -name tests -o -name test -o -name testdata \) -prune -o \
       -type f -name '*.sh' -print
}
# Print matching lines (file:line:text) across the walked tree; one grep per file
# so a failure to read one file cannot hide the rest.
scan_tree() {
  local f
  while IFS= read -r f; do
    grep -nP -e "$PATTERN" "$f" /dev/null
  done < <(list_shell_files "$1")
}

# Positive control on planted text: every flag spelling matches, the safe form
# and a non-credential field do not.
for flag in --data-urlencode --data-raw --data-binary --data --form-string --form -d -F; do
  for field in token user; do
    printf '%s\n' "curl $flag \"$field=\$SECRET\" url" > "$TMP/line"
    if grep -qP -e "$PATTERN" "$TMP/line"; then :; else bad "pattern misses: $flag $field"; fi
    printf '%s\n' "curl $flag $field=\${SECRET} url" > "$TMP/line"
    if grep -qP -e "$PATTERN" "$TMP/line"; then :; else bad "pattern misses unquoted: $flag $field"; fi
  done
done
printf '%s\n' '--form-string "token=$(cfg_escape "$TOKEN")"' > "$TMP/line"
if grep -qP -e "$PATTERN" "$TMP/line"; then bad "pattern flags the cfg_escape form"; else ok "pattern ignores the cfg_escape form"; fi
printf '%s\n' 'curl --form-string "title=$TITLE" url' > "$TMP/line"
if grep -qP -e "$PATTERN" "$TMP/line"; then bad "pattern flags a non-credential field"; else ok "pattern ignores non-credential fields"; fi
ok "pattern positive control ran (8 flags x 2 fields x 2 quoting styles)"

# Canary: a planted file in a throwaway tree, walked by the same function.
CAN="$TMP/canary_tree"; mkdir -p "$CAN/scripts" "$CAN/tests"
printf '%s\n' 'curl -d "token=$X&user=$Y" https://example.invalid' > "$CAN/scripts/leaky.sh"
printf '%s\n' 'curl -d "token=$X&user=$Y" https://example.invalid' > "$CAN/tests/ignored.sh"
hits="$(scan_tree "$CAN")"
if [ "$(printf '%s\n' "$hits" | grep -c 'scripts/leaky.sh')" -eq 1 ] && ! printf '%s' "$hits" | grep -q 'tests/ignored.sh'; then
  ok "canary found by the file walk; test dirs excluded"
else
  bad "canary not found / test dir not excluded (hits: $hits)"
fi

# Coverage as a number: the walk must reach the three known senders.
walked="$(list_shell_files "$ROOT")"
for need in deploy/pushover_notify.sh scripts/birdnet-health-check.sh scripts/birdnet-recovery.sh; do
  if printf '%s\n' "$walked" | grep -qF "/$need"; then :; else bad "file walk did not reach $need"; fi
done

hits="$(scan_tree "$ROOT")"
if [ -z "$hits" ]; then
  ok "no shipped shell file puts token=/user= in curl argv ($(printf '%s\n' "$walked" | grep -c .) files scanned)"
else
  bad "credential fields in curl argv:"; printf '%s\n' "$hits" | sed 's/^/       /'
fi

# --------------------------------------------------------- Part 2: stub curl
FAKE_TOK="faketoken_ABC123xyz"
FAKE_USR="fakeuser_QRS789"
mkdir -p "$TMP/bin"
cat > "$TMP/bin/curl" <<'STUB'
#!/usr/bin/env bash
# Records argv (one arg per line) and stdin; replies per STUB_OUT / STUB_RC.
# Callers give it </dev/null by default, so an argv-style sender (no stdin
# payload) records an empty stdin instead of blocking.
printf '%s\n' "$@" > "$STUB_DIR/argv"
cat > "$STUB_DIR/stdin"
printf '%s' "${STUB_OUT:-}"
exit "${STUB_RC:-0}"
STUB
chmod +x "$TMP/bin/curl"
export STUB_DIR="$TMP"

reset_stub() { rm -f "$TMP/argv" "$TMP/stdin"; }
# Assert: stub ran, creds absent from argv, present on stdin, and -K /dev/stdin used.
check_wire() { # label
  local label="$1"
  if [ ! -f "$TMP/argv" ]; then bad "$label: stub curl never ran"; return; fi
  if grep -qF -e "$FAKE_TOK" -e "$FAKE_USR" "$TMP/argv"; then bad "$label: credential in argv"; else ok "$label: credentials absent from argv"; fi
  if grep -qF "$FAKE_TOK" "$TMP/stdin" && grep -qF "$FAKE_USR" "$TMP/stdin"; then ok "$label: credentials present on stdin"; else bad "$label: credentials missing from stdin"; fi
  if grep -qxF -e '-K' "$TMP/argv" && grep -qxF '/dev/stdin' "$TMP/argv"; then ok "$label: curl invoked with -K /dev/stdin"; else bad "$label: no -K /dev/stdin in argv"; fi
}

# ---- deploy/pushover_notify.sh (all three send sites)
NOTIFY="$ROOT/deploy/pushover_notify.sh"
run_notify() { # mode  (env: STUB_RC STUB_OUT VERSION)
  PATH="$TMP/bin:$PATH" PUSHOVER_API_TOKEN="$FAKE_TOK" PUSHOVER_USER_KEY="$FAKE_USR" \
    PI_HOST=pi RUN_URL=http://run bash "$NOTIFY" "$1" >/dev/null 2>&1 </dev/null
}
for mode in preflight notify notify-success; do
  reset_stub
  STUB_RC=0 STUB_OUT='{"status":1}' run_notify "$mode"; rc=$?
  [ "$rc" -eq 0 ] && ok "pushover_notify $mode: rc 0 on stub success" || bad "pushover_notify $mode: rc $rc on stub success"
  check_wire "pushover_notify $mode"
  reset_stub
  STUB_RC=22 STUB_OUT='' run_notify "$mode"; rc=$?
  case "$mode" in
    notify)         want=1 ;;  # a dead alarm must fail the step
    notify-success) want=0 ;;  # loud, not fatal
    preflight)      want=0 ;;  # invalid creds are reported, not fatal
  esac
  [ "$rc" -eq "$want" ] && ok "pushover_notify $mode: rc $want on stub rc 22" || bad "pushover_notify $mode: rc $rc on stub rc 22 (want $want)"
  check_wire "pushover_notify $mode (rc 22)"
done
# Hostile value: quote + newline in a field must stay inside one config line.
reset_stub
VERSION=$'1"x\ny' STUB_RC=0 run_notify notify
if [ "$(wc -l < "$TMP/stdin")" -eq 5 ] && grep -qF '1\"x\ny' "$TMP/stdin"; then ok "pushover_notify: quote/newline escaped, 5 config lines"; else bad "pushover_notify: hostile value not contained"; fi

# ---- scripts/birdnet-health-check.sh: send_pushover_alert()
HC="$ROOT/scripts/birdnet-health-check.sh"
extract_fn() { awk -v n="$2" '$0 ~ "^"n"\\(\\) \\{"{p=1} p{print} p&&/^}/{exit}' "$1"; }
{ extract_fn "$HC" cfg_escape; extract_fn "$HC" send_pushover_alert; } > "$TMP/hc_fns.sh"
if [ "$(grep -c '^cfg_escape() {' "$TMP/hc_fns.sh")" -eq 1 ] && [ "$(grep -c '^send_pushover_alert() {' "$TMP/hc_fns.sh")" -eq 1 ]; then
  ok "health-check: extracted cfg_escape and send_pushover_alert"
else bad "health-check: function extraction failed"; fi
run_hc() { # (env STUB_RC STUB_OUT)
  PATH="$TMP/bin:$PATH" bash -c '
    log() { echo "$1" >> "$HCLOG"; }
    PUSHOVER_API_TOKEN="$FAKE_TOK"; PUSHOVER_USER_KEY="$FAKE_USR"
    . "$HCFNS"; send_pushover_alert "BOYA down"' </dev/null
}
export HCFNS="$TMP/hc_fns.sh" HCLOG="$TMP/hc.log" FAKE_TOK FAKE_USR
reset_stub; : > "$HCLOG"
STUB_RC=0 STUB_OUT=200 run_hc; rc=$?
[ "$rc" -eq 0 ] && grep -q 'Pushover alert sent' "$HCLOG" && ok "health-check: rc 0 and logged success on HTTP 200" || bad "health-check: success path (rc $rc)"
check_wire "health-check"
reset_stub; : > "$HCLOG"
STUB_RC=7 STUB_OUT=000 run_hc; rc=$?
[ "$rc" -eq 1 ] && grep -q 'Pushover request failed' "$HCLOG" && ok "health-check: curl rc 7 -> return 1 and logged failure" || bad "health-check: curl-failure path (rc $rc)"
check_wire "health-check (rc 7)"

# ---- scripts/birdnet-recovery.sh: the send block (dead code, fixed anyway)
RC="$ROOT/scripts/birdnet-recovery.sh"
{ extract_fn "$RC" cfg_escape
  awk '/^ *message="BOYA/{p=1} p{print} p&&/^CFG$/{exit}' "$RC"; } > "$TMP/rec_blk.sh"
if grep -q '^CFG$' "$TMP/rec_blk.sh" && grep -q 'curl -sf -X POST' "$TMP/rec_blk.sh"; then ok "recovery: extracted send block"; else bad "recovery: block extraction failed"; fi
for rc_stub in 0 22; do
  reset_stub
  PATH="$TMP/bin:$PATH" STUB_RC=$rc_stub PUSHOVER_API_TOKEN="$FAKE_TOK" PUSHOVER_USER_KEY="$FAKE_USR" \
    bash -c 'set -euo pipefail; . "$1"; echo reached-end' _ "$TMP/rec_blk.sh" > "$TMP/rec.out" 2>&1 </dev/null; rc=$?
  { [ "$rc" -eq 0 ] && grep -q reached-end "$TMP/rec.out"; } && ok "recovery: stub rc $rc_stub never aborts the script (|| true kept)" || bad "recovery: aborted on stub rc $rc_stub (rc $rc)"
  check_wire "recovery (stub rc $rc_stub)"
  grep -qF 'title=BirdNET Hard Desync' "$TMP/stdin" && grep -qF 'priority=1' "$TMP/stdin" && ok "recovery: title and priority kept" || bad "recovery: title/priority lost"
done

echo "----"
if [ "$fail" -eq 0 ]; then echo "ALL TESTS PASSED"; else echo "SOME TESTS FAILED"; fi
exit "$fail"
