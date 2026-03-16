#!/bin/bash
# BirdNET-Go Health Check Script v5
# Changes from v4:
# - Check inference activity (not just clips) to distinguish "no birds" from "system broken"
# - Only restart if inference has stopped, not just because no birds detected
# - Add audio level sanity check
# - Better logging with distinct failure modes

set -euo pipefail

CLIPS_DIR="/root/birdnet-go-app/data/clips"
ANALYSIS_LOG="/home/jeremy/logs/analysis-processor.log"
MAX_INFERENCE_GAP_MINUTES=5  # If no inference for 5 min, something is wrong
PROCESS_NAME="birdnet-go"
LOG_FILE="/var/log/birdnet-health-check.log"
SERVICE_NAME="birdnet-go-native.service"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$LOG_FILE"
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
    local pids=$(get_all_pids)
    if [[ -n "$pids" ]]; then
        log "INFO: Killing all BirdNET-Go processes: $pids"
        for pid in $pids; do
            kill -9 "$pid" 2>/dev/null || true
        done
        sleep 2
    fi
}

check_duplicate_processes() {
    local count=$(get_pid_count)
    if [[ $count -gt 1 ]]; then
        log "ERROR: Multiple BirdNET-Go processes detected ($count instances)"
        return 1
    fi
    return 0
}

check_process_running() {
    local pid=$(get_pid)
    if [[ -z "$pid" ]]; then
        log "ERROR: BirdNET-Go process not running!"
        return 1
    fi
    return 0
}

check_stale_audio_fd() {
    local pid=$(get_pid)
    if [[ -z "$pid" ]]; then
        return 1  # Already handled by check_process_running
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

# NEW: Check if inference engine is actually running by looking at analysis-processor.log
check_inference_activity() {
    if [[ ! -f "$ANALYSIS_LOG" ]]; then
        log "WARNING: Analysis processor log not found: $ANALYSIS_LOG"
        return 1
    fi

    # Look for recent "Detection processing results" entries
    local recent_inference=$(find "$ANALYSIS_LOG" -mmin -${MAX_INFERENCE_GAP_MINUTES} 2>/dev/null | wc -l)
    
    if [[ $recent_inference -eq 0 ]]; then
        log "WARNING: Analysis log not updated in ${MAX_INFERENCE_GAP_MINUTES} minutes"
        # Double-check by looking at actual log content
        local last_entry=$(tail -1 "$ANALYSIS_LOG" 2>/dev/null | grep -o '"time":"[^"]*"' | head -1)
        log "WARNING: Last log entry timestamp: $last_entry"
        return 1
    fi

    # Check if we're seeing inference results (not just startup messages)
    local inference_count=$(tail -20 "$ANALYSIS_LOG" 2>/dev/null | grep -c "process_detections_summary" || echo 0)
    
    if [[ $inference_count -gt 0 ]]; then
        log "INFO: Inference engine active ($inference_count recent processing cycles)"
        return 0
    else
        log "WARNING: No recent inference activity in log"
        return 1
    fi
}

# NEW: Report detection statistics without triggering restart
report_detection_stats() {
    local hour=$(date +%H)
    
    # Count recent clips
    local clips_1h=0
    local clips_24h=0
    
    if [[ -d "$CLIPS_DIR" ]]; then
        clips_1h=$(find "$CLIPS_DIR" -name "*.wav" -mmin -60 2>/dev/null | wc -l)
        clips_24h=$(find "$CLIPS_DIR" -name "*.wav" -mmin -1440 2>/dev/null | wc -l)
    fi
    
    log "INFO: Detection stats - Last hour: $clips_1h clips, Last 24h: $clips_24h clips"
    
    # Informational warning if no detections during peak hours
    if [[ 10#$hour -ge 6 && 10#$hour -le 10 && $clips_1h -eq 0 ]]; then
        log "INFO: No detections during morning peak hours (may be normal)"
    fi
}

restart_birdnet() {
    log "INFO: Restarting BirdNET-Go..."

    # Step 1: Stop systemd service
    log "INFO: Stopping systemd service..."
    systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    sleep 2

    # Step 2: Kill ALL remaining processes
    kill_all_birdnet

    # Step 3: Verify all processes are dead
    if [[ $(get_pid_count) -gt 0 ]]; then
        log "WARNING: Processes still running after kill, forcing..."
        kill_all_birdnet
        sleep 2
    fi

    # Step 4: Start fresh via systemd
    log "INFO: Starting systemd service..."
    if systemctl start "$SERVICE_NAME" 2>/dev/null; then
        sleep 5
        local new_pid=$(get_pid)
        local count=$(get_pid_count)
        if [[ -n "$new_pid" && $count -eq 1 ]]; then
            log "INFO: Successfully restarted - PID: $new_pid (single instance)"
            return 0
        elif [[ $count -gt 1 ]]; then
            log "ERROR: Multiple processes after restart ($count), cleaning up..."
            local first_pid=$(get_pid)
            for pid in $(get_all_pids); do
                if [[ "$pid" != "$first_pid" ]]; then
                    kill -9 "$pid" 2>/dev/null || true
                fi
            done
            return 0
        fi
    fi

    # Fallback: manual start
    log "WARNING: systemctl failed, trying manual start"
    cd /home/jeremy
    nohup /usr/local/bin/birdnet-go realtime > /tmp/birdnet-go.log 2>&1 &
    sleep 5

    if [[ $(get_pid_count) -eq 1 ]]; then
        log "INFO: BirdNET-Go started manually - PID: $(get_pid)"
        return 0
    else
        log "ERROR: Failed to restart BirdNET-Go properly"
        return 1
    fi
}

main() {
    log "INFO: Starting BirdNET-Go health check v5"

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

    # Check 4: Is inference actually running? (KEY IMPROVEMENT)
    # This catches the "audio thread stuck" case from Jan 5
    if [[ "$needs_restart" == "false" ]]; then
        if ! check_inference_activity; then
            needs_restart=true
            restart_reason="inference engine not active"
        fi
    fi

    # Always report detection stats (informational only)
    report_detection_stats

    # Restart if needed
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
