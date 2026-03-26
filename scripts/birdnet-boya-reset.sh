#!/usr/bin/env bash
# birdnet-boya-reset.sh — BOYA wireless mic soft desync recovery
#
# Detects zero-amplitude (soft desync) via MQTT sound levels and attempts
# USB unbind/rebind recovery. Runs from cron every 15 minutes.
#
# Two BOYA failure modes:
#   Soft desync: zero amplitude (-200 dB), USB unbind/rebind fixes it
#   Hard desync: I/O error from arecord, needs physical power cycle (4AM reboot handles overnight)
#
# Requires: mosquitto_sub (mosquitto-clients), python3

set -euo pipefail

PATH="/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

# --- Concurrent execution guard ---
LOCKFILE="/var/run/boya-reset.lock"
exec 9>"$LOCKFILE"
if ! flock -n 9; then
    logger -t "boya-reset" "another instance is running — exiting"
    exit 0
fi

# --- Configuration ---
USB_PORT="1-1.2"
MQTT_BROKER="192.168.1.148"
MQTT_USER="birdnet"
MQTT_PASS="secret"
MQTT_TOPIC="birdnet/soundlevel"
SILENCE_THRESHOLD=-97   # integer dB — below this means silence/desync
SERVICE="birdnet-go-native.service"
DAYTIME_START=5          # 5 AM
DAYTIME_END=22           # 10 PM
MQTT_TIMEOUT=20          # seconds to wait for an MQTT message

TAG="boya-reset"

log() { logger -t "$TAG" "$*"; }

# --- Guard: only run during daytime hours ---
HOUR=$(date +%-H)
if (( HOUR < DAYTIME_START || HOUR >= DAYTIME_END )); then
    exit 0
fi

# --- Guard: birdnet-go service must be running ---
if ! systemctl is-active --quiet "$SERVICE"; then
    log "service $SERVICE is not running — skipping check"
    exit 0
fi

# --- Check: read latest MQTT sound level ---
# Note: We cannot use arecord as a smoke test because BirdNET-Go holds
# the ALSA device exclusively. MQTT sound level is our only indicator.
# Soft desync shows as -200 dB; hard desync also shows silence but won't
# recover from USB reset — the 4AM daily reboot handles those.
# Get exactly one message with a timeout. If broker is unreachable or topic
# has no retained message within the timeout, exit safely (don't reset unnecessarily).
MQTT_MSG=$(mosquitto_sub \
    -h "$MQTT_BROKER" \
    -u "$MQTT_USER" \
    -P "$MQTT_PASS" \
    -t "$MQTT_TOPIC" \
    -C 1 \
    -W "$MQTT_TIMEOUT" 2>/dev/null) || {
    log "WARNING: could not read MQTT sound level (broker unreachable or timeout) — skipping"
    exit 0
}

if [[ -z "$MQTT_MSG" ]]; then
    log "WARNING: empty MQTT message — skipping"
    exit 0
fi

# Extract 1.0 kHz band mean dB from JSON payload and convert to integer
# Payload structure: {"b": {"1.0_kHz": {"m": -85.3, ...}, ...}, ...}
read -r DB_FLOAT DB_INT < <(echo "$MQTT_MSG" | python3 -c "
import json, sys
try:
    d = json.load(sys.stdin)
    val = d['b']['1.0_kHz']['m']
    print(f'{val} {int(val)}')
except (json.JSONDecodeError, KeyError, TypeError, ValueError):
    print('ERR ERR')
")

if [[ "$DB_INT" == "ERR" ]]; then
    log "WARNING: could not parse sound level from MQTT payload — skipping"
    exit 0
fi

# --- Decision: is this silence? ---
if (( DB_INT >= SILENCE_THRESHOLD )); then
    # Audio is fine, nothing to do
    exit 0
fi

# --- Silence detected — attempt USB unbind/rebind recovery ---
log "silence detected: ${DB_FLOAT} dB (threshold: ${SILENCE_THRESHOLD} dB) — attempting USB reset"

# Stop the service first
log "stopping $SERVICE"
systemctl stop "$SERVICE" 2>/dev/null || true

# Kill orphaned birdnet-go processes that may hold /dev/snd/pcmC2D0c
# (systemd udev trigger can spawn extras via Wants=sys-subsystem-sound-devices-card2.device)
sleep 1
if pgrep -f 'birdnet-go realtime' >/dev/null 2>&1; then
    log "killing orphaned birdnet-go processes"
    pkill -9 -f 'birdnet-go realtime' 2>/dev/null || true
    sleep 1
fi

# USB unbind
log "unbinding USB port $USB_PORT"
if echo "$USB_PORT" > /sys/bus/usb/drivers/usb/unbind 2>/dev/null; then
    sleep 5

    # USB bind
    log "binding USB port $USB_PORT"
    if echo "$USB_PORT" > /sys/bus/usb/drivers/usb/bind 2>/dev/null; then
        sleep 3
        log "USB rebind complete"
    else
        log "WARNING: USB bind failed"
    fi
else
    log "WARNING: USB unbind failed"
fi

# Always restart the service, even if USB reset had issues
log "starting $SERVICE"
systemctl start "$SERVICE" 2>/dev/null || {
    log "ERROR: failed to start $SERVICE after reset"
    exit 1
}

log "reset complete — service restarted"
exit 0
