#!/bin/bash
# BirdNET-Go Health Check Script v7
#
# Consolidates health check v6 + boya-reset.sh into a single script.
# Changes from v6:
# - Merged BOYA soft desync detection and USB recovery (from boya-reset.sh)
# - Added post-recovery MQTT verification (30s wait, re-check)
# - Added Pushover alerting for hard desync (unrecoverable)
# - Added 2-hour cooldown after failed soft recovery
# - Reads Pushover creds from /etc/default/birdnet-health-check
#
# Deploy to: /usr/local/bin/birdnet-health-check.sh
# Schedule: systemd timer every 15 minutes
# Requires: mosquitto_sub (mosquitto-clients), python3, curl

set -euo pipefail

PATH="/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

# ============================================================================
# CONFIGURATION
# ============================================================================

PROCESS_NAME="birdnet-go"
SERVICE_NAME="birdnet-go-native.service"
LOG_FILE="/var/log/birdnet-health-check.log"
CLIPS_DIR="/root/birdnet-go-app/data/clips"

# MQTT
MQTT_BROKER="192.168.1.148"
MQTT_USER="birdnet"
MQTT_PASS="secret"
MQTT_TOPIC="birdnet/soundlevel"
MQTT_TIMEOUT=20

# USB
USB_PORT="1-1.2"

# Desync cooldown
COOLDOWN_FILE="/tmp/boya-desync-cooldown"
COOLDOWN_SECONDS=7200  # 2 hours

# Pushover (loaded from env file)
PUSHOVER_USER_KEY="${PUSHOVER_USER_KEY:-}"
PUSHOVER_API_TOKEN="${PUSHOVER_API_TOKEN:-}"

# ============================================================================
# LOAD ENVIRONMENT
# ============================================================================

ENV_FILE="/etc/default/birdnet-health-check"
if [[ -f "$ENV_FILE" ]]; then
    # shellcheck source=/dev/null
    source "$ENV_FILE"
fi

# ============================================================================
# CONCURRENT EXECUTION GUARD
# ============================================================================

LOCKFILE="/var/run/birdnet-health-check.lock"
exec 9>"$LOCKFILE"
if ! flock -n 9; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') - INFO: Another instance running — exiting" >> "$LOG_FILE"
    exit 0
fi

# ============================================================================
# HELPERS
# ============================================================================

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" >> "$LOG_FILE"
}

get_all_pids() {
    pgrep -f "${PROCESS_NAME} realtime" 2>/dev/null || true
}

get_pid() {
    pgrep -f "${PROCESS_NAME} realtime" 2>/dev/null | head -1
}

get_pid_count() {
    pgrep -f "${PROCESS_NAME} realtime" 2>/dev/null | wc -l
}

kill_all_birdnet() {
    local pids
    pids=$(get_all_pids)
    if [[ -n "$pids" ]]; then
        log "INFO: Killing all BirdNET-Go processes: $pids"
        for pid in $pids; do
            kill -9 "$pid" 2>/dev/null || true
        done
        sleep 2
    fi
}

# ============================================================================
# HEALTH CHECKS
# ============================================================================

check_process_running() {
    local pid
    pid=$(get_pid)
    if [[ -z "$pid" ]]; then
        log "ERROR: BirdNET-Go process not running!"
        return 1
    fi
    return 0
}

check_duplicate_processes() {
    local count
    count=$(get_pid_count)
    if [[ $count -gt 1 ]]; then
        log "ERROR: Multiple BirdNET-Go processes detected ($count instances)"
        return 1
    fi
    return 0
}

check_stale_audio_fd() {
    local pid
    pid=$(get_pid)
    if [[ -z "$pid" ]]; then
        return 1
    fi

    if ls -la /proc/"$pid"/fd 2>/dev/null | grep -q 'snd.*deleted'; then
        log "ERROR: Stale audio device detected (deleted handle) for PID $pid"
        return 1
    fi

    if ! ls -la /proc/"$pid"/fd 2>/dev/null | grep -q '/dev/snd/pcm'; then
        log "ERROR: No audio capture device open for PID $pid"
        return 1
    fi

    return 0
}

# Read MQTT sound level message. Sets global MQTT_MSG on success.
# Returns 0 if a recent message was received, 1 otherwise.
read_mqtt_sound_level() {
    MQTT_MSG=$(mosquitto_sub \
        -h "$MQTT_BROKER" \
        -u "$MQTT_USER" \
        -P "$MQTT_PASS" \
        -t "$MQTT_TOPIC" \
        -C 1 \
        -W "$MQTT_TIMEOUT" 2>/dev/null) || {
        log "WARNING: Could not read MQTT sound level (broker unreachable or timeout)"
        return 1
    }

    if [[ -z "$MQTT_MSG" ]]; then
        log "WARNING: Empty MQTT message"
        return 1
    fi

    # Check timestamp freshness (< 2 minutes)
    local ts_age
    ts_age=$(echo "$MQTT_MSG" | python3 -c "
import json, sys
from datetime import datetime
try:
    d = json.load(sys.stdin)
    ts = d.get('ts', '')
    dt = datetime.fromisoformat(ts)
    now = datetime.now(dt.tzinfo)
    print(int((now - dt).total_seconds()))
except Exception:
    print(-1)
" 2>/dev/null)

    if [[ "$ts_age" == "-1" ]]; then
        log "WARNING: Could not parse MQTT timestamp"
        return 1
    fi

    if [[ "$ts_age" -gt 120 ]]; then
        log "WARNING: MQTT sound level is stale (age: ${ts_age}s)"
        return 1
    fi

    log "INFO: Audio pipeline active (MQTT age: ${ts_age}s)"
    return 0
}

# Check if all MQTT frequency bands report -200 dB (silence / desync).
# Requires MQTT_MSG to be set by read_mqtt_sound_level().
# Returns 0 if ALL bands are -200 (desync detected), 1 if audio is present.
check_all_bands_silent() {
    local result
    result=$(echo "$MQTT_MSG" | python3 -c "
import json, sys
try:
    d = json.load(sys.stdin)
    bands = d.get('b', {})
    if not bands:
        print('NO_BANDS')
        sys.exit()
    all_silent = all(band.get('m', 0) <= -199 for band in bands.values())
    print('SILENT' if all_silent else 'AUDIO_OK')
except Exception:
    print('ERR')
" 2>/dev/null)

    case "$result" in
        SILENT)
            log "WARNING: All frequency bands at -200 dB — BOYA desync detected"
            return 0
            ;;
        AUDIO_OK)
            return 1
            ;;
        *)
            log "WARNING: Could not parse frequency bands"
            return 1
            ;;
    esac
}

# ============================================================================
# DESYNC RECOVERY
# ============================================================================

# Check if cooldown is active (failed recovery < 2 hours ago).
is_cooldown_active() {
    if [[ ! -f "$COOLDOWN_FILE" ]]; then
        return 1
    fi
    local mtime now age
    mtime=$(stat -c %Y "$COOLDOWN_FILE" 2>/dev/null) || return 1
    now=$(date +%s)
    age=$(( now - mtime ))
    if [[ $age -lt $COOLDOWN_SECONDS ]]; then
        local remaining=$(( (COOLDOWN_SECONDS - age) / 60 ))
        log "INFO: Desync cooldown active (${remaining}m remaining) — skipping USB recovery"
        return 0
    fi
    return 1
}

# Attempt USB unbind/rebind soft recovery.
# Returns 0 if audio recovered, 1 if still silent (hard desync).
attempt_soft_recovery() {
    log "INFO: Attempting BOYA soft desync recovery via USB unbind/rebind"

    # Stop service
    log "INFO: Stopping $SERVICE_NAME"
    systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    sleep 1

    # Kill orphaned processes (USB rebind can spawn extras via systemd)
    kill_all_birdnet

    # USB unbind
    log "INFO: Unbinding USB port $USB_PORT"
    if ! echo "$USB_PORT" > /sys/bus/usb/drivers/usb/unbind 2>/dev/null; then
        log "WARNING: USB unbind failed"
    fi
    sleep 5

    # USB rebind
    log "INFO: Rebinding USB port $USB_PORT"
    if ! echo "$USB_PORT" > /sys/bus/usb/drivers/usb/bind 2>/dev/null; then
        log "WARNING: USB rebind failed"
    fi
    sleep 3

    # Kill any processes spawned by USB rebind (systemd udev trigger)
    kill_all_birdnet

    # Start service
    log "INFO: Starting $SERVICE_NAME"
    systemctl start "$SERVICE_NAME" 2>/dev/null || {
        log "ERROR: Failed to start $SERVICE_NAME after USB reset"
        return 1
    }

    # Wait for audio pipeline to initialize and publish MQTT
    log "INFO: Waiting 30s for audio pipeline to stabilize..."
    sleep 30

    # Re-check MQTT
    if ! read_mqtt_sound_level; then
        log "ERROR: MQTT not active after soft recovery"
        return 1
    fi

    if check_all_bands_silent; then
        log "ERROR: Still -200 dB after USB rebind — hard desync confirmed"
        return 1
    fi

    log "INFO: Soft recovery successful — audio restored"
    return 0
}

# ============================================================================
# PUSHOVER ALERTING
# ============================================================================

send_pushover_alert() {
    local message="$1"

    if [[ -z "$PUSHOVER_USER_KEY" || -z "$PUSHOVER_API_TOKEN" ]]; then
        log "WARNING: Pushover credentials not configured — cannot send alert"
        return 1
    fi

    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        --form-string "token=$PUSHOVER_API_TOKEN" \
        --form-string "user=$PUSHOVER_USER_KEY" \
        --form-string "title=BirdNET BOYA Hard Desync" \
        --form-string "message=$message" \
        --form-string "priority=0" \
        https://api.pushover.net/1/messages.json 2>/dev/null) || {
        log "WARNING: Pushover request failed"
        return 1
    }

    if [[ "$response" == "200" ]]; then
        log "INFO: Pushover alert sent"
    else
        log "WARNING: Pushover returned HTTP $response"
    fi
}

# ============================================================================
# SERVICE RESTART
# ============================================================================

restart_birdnet() {
    log "INFO: Restarting BirdNET-Go..."

    systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    sleep 2
    kill_all_birdnet

    if [[ $(get_pid_count) -gt 0 ]]; then
        log "WARNING: Processes still running after kill, forcing..."
        kill_all_birdnet
        sleep 2
    fi

    if systemctl start "$SERVICE_NAME" 2>/dev/null; then
        sleep 5
        local new_pid count
        new_pid=$(get_pid)
        count=$(get_pid_count)
        if [[ -n "$new_pid" && $count -eq 1 ]]; then
            log "INFO: Successfully restarted — PID: $new_pid (single instance)"
            return 0
        elif [[ $count -gt 1 ]]; then
            log "ERROR: Multiple processes after restart ($count), cleaning up..."
            local first_pid
            first_pid=$(get_pid)
            for pid in $(get_all_pids); do
                if [[ "$pid" != "$first_pid" ]]; then
                    kill -9 "$pid" 2>/dev/null || true
                fi
            done
            return 0
        fi
    fi

    log "ERROR: Failed to restart BirdNET-Go properly"
    return 1
}

# ============================================================================
# DETECTION STATS (informational only)
# ============================================================================

report_detection_stats() {
    local hour
    hour=$(date +%H)

    local clips_1h=0 clips_24h=0
    if [[ -d "$CLIPS_DIR" ]]; then
        clips_1h=$(find "$CLIPS_DIR" -name "*.wav" -mmin -60 2>/dev/null | wc -l)
        clips_24h=$(find "$CLIPS_DIR" -name "*.wav" -mmin -1440 2>/dev/null | wc -l)
    fi

    log "INFO: Detection stats — Last hour: $clips_1h clips, Last 24h: $clips_24h clips"

    if [[ 10#$hour -ge 6 && 10#$hour -le 10 && $clips_1h -eq 0 ]]; then
        log "INFO: No detections during morning peak hours (may be normal)"
    fi
}

# ============================================================================
# MAIN
# ============================================================================

main() {
    log "INFO: Starting BirdNET-Go health check v7"

    local needs_restart=false
    local restart_reason=""

    # Check 1: Process running?
    if ! check_process_running; then
        needs_restart=true
        restart_reason="process not running"
    fi

    # Check 2: Duplicate processes?
    if ! check_duplicate_processes; then
        needs_restart=true
        restart_reason="duplicate processes"
    fi

    # Check 3: Stale audio file descriptor?
    if [[ "$needs_restart" == "false" ]]; then
        if ! check_stale_audio_fd; then
            needs_restart=true
            restart_reason="stale audio device"
        fi
    fi

    # Check 4: MQTT pipeline active?
    # Note: Hard desync prevents BirdNET-Go from publishing fresh MQTT
    # (ALSA I/O errors → no sound level data → stale MQTT). If MQTT is stale
    # but the last message shows all bands at -200 dB, route to desync recovery
    # instead of a normal restart (which would loop forever).
    local mqtt_stale=false
    if [[ "$needs_restart" == "false" ]]; then
        if ! read_mqtt_sound_level; then
            mqtt_stale=true
            # Check if stale message indicates desync (-200 dB on all bands)
            if [[ -n "${MQTT_MSG:-}" ]] && check_all_bands_silent; then
                log "INFO: MQTT stale with all bands at -200 dB — routing to desync recovery"
            else
                needs_restart=true
                restart_reason="audio pipeline not active (no recent MQTT sound level)"
            fi
        fi
    fi

    # Check 5: BOYA desync? (fresh MQTT with all bands silent, or stale MQTT with -200 dB)
    if { [[ "$needs_restart" == "false" ]] && check_all_bands_silent; } || \
       { [[ "$mqtt_stale" == "true" && "$needs_restart" == "false" ]]; }; then
        if is_cooldown_active; then
            log "INFO: Desync persists but cooldown active — skipping recovery"
        else
            if attempt_soft_recovery; then
                log "INFO: BOYA soft desync recovered successfully"
                # Remove cooldown file if it exists (recovery worked)
                rm -f "$COOLDOWN_FILE"
            else
                # Hard desync — set cooldown, send alert
                touch "$COOLDOWN_FILE"
                local cooldown_until
                cooldown_until=$(date -d "+${COOLDOWN_SECONDS} seconds" '+%H:%M')
                send_pushover_alert "Soft recovery failed (USB rebind). Manual re-pair needed. Cooldown until ${cooldown_until}."
                log "INFO: Cooldown set — next recovery attempt after ${cooldown_until}"
            fi
        fi
        # Skip normal restart — desync path handled it
        needs_restart=false
    fi

    # Always report detection stats
    report_detection_stats

    # Normal restart if needed (non-desync issues)
    if [[ "$needs_restart" == "true" ]]; then
        log "INFO: Restart triggered due to: $restart_reason"
        restart_birdnet
        sleep 5
        if check_process_running && check_stale_audio_fd && check_duplicate_processes; then
            log "INFO: Restart successful, system healthy"
        else
            log "ERROR: Issues remain after restart"
        fi
    else
        log "INFO: All health checks passed"
    fi

    log "INFO: Health check complete"
}

main "$@"
