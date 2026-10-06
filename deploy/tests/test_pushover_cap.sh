#!/usr/bin/env bash
# Every Pushover sender must cap its message at 1024 characters (ops #4372).
# Pushover rejects a longer message outright, so the page is lost.
#
# Part 1: ENUMERATE senders from the tracked files (not a hand-written list) and
#         require each to carry the cap logic. Finding none is a failure.
# Part 2: drive each known sender with a 3000-char message against a stub curl
#         and assert the message field that reaches curl is <= 1024 characters.
#         A sender with no driver here fails, so a new sender cannot slip by.
#
# Run: bash deploy/tests/test_pushover_cap.sh
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
export LC_ALL=C.UTF-8

fail=0
ok()  { printf 'ok   - %s\n' "$1"; }
bad() { printf 'FAIL - %s\n' "$1"; fail=1; }

# ------------------------------------------------------------ Part 1: enumerate
# Tracked files that POST to Pushover. Tests, docs and archive/ are not senders.
list_senders() {
  git -C "$ROOT" grep -l -I -e 'api\.pushover\.net' -e 'PUSHOVER_URL' -e 'messages\.json' -- . \
      ':(exclude)*.md' ':(exclude)doc' ':(exclude)docs' ':(exclude)archive' \
      ':(exclude)*/archive/*' ':(exclude)*/tests/*' ':(exclude)*/test/*' \
      ':(exclude)*/testdata/*' ':(exclude).github' ':(exclude)frontend' \
    | LC_ALL=C sort
}
senders="$(list_senders)"
n=$(printf '%s\n' "$senders" | grep -c .)
if [ "$n" -ge 1 ]; then ok "enumerated $n Pushover sender file(s)"; else bad "enumeration found no senders (it must find at least one)"; fi
for need in deploy/pushover_notify.sh scripts/birdnet-health-check.sh scripts/birdnet-recovery.sh; do
  printf '%s\n' "$senders" | grep -qxF "$need" || bad "enumeration did not reach known sender $need"
done

has_cap() { # file: limit, suffix and the ops marker all present
  grep -qF 'ops #4372' "$1" && grep -qF '1024' "$1" && grep -qF '… (truncated)' "$1"
}
while IFS= read -r f; do
  [ -n "$f" ] || continue
  if has_cap "$ROOT/$f"; then ok "$f carries the 1024-char cap"; else bad "$f is a Pushover sender with NO cap (ops #4372)"; fi
done <<< "$senders"

# ----------------------------------------------------------- Part 2: behaviour
FAKE_TOK="faketoken_ABC123xyz"
FAKE_USR="fakeuser_QRS789"
mkdir -p "$TMP/bin"
cat > "$TMP/bin/curl" <<'STUB'
#!/usr/bin/env bash
cat > "$STUB_DIR/stdin"
printf '%s' "${STUB_OUT:-}"
exit 0
STUB
chmod +x "$TMP/bin/curl"
export STUB_DIR="$TMP"

# Characters in the message field that reached curl (undoing cfg_escape's \n etc.
# is unnecessary: the long inputs here contain nothing it rewrites).
sent_message() { sed -n 's/^--form-string "message=\(.*\)"$/\1/p' "$TMP/stdin"; }
check_len() { # label
  local m len
  m="$(sent_message)"; len=${#m}
  if [ "$len" -gt 0 ] && [ "$len" -le 1024 ]; then ok "$1: message is $len chars (<= 1024)"; else bad "$1: message is $len chars"; fi
  case "$m" in *'… (truncated)') ok "$1: carries the truncation suffix" ;; *) bad "$1: no truncation suffix" ;; esac
}
LONG_ASCII="$(printf 'a%.0s' $(seq 1 3000))"
LONG_UTF8="$(printf 'é%.0s' $(seq 1 3000))"   # 6000 bytes, 3000 characters

drive_notify() { # mode version
  rm -f "$TMP/stdin"
  PATH="$TMP/bin:$PATH" PUSHOVER_API_TOKEN="$FAKE_TOK" PUSHOVER_USER_KEY="$FAKE_USR" \
    VERSION="$2" PI_HOST=pi RUN_URL=http://run bash "$ROOT/deploy/pushover_notify.sh" "$1" >/dev/null 2>&1 </dev/null
}
extract_fn() { awk -v n="$2" '$0 ~ "^"n"\\(\\) \\{"{p=1} p{print} p&&/^}/{exit}' "$1"; }

drivers=""
for v in "$LONG_ASCII" "$LONG_UTF8"; do
  for mode in notify notify-success; do
    drive_notify "$mode" "$v"; check_len "pushover_notify $mode (${#v} chars in)"
  done
done
drivers="$drivers deploy/pushover_notify.sh"
# Short messages must pass through untouched.
drive_notify notify "1.2.3"
[ "$(sent_message)" = "Version 1.2.3 failed to deploy to pi. See http://run" ] && ok "pushover_notify: short message unchanged" || bad "pushover_notify: short message altered"

HC="$ROOT/scripts/birdnet-health-check.sh"
{ extract_fn "$HC" cfg_escape; extract_fn "$HC" send_pushover_alert; } > "$TMP/hc_fns.sh"
for v in "$LONG_ASCII" "$LONG_UTF8"; do
  rm -f "$TMP/stdin"
  PATH="$TMP/bin:$PATH" MSG="$v" bash -c '
    log() { :; }
    PUSHOVER_API_TOKEN='"$FAKE_TOK"'; PUSHOVER_USER_KEY='"$FAKE_USR"'
    . "'"$TMP"'/hc_fns.sh"; send_pushover_alert "$MSG"' >/dev/null 2>&1 </dev/null
  check_len "health-check send_pushover_alert (${#v} chars in)"
done
drivers="$drivers scripts/birdnet-health-check.sh"

# recovery: the message is a literal, so substitute a long one into the real block.
RC="$ROOT/scripts/birdnet-recovery.sh"
for v in "$LONG_ASCII" "$LONG_UTF8"; do
  { extract_fn "$RC" cfg_escape
    awk '/^ *message="BOYA/{p=1} p{print} p&&/^CFG$/{exit}' "$RC" \
      | LONGV="$v" awk '/^ *message="BOYA/{print "message=\"" ENVIRON["LONGV"] "\""; next} {print}'; } > "$TMP/rec_blk.sh"
  rm -f "$TMP/stdin"
  PATH="$TMP/bin:$PATH" PUSHOVER_API_TOKEN="$FAKE_TOK" PUSHOVER_USER_KEY="$FAKE_USR" \
    bash -c 'set -euo pipefail; . "$1"' _ "$TMP/rec_blk.sh" >/dev/null 2>&1 </dev/null
  check_len "recovery send block (${#v} chars in)"
done
drivers="$drivers scripts/birdnet-recovery.sh"

# Every enumerated sender needs a behavioural driver above.
while IFS= read -r f; do
  [ -n "$f" ] || continue
  case " $drivers " in *" $f "*) ;; *) bad "$f has no behavioural driver in this test" ;; esac
done <<< "$senders"

echo "----"
if [ "$fail" -eq 0 ]; then echo "ALL TESTS PASSED"; else echo "SOME TESTS FAILED"; fi
exit "$fail"
