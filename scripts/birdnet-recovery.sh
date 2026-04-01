#!/bin/bash
# BirdNET-Go Recovery Script
# Checks /health endpoint and performs USB recovery if needed.
# Deploy to: /usr/local/bin/birdnet-recovery.sh
# Schedule: systemd timer every 15 minutes

set -euo pipefail

LOG="/var/log/birdnet-recovery.log"
COOLDOWN_FILE="/tmp/birdnet-recovery-cooldown"
COOLDOWN_SECONDS=7200
USB_HUB="1-1"
USB_HUB_PORT="2"
SERVICE="birdnet-go-native.service"

log() { echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" >> "$LOG"; }

# Load Pushover credentials
[[ -f /etc/default/birdnet-health-check ]] && source /etc/default/birdnet-health-check

# Concurrent execution guard
LOCKFILE="/var/run/birdnet-recovery.lock"
exec 9>"$LOCKFILE"
flock -n 9 || { log "INFO: Another instance running"; exit 0; }

# Check cooldown
if [[ -f "$COOLDOWN_FILE" ]]; then
    cooldown_time=$(cat "$COOLDOWN_FILE")
    now=$(date +%s)
    if (( now - cooldown_time < COOLDOWN_SECONDS )); then
        log "INFO: In cooldown period, skipping"
        exit 0
    fi
    rm -f "$COOLDOWN_FILE"
fi

# Query health endpoint
health=$(curl -sf --max-time 5 http://localhost:8080/api/v2/health 2>/dev/null) || {
    log "ERROR: Health endpoint unreachable — restarting service"
    sudo systemctl restart "$SERVICE"
    exit 0
}

status=$(echo "$health" | python3 -c "import sys,json; print(json.load(sys.stdin)['status'])" 2>/dev/null || echo "unknown")

if [[ "$status" == "healthy" ]]; then
    exit 0
fi

log "WARN: Status=$status — attempting USB recovery"

# Stop service and kill orphans
sudo systemctl stop "$SERVICE"
sleep 2
sudo kill -9 $(pgrep -f 'birdnet-go realtime') 2>/dev/null || true
sleep 1

# USB power cycle via uhubctl (cuts electrical power, more effective than unbind/rebind)
sudo uhubctl -l "$USB_HUB" -p "$USB_HUB_PORT" -a off > /dev/null 2>&1 || true
sleep 10
sudo uhubctl -l "$USB_HUB" -p "$USB_HUB_PORT" -a on > /dev/null 2>&1 || true
sleep 5

# Restart service
sudo systemctl start "$SERVICE"
sleep 30

# Re-check
health=$(curl -sf --max-time 5 http://localhost:8080/api/v2/health 2>/dev/null) || health=""
status=$(echo "$health" | python3 -c "import sys,json; print(json.load(sys.stdin)['status'])" 2>/dev/null || echo "unknown")

if [[ "$status" != "healthy" ]]; then
    log "ERROR: Still unhealthy after recovery — hard desync, needs physical intervention"
    date +%s > "$COOLDOWN_FILE"
    # Send Pushover alert
    if [[ -n "${PUSHOVER_USER_KEY:-}" && -n "${PUSHOVER_API_TOKEN:-}" ]]; then
        curl -sf -X POST https://api.pushover.net/1/messages.json \
            -d "token=$PUSHOVER_API_TOKEN&user=$PUSHOVER_USER_KEY&title=BirdNET Hard Desync&message=BOYA mic unrecoverable. Needs physical re-pair.&priority=1" \
            > /dev/null 2>&1 || true
    fi
else
    log "INFO: Recovery successful"
fi
