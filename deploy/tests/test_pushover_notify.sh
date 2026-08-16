#!/usr/bin/env bash
# Tests for deploy/pushover_notify.sh — the Pi 5 deploy's alarm (ops #2480, #2914).
#
# These drive a REAL curl against a REAL local HTTP server, rather than stubbing
# the call out. That is deliberate: ops #2480 existed precisely because nothing
# ever exercised the HTTP path — the alarm was `if: failure()`-only, so it ran
# solely during incidents, and the swallowed exit code hid the result when it
# did. A test that stubs the request would reproduce that same blind spot.
#
# Run: bash deploy/tests/test_pushover_notify.sh
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
SCRIPT="$HERE/../pushover_notify.sh"
TMP="$(mktemp -d)"
SRV_PID=""
cleanup() { [ -n "$SRV_PID" ] && kill "$SRV_PID" 2>/dev/null; rm -rf "$TMP"; }
trap cleanup EXIT

fail=0
ok()  { printf 'ok   - %s\n' "$1"; }
bad() { printf 'FAIL - %s\n' "$1"; fail=1; }

# --- a tiny stub Pushover -----------------------------------------------------
# Replies with whatever body/status the current mode file says, and records the
# request so we can assert the credentials actually travelled.
start_server() {
  cat > "$TMP/server.py" <<'PY'
import http.server, os, sys, json

STATE = os.environ["STUB_STATE"]

class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(n)
        with open(STATE + "/last_request", "wb") as fh:
            fh.write(body)
        with open(STATE + "/mode") as fh:
            mode = fh.read().strip()
        if mode == "valid":
            code, payload = 200, {"status": 1, "request": "abc"}
        elif mode == "invalid":
            code, payload = 400, {"status": 0, "errors": ["application token is invalid"]}
        else:
            code, payload = 500, {"status": 0, "errors": ["boom"]}
        # Compact separators, mirroring the real Pushover API's response shape.
        raw = json.dumps(payload, separators=(",", ":")).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def log_message(self, *a):
        pass

http.server.HTTPServer(("127.0.0.1", int(sys.argv[1])), H).serve_forever()
PY
  PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1]);s.close()')
  echo valid > "$TMP/mode"
  STUB_STATE="$TMP" python3 "$TMP/server.py" "$PORT" &
  SRV_PID=$!
  URL="http://127.0.0.1:$PORT/x"
  for _ in $(seq 1 50); do
    curl -s --max-time 1 -X POST "$URL" -o /dev/null 2>/dev/null && break
    sleep 0.1
  done
}

if ! command -v python3 >/dev/null 2>&1; then
  echo "skip - python3 unavailable"; exit 0
fi
start_server

run() {  # run <mode> [env assignments...]
  local m="$1"; shift
  env PUSHOVER_VALIDATE_URL="$URL" PUSHOVER_API_URL="$URL" \
      VERSION=v1.2.3 PI_HOST=birdnet-pi5 RUN_URL=http://run/1 \
      "$@" bash "$SCRIPT" "$m" 2>&1
}

# --- preflight: missing credentials is LOUD but non-fatal --------------------
out="$(run preflight PUSHOVER_API_TOKEN= PUSHOVER_APP_TOKEN= PUSHOVER_USER_KEY=)"; rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q '::warning::'; then
  ok "preflight: missing creds warns loudly and does not block the deploy"
else
  bad "preflight: missing creds warns loudly and does not block the deploy (rc=$rc)"; echo "$out"
fi

# --- preflight: REJECTED credentials are surfaced (the ops #2480 case) -------
# This is the case that matters most for THIS repo: its PUSHOVER_API_TOKEN was
# created 2026-05-10, predates the ops #2480 rename, and its value has never been
# validated from a run — the inline curl it replaced could not tell set-but-wrong
# from set-and-valid.
echo invalid > "$TMP/mode"
out="$(run preflight PUSHOVER_API_TOKEN=tok PUSHOVER_USER_KEY=usr)"; rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q '::error::'; then
  ok "preflight: set-but-rejected creds raise ::error:: (presence != validity)"
else
  bad "preflight: set-but-rejected creds raise ::error:: (rc=$rc)"; echo "$out"
fi
if printf '%s' "$out" | grep -q 'application token is invalid'; then
  ok "preflight: surfaces Pushover's own error text"
else
  bad "preflight: surfaces Pushover's own error text"; echo "$out"
fi

# --- preflight: valid credentials pass quietly -------------------------------
echo valid > "$TMP/mode"
out="$(run preflight PUSHOVER_API_TOKEN=tok PUSHOVER_USER_KEY=usr)"; rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q 'credentials validated'; then
  ok "preflight: valid creds pass"
else
  bad "preflight: valid creds pass (rc=$rc)"; echo "$out"
fi

# --- notify: a rejected alert is FATAL (the inline `curl -s … > /dev/null`
#     this replaces returned 0 here, which is the entire ops #2480 defect) -----
echo invalid > "$TMP/mode"
out="$(run notify PUSHOVER_API_TOKEN=tok PUSHOVER_USER_KEY=usr)"; rc=$?
if [ "$rc" -ne 0 ] && printf '%s' "$out" | grep -q '::error::'; then
  ok "notify: a rejected FAILURE alert EXITS NON-ZERO — the ops #2480 regression"
else
  bad "notify: a rejected FAILURE alert EXITS NON-ZERO (rc=$rc)"; echo "$out"
fi

# --- notify: missing creds is fatal too --------------------------------------
out="$(run notify PUSHOVER_API_TOKEN= PUSHOVER_APP_TOKEN= PUSHOVER_USER_KEY=)"; rc=$?
if [ "$rc" -ne 0 ]; then
  ok "notify: cannot-send is fatal rather than silent"
else
  bad "notify: cannot-send is fatal rather than silent (rc=$rc)"; echo "$out"
fi

# --- notify: happy path sends and the credentials actually travel ------------
echo valid > "$TMP/mode"
rm -f "$TMP/last_request"
out="$(run notify PUSHOVER_API_TOKEN=tok-canonical PUSHOVER_USER_KEY=usr)"; rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q 'failure alert sent'; then
  ok "notify: happy path sends"
else
  bad "notify: happy path sends (rc=$rc)"; echo "$out"
fi
# Assert on what was SENT, not merely that the call returned 0 — a stub that
# answers 200 to anything would otherwise pass with no credentials at all.
if grep -q 'tok-canonical' "$TMP/last_request" 2>/dev/null; then
  ok "notify: the token actually reached the wire"
else
  bad "notify: the token actually reached the wire"
fi
if grep -q 'v1.2.3' "$TMP/last_request" 2>/dev/null && \
   grep -q 'birdnet-pi5' "$TMP/last_request" 2>/dev/null; then
  ok "notify: the message names the version and the host"
else
  bad "notify: the message names the version and the host"
fi

# --- notify-success: rejection is LOUD but not fatal -------------------------
# Deliberately asymmetric with `notify`. A Pushover outage must not turn a good
# deploy red; the preflight in the same run is what raises ::error:: about the
# alarm being dead. Pinning both halves so neither is "fixed" into the other.
echo invalid > "$TMP/mode"
out="$(run notify-success PUSHOVER_API_TOKEN=tok PUSHOVER_USER_KEY=usr)"; rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q '::error::'; then
  ok "notify-success: a rejected notice is loud but does not fail a good deploy"
else
  bad "notify-success: a rejected notice is loud but non-fatal (rc=$rc)"; echo "$out"
fi

echo valid > "$TMP/mode"
rm -f "$TMP/last_request"
out="$(run notify-success PUSHOVER_API_TOKEN=tok-canonical PUSHOVER_USER_KEY=usr)"; rc=$?
if [ "$rc" -eq 0 ] && grep -q 'tok-canonical' "$TMP/last_request" 2>/dev/null; then
  ok "notify-success: happy path sends with the canonical token"
else
  bad "notify-success: happy path sends with the canonical token (rc=$rc)"; echo "$out"
fi
# multipart/form-data, so the value sits on its own line after the part header
# rather than as `priority=-1`. Assert the field AND the exact value line: a
# bare `grep -- -1` would match almost any body and pass with the feature gone.
if grep -q 'name="priority"' "$TMP/last_request" 2>/dev/null && \
   grep -qE -- $'^-1\r?$' "$TMP/last_request" 2>/dev/null; then
  ok "notify-success: sends at low priority so a good deploy does not page"
else
  bad "notify-success: sends at low priority"; echo "$out"
fi
# ...and the failure alert must be HIGH priority, or the two are indistinguishable.
rm -f "$TMP/last_request"
run notify PUSHOVER_API_TOKEN=tok PUSHOVER_USER_KEY=usr >/dev/null 2>&1
if grep -q 'name="priority"' "$TMP/last_request" 2>/dev/null && \
   grep -qE -- $'^1\r?$' "$TMP/last_request" 2>/dev/null; then
  ok "notify: the FAILURE alert is sent at high priority"
else
  bad "notify: the FAILURE alert is sent at high priority"
fi

# --- an unknown mode must fail, not silently no-op ---------------------------
out="$(run definitely-not-a-mode PUSHOVER_API_TOKEN=tok PUSHOVER_USER_KEY=usr)"; rc=$?
if [ "$rc" -eq 2 ]; then
  ok "an unknown mode exits non-zero rather than silently doing nothing"
else
  bad "an unknown mode exits non-zero (rc=$rc)"; echo "$out"
fi

# --- the retired PUSHOVER_APP_TOKEN name must not be honoured (ops #2495) ----
# stormcrow and proxmox-agent both carried an either-name fallback whose only
# reachable behaviour was to turn "canonical token missing" into "send with a
# known-bad credential". This repo never had one; these cases pin that it stays
# that way, so the port cannot re-introduce what ops #2495 removed.
rm -f "$TMP/last_request"
out="$(run notify PUSHOVER_API_TOKEN= PUSHOVER_APP_TOKEN=tok-legacy PUSHOVER_USER_KEY=usr)"; rc=$?
if [ "$rc" -ne 0 ] && ! grep -q 'tok-legacy' "$TMP/last_request" 2>/dev/null; then
  ok "notify: PUSHOVER_APP_TOKEN does NOT rescue a missing canonical token — fails loudly"
else
  bad "notify: PUSHOVER_APP_TOKEN still rescues a missing canonical token (rc=$rc)"; echo "$out"
fi
out="$(run preflight PUSHOVER_API_TOKEN= PUSHOVER_USER_KEY=)"
if ! printf '%s' "$out" | grep -q 'PUSHOVER_APP_TOKEN'; then
  ok "preflight: guidance names only the canonical secret"
else
  bad "preflight: mentions the retired PUSHOVER_APP_TOKEN"; echo "$out"
fi

# --- secrets must not leak into the deploy log -------------------------------
echo invalid > "$TMP/mode"
out="$(run preflight PUSHOVER_API_TOKEN=SUPERSECRETTOKEN PUSHOVER_USER_KEY=SUPERSECRETUSER)"
if ! printf '%s' "$out" | grep -q 'SUPERSECRETTOKEN' && \
   ! printf '%s' "$out" | grep -q 'SUPERSECRETUSER'; then
  ok "neither the token nor the user key is ever printed"
else
  bad "a credential leaked into the log"; echo "$out"
fi

# --- the workflow must actually CALL this script -----------------------------
# The script existing changes nothing on its own; ops #2914's whole finding is
# that a control which is merged but not wired reads exactly like a working one.
# Match the step invocation, and carry a positive control so a broken matcher
# cannot report a clean result.
WF="$HERE/../../.github/workflows/deploy-birdnet-pi5.yml"
if [ ! -f "$WF" ]; then
  bad "deploy-birdnet-pi5.yml not found at $WF — this guard silently checks nothing"
else
  calls="$(grep -cE 'deploy/pushover_notify\.sh' "$WF" || true)"
  steps="$(grep -cE '^[[:space:]]*- name:' "$WF" || true)"
  # Positive control: a workflow with no steps at all would give calls=0 too.
  if [ "$steps" -eq 0 ]; then
    bad "workflow guard is broken: parsed no steps from $WF at all"
  elif [ "$calls" -ge 3 ]; then
    ok "deploy-birdnet-pi5.yml invokes pushover_notify.sh ($calls sites, $steps steps)"
  else
    bad "deploy-birdnet-pi5.yml invokes pushover_notify.sh only $calls time(s); expected preflight + notify + notify-success"
  fi
  # ...and the inline curl it replaced must be gone, or both paths ship at once.
  inline="$(sed 's/#.*//' "$WF" | grep -cE 'api\.pushover\.net' || true)"
  if [ "$inline" -eq 0 ]; then
    ok "deploy-birdnet-pi5.yml no longer curls api.pushover.net inline"
  else
    bad "deploy-birdnet-pi5.yml still has $inline inline api.pushover.net call(s)"
  fi
fi

echo "----"
if [ "$fail" -eq 0 ]; then echo "ALL TESTS PASSED"; else echo "SOME TESTS FAILED"; fi
exit "$fail"
