#!/usr/bin/env bash
# Pushover alerting for the Pi 5 deploy — credential resolution, preflight
# validation, and the notifications themselves.
#
# Why this is a script and not inline YAML (ops #2480, ops #2914)
# --------------------------------------------------------------
# deploy-birdnet-pi5.yml used to notify with two inline
# `curl -s … > /dev/null` calls and no preflight at all. That path cannot
# report its own failure in either direction:
#
#   1. No `--fail`, so a Pushover rejection (HTTP 4xx with a JSON error body)
#      still exits 0 and the step reports success. `-o /dev/null` then throws
#      away the body that said what was wrong.
#   2. No preflight, so a missing or invalid credential produces NO signal
#      whatsoever on a green deploy — it just silently notifies nobody.
#
# ops #2480 is the incident this exists to prevent: stormcrow's only alarm on a
# failed production deploy was an inline `curl … || true`, it returned HTTP 400,
# the step reported success, and a genuinely failed deploy told nobody. An
# alerting path that fails OPEN is worse than no alerting at all, because the
# absence of alerts reads as "nothing went wrong".
#
# ops #2914 measured this repo as the last of three PUSHOVER-using deploy
# workflows still carrying the inline shape. The other two (stormcrow,
# proxmox-agent) run this script; this is the port, and the mode names match
# theirs so the fleet reads the same.
#
# Extracted here so the logic is testable at all — inline YAML in an
# `if: failure()` step can only ever be exercised by failing a real deploy.
# See deploy/tests/test_pushover_notify.sh.
#
# Modes:
#   preflight       — validate the credentials WITHOUT sending anything, so a
#                     broken notification path is discovered on a GREEN deploy
#                     rather than during the incident that needed it. Never
#                     fatal: a Pushover outage must not block shipping. Loud
#                     instead — ::warning::/::error:: annotations plus a
#                     $GITHUB_STEP_SUMMARY line.
#   notify          — send the deploy-FAILURE alert. FATAL on failure (no
#                     `|| true`), so a broken alarm shows as a red step instead
#                     of vanishing. This is the alarm; it must not fail open.
#   notify-success  — send the deploy-succeeded notice. Loud but NOT fatal: a
#                     Pushover outage must not turn a good deploy red. The
#                     preflight above is what carries the dead-alarm signal, and
#                     it runs on every deploy including this one.
#
# Never prints a token or user key — only presence, and the API's own status.
#
# Test hooks: PUSHOVER_API_URL / PUSHOVER_VALIDATE_URL override the endpoints;
#             GITHUB_STEP_SUMMARY is honoured if set (Actions sets it).
set -u

MODE="${1:-preflight}"
VALIDATE_URL="${PUSHOVER_VALIDATE_URL:-https://api.pushover.net/1/users/validate.json}"
MESSAGES_URL="${PUSHOVER_API_URL:-https://api.pushover.net/1/messages.json}"

log() { echo "[pushover] $*"; }

summary() {
  # Surface on the run's summary page, not only in the step log, so a warning on
  # a green deploy is actually seen. No-op when unset (local runs / tests).
  [ -n "${GITHUB_STEP_SUMMARY:-}" ] && printf '%s\n' "$*" >> "$GITHUB_STEP_SUMMARY"
  return 0
}

# PUSHOVER_API_TOKEN is the only accepted name, matching stormcrow and
# proxmox-agent. No PUSHOVER_APP_TOKEN fallback is accepted here and none should
# be added: ops #2480 renamed the secret, ops #2495 deleted the stale one
# fleet-wide, and the fallback's only reachable behaviour was to turn "canonical
# token missing" into "send with a known-bad credential" — degrading silently
# instead of failing loudly. Missing is missing.
TOKEN="${PUSHOVER_API_TOKEN:-}"
TOKEN_SRC="PUSHOVER_API_TOKEN"
USER_KEY="${PUSHOVER_USER_KEY:-}"

missing=""
[ -z "$TOKEN" ] && missing="token"
[ -z "$USER_KEY" ] && missing="${missing:+$missing and }user key"

VERSION="${VERSION:-unknown}"
PI_HOST="${PI_HOST:-the Pi}"
RUN_URL="${RUN_URL:-}"

case "$MODE" in
  preflight)
    if [ -n "$missing" ]; then
      log "::warning::Pushover $missing not configured — a FAILED deploy will notify nobody."
      log "         Set PUSHOVER_API_TOKEN and PUSHOVER_USER_KEY"
      log "         as repository secrets. This is not fatal, but it means the deploy's"
      log "         only failure alarm is dead (ops #2480, ops #2914)."
      summary "⚠️ **Pushover $missing missing** — a failed deploy will not notify anyone (ops #2480)."
      exit 0
    fi

    # Capability-test the actual values. Presence is not validity: the 400 that
    # started ops #2480 came from a credential that WAS set in the workflow, and
    # this repo's PUSHOVER_API_TOKEN predates that rename — its value has never
    # been validated from a run. users/validate.json checks token+user without
    # sending a notification.
    body="$(curl -sS --max-time 10 \
              --form-string "token=${TOKEN}" \
              --form-string "user=${USER_KEY}" \
              "$VALIDATE_URL" 2>/dev/null)" || body=""
    # Whitespace-tolerant: JSON permits spaces after the colon, and matching the
    # compact form only would silently report a VALID credential as rejected.
    if printf '%s' "$body" | grep -qE '"status":[[:space:]]*1'; then
      log "credentials validated via ${VALIDATE_URL} (token from ${TOKEN_SRC})"
      exit 0
    fi
    # Pushover echoes the offending field in `errors`; that names what is wrong
    # without revealing either value.
    errs="$(printf '%s' "$body" | sed -n 's/.*"errors":[[:space:]]*\[\([^]]*\)\].*/\1/p')"
    log "::error::Pushover credentials are set but REJECTED — a failed deploy will notify nobody."
    log "         endpoint: ${VALIDATE_URL}"
    log "         token source: ${TOKEN_SRC}"
    log "         pushover said: ${errs:-<no parseable error; check the token/user pair>}"
    summary "🚨 **Pushover credentials rejected** (${TOKEN_SRC}) — the deploy's failure alarm is dead (ops #2480). Pushover said: ${errs:-unparseable}"
    # Deliberately non-fatal: a Pushover outage must not block a deploy. The
    # ::error:: annotation and summary line are the alarm.
    exit 0
    ;;

  notify)
    if [ -n "$missing" ]; then
      log "::error::cannot send the deploy-failure alert: Pushover $missing not configured."
      summary "🚨 **Deploy failed AND the alert could not be sent** — Pushover $missing missing."
      exit 1
    fi
    # No `|| true`, and `-f` so an HTTP 4xx is an error rather than a body we
    # discard. If the alarm cannot fire, that must be visible as a failed step;
    # swallowing it is the whole of ops #2480.
    if curl -fsS --max-time 10 \
         --form-string "token=${TOKEN}" \
         --form-string "user=${USER_KEY}" \
         --form-string "priority=1" \
         --form-string "title=BirdNET-Go deploy FAILED" \
         --form-string "message=Version ${VERSION} failed to deploy to ${PI_HOST}. See ${RUN_URL}" \
         "$MESSAGES_URL" -o /dev/null; then
      log "failure alert sent (token from ${TOKEN_SRC})"
      exit 0
    fi
    log "::error::the deploy-failure Pushover alert was REJECTED (token from ${TOKEN_SRC})."
    summary "🚨 **Deploy failed AND the alert was rejected by Pushover** (ops #2480)."
    exit 1
    ;;

  notify-success)
    if [ -n "$missing" ]; then
      log "::warning::deploy succeeded but the notice could not be sent: Pushover $missing not configured."
      summary "⚠️ **Deploy succeeded; the Pushover notice was not sent** — $missing missing."
      exit 0
    fi
    if curl -fsS --max-time 10 \
         --form-string "token=${TOKEN}" \
         --form-string "user=${USER_KEY}" \
         --form-string "priority=-1" \
         --form-string "title=BirdNET-Go deployed to Pi 5" \
         --form-string "message=Version ${VERSION} is live on ${PI_HOST}." \
         "$MESSAGES_URL" -o /dev/null; then
      log "success notice sent (token from ${TOKEN_SRC})"
      exit 0
    fi
    # Loud, not fatal — see the header. A rejection here means the alarm is dead
    # too, and the preflight in this same run is what says so at ::error:: level.
    log "::error::the deploy-success Pushover notice was REJECTED (token from ${TOKEN_SRC})."
    summary "🚨 **Pushover rejected the deploy-success notice** — the failure alarm is dead too (ops #2480)."
    exit 0
    ;;

  *)
    log "usage: $0 [preflight|notify|notify-success]"
    exit 2
    ;;
esac
