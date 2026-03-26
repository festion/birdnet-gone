# BirdNET-Go Simplification — Baseline Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Strip the BirdNET Pi to a stable, minimal baseline — core detection + MQTT + built-in web UI — then selectively re-add integrations only after proving stability.

**Architecture:** The Pi (192.168.1.197) currently runs 4 custom services. We reduce to 1 (birdnet-go-native) + a simple shell-based BOYA watchdog. The Flask display, VicoHome bridge, and Python watchdog are disabled (not deleted). BirdNET-Go config is simplified to remove features that add failure surface without core value. The kiosk display points at BirdNET-Go's built-in web UI (port 8080) instead of the custom Flask app.

**Tech Stack:** Shell scripts (systemd, cron), BirdNET-Go config (YAML), Chromium kiosk

**Constraints (from user):**
- MQTT stays — used throughout the homelab, not just BirdNET
- BOYA mic stays — hardware is not changing
- Fork stays — upstream hasn't fixed 3 critical MQTT patches (verified against nightly-20260322)
- 4AM daily reboot stays — already configured in `/etc/cron.d/daily-reboot`

---

## Task 1: Disable Non-Essential Services

**Target:** BirdNET Pi (192.168.1.197) via SSH

**Why:** Remove 3 of 4 custom services. The Flask display exposed unauthenticated `sudo reboot` / `sudo poweroff` endpoints and caused an unscheduled reboot today at 07:15 via `POST /reboot` from localhost. The Python watchdog can only alert, not remediate. The VicoHome bridge adds cloud API dependency for supplementary camera data.

**Step 1: Stop and disable bird-display.service**

```bash
ssh jeremy@192.168.1.197 "sudo systemctl stop bird-display.service && sudo systemctl disable bird-display.service"
```

Expected: Service stops, no longer starts on boot. Files remain at `/home/jeremy/birdnet_display/`.

**Step 2: Stop and disable vicohome-bridge.service**

```bash
ssh jeremy@192.168.1.197 "sudo systemctl stop vicohome-bridge.service && sudo systemctl disable vicohome-bridge.service"
```

Expected: Service stops. Files remain at `/home/jeremy/vicohome-bridge/`.

**Step 3: Stop and disable birdnet-watchdog.service**

```bash
ssh jeremy@192.168.1.197 "sudo systemctl stop birdnet-watchdog.service && sudo systemctl disable birdnet-watchdog.service"
```

Expected: Service stops. Files remain at `/home/jeremy/birdnet-watchdog/`.

**Step 4: Verify only birdnet-go-native remains**

```bash
ssh jeremy@192.168.1.197 "systemctl list-units --type=service --state=running | grep -E 'birdnet|bird-display|vicohome|watchdog'"
```

Expected: Only `birdnet-go-native.service` appears.

**Step 5: Verify BirdNET-Go is still detecting**

```bash
ssh jeremy@192.168.1.197 "journalctl -u birdnet-go-native.service --since '5 min ago' --no-pager | grep -c 'Published sound level data'"
```

Expected: Non-zero count (sound level publishes every ~10-30s when healthy).

---

## Task 2: Simplify BirdNET-Go Config

**Target:** `/home/jeremy/.config/birdnet-go/config.yaml` on Pi

**Why:** The current config enables BirdWeather, weather, dynamic threshold, dog bark filter, monitoring, and debug mode. Each adds code paths and external dependencies. For a stable baseline, only detection + MQTT + clip export matter.

**Step 1: Backup current config**

```bash
ssh jeremy@192.168.1.197 "cp /home/jeremy/.config/birdnet-go/config.yaml /home/jeremy/.config/birdnet-go/config.yaml.pre-simplification"
```

**Step 2: Disable BirdWeather**

In `config.yaml`, set:
```yaml
realtime:
  birdweather:
    enabled: false
```

BirdWeather uploads detections to a community platform. Not needed for baseline stability. Can re-enable later.

**Step 3: Disable weather integration**

```yaml
realtime:
  weather:
    provider: ""
```

Weather correlation is cosmetic. Removes yr.no API dependency.

**Step 4: Disable dynamic threshold**

```yaml
realtime:
  dynamicthreshold:
    enabled: false
```

Dynamic threshold adjusts confidence based on recent detection patterns. Simplifies detection behavior — the static threshold (0.65) and per-species thresholds are sufficient.

**Step 5: Disable dog bark filter**

```yaml
realtime:
  dogbarkfilter:
    enabled: false
```

Privacy/noise filter. Not needed for a backyard mic in a residential area. Removes a processing step.

**Step 6: Disable monitoring telemetry**

```yaml
realtime:
  monitoring:
    enabled: false
```

CPU/mem/disk monitoring within BirdNET-Go. Redundant with Fluent Bit + Grafana already monitoring the Pi.

**Step 7: Set debug to false**

```yaml
webserver:
  debug: false
```

Debug mode adds verbose logging overhead. Not needed for baseline operation.

**Step 8: Keep these settings unchanged**

- `realtime.mqtt` — enabled, HA discovery, sound level (homelab MQTT integration)
- `realtime.audio` — BOYA Magic USB Audio source, equalizer, sound level
- `birdnet` — sensitivity 1, threshold 0.65, overlap 2.4, range filter 0.01
- `realtime.audio.export` — clip export enabled (WAV, retention policy)
- Species-specific thresholds (11 species) — these are well-tuned
- `realtime.privacyfilter` — keep enabled (voice detection is a good default)

**Step 9: Restart BirdNET-Go and verify**

```bash
ssh jeremy@192.168.1.197 "sudo systemctl restart birdnet-go-native.service && sleep 10 && journalctl -u birdnet-go-native.service --since '30 sec ago' --no-pager"
```

Expected: Clean startup logs, no errors. Sound level publishing resumes within 30s.

**Step 10: Verify MQTT still publishing**

```bash
ssh jeremy@192.168.1.197 "timeout 15 mosquitto_sub -h 192.168.1.148 -u birdnet -P secret -t 'birdnet/soundlevel' -C 1 2>/dev/null && echo 'MQTT OK' || echo 'MQTT FAIL'"
```

Expected: Receives one sound level message within 15s. If `mosquitto_sub` isn't installed, check via HA or MQTT broker directly.

---

## Task 3: Replace Python Watchdog with Shell BOYA Reset Script

**Target:** Create `/home/jeremy/birdnet-boya-reset.sh` on Pi, add cron job

**Why:** The Python watchdog (166 LOC, paho-mqtt dependency, venv) can only alert via webhook. A shell script can detect silence AND attempt USB rebind recovery. For hard desyncs (I/O error), the 4AM daily reboot is the recovery mechanism.

**Step 1: Create the reset script in the repo**

Create file: `scripts/birdnet-boya-reset.sh`

```bash
#!/bin/bash
# BOYA Magic wireless mic soft desync recovery
# Runs via cron every 15 minutes. Checks if audio is silent (soft desync)
# and attempts USB unbind/rebind to recover.
#
# Hard desyncs (I/O error) require physical power cycle of the BOYA
# transmitter — the daily 4AM reboot in /etc/cron.d/daily-reboot
# is the recovery mechanism for overnight hard desyncs.
#
# USB port: 1-1.2 (BOYA USB receiver on RPi)
# Silence threshold: -97 dB (normal ambient is -80 to -90 dB)

set -euo pipefail

USB_PORT="1-1.2"
SOUND_CARD="hw:2,0"
SERVICE="birdnet-go-native.service"
LOG_TAG="boya-reset"
SILENCE_THRESHOLD="-97"

log() { logger -t "$LOG_TAG" "$1"; }

# Only run during hours when silence is actionable (not overnight)
HOUR=$(date +%H)
if (( HOUR < 5 || HOUR > 22 )); then
    exit 0
fi

# Check if BirdNET service is running
if ! systemctl is-active --quiet "$SERVICE"; then
    log "Service not running, skipping"
    exit 0
fi

# Quick audio test — record 2 seconds, check for signal
AUDIO_CHECK=$(arecord -D "$SOUND_CARD" -d 2 -f S24_3LE -r 48000 -c 2 /dev/null 2>&1) || {
    # I/O error = hard desync, USB reset won't help
    if echo "$AUDIO_CHECK" | grep -q "Input/output error"; then
        log "HARD DESYNC: I/O error on $SOUND_CARD — needs physical power cycle"
        exit 1
    fi
    log "arecord error: $AUDIO_CHECK"
    exit 1
}

# Check sound level from MQTT (last published value)
# If we can't check MQTT, skip — don't reset unnecessarily
LEVEL=$(timeout 20 mosquitto_sub -h 192.168.1.148 -u birdnet -P secret \
    -t 'birdnet/soundlevel' -C 1 2>/dev/null | \
    python3 -c "import sys,json; d=json.load(sys.stdin); print(f\"{d['b']['1.0_kHz']['m']:.1f}\")" 2>/dev/null) || {
    log "Could not read MQTT sound level, skipping"
    exit 0
}

# Compare as integers (bash can't do float comparison)
LEVEL_INT=${LEVEL%.*}
if (( LEVEL_INT > SILENCE_THRESHOLD )); then
    exit 0  # Audio is fine
fi

log "Silence detected (${LEVEL} dB < ${SILENCE_THRESHOLD} dB), attempting USB reset"

# Kill orphan birdnet-go processes FIRST (they hold /dev/snd/pcmC2D0c)
systemctl stop "$SERVICE"
sleep 2
pkill -9 -x birdnet-go 2>/dev/null || true
sleep 1

# USB unbind/rebind
echo "$USB_PORT" > /sys/bus/usb/drivers/usb/unbind 2>/dev/null || {
    log "USB unbind failed for $USB_PORT"
    systemctl start "$SERVICE"
    exit 1
}
sleep 5
echo "$USB_PORT" > /sys/bus/usb/drivers/usb/bind 2>/dev/null || {
    log "USB bind failed for $USB_PORT"
    systemctl start "$SERVICE"
    exit 1
}
sleep 3

# Restart service
systemctl start "$SERVICE"
log "USB reset complete, service restarted"
```

**Step 2: Deploy to Pi**

```bash
scp scripts/birdnet-boya-reset.sh jeremy@192.168.1.197:/home/jeremy/birdnet-boya-reset.sh
ssh jeremy@192.168.1.197 "chmod +x /home/jeremy/birdnet-boya-reset.sh"
```

**Step 3: Add cron job (runs as root for USB access)**

```bash
ssh jeremy@192.168.1.197 "sudo tee /etc/cron.d/boya-reset << 'EOF'
# BOYA wireless mic soft desync recovery — every 15 minutes
*/15 * * * * root /home/jeremy/birdnet-boya-reset.sh
EOF"
```

**Step 4: Verify cron is loaded**

```bash
ssh jeremy@192.168.1.197 "cat /etc/cron.d/boya-reset"
```

Expected: Shows the cron entry.

**Step 5: Verify mosquitto_sub is available on Pi**

```bash
ssh jeremy@192.168.1.197 "which mosquitto_sub || echo 'NOT INSTALLED'"
```

If not installed: `ssh jeremy@192.168.1.197 "sudo apt-get install -y mosquitto-clients"`

**Step 6: Test the script manually (dry run)**

```bash
ssh jeremy@192.168.1.197 "sudo /home/jeremy/birdnet-boya-reset.sh; echo 'Exit code:' \$?"
```

Expected: Either exits 0 silently (audio OK) or logs a reset attempt. Check: `ssh jeremy@192.168.1.197 "journalctl -t boya-reset --since '1 min ago' --no-pager"`

---

## Task 4: Point Kiosk at Built-in Web UI

**Target:** Chromium kiosk on Pi display

**Why:** BirdNET-Go has a built-in web UI at `http://localhost:8080` with detection history, spectrograms, species info, and live updates via SSE. This replaces the custom Flask display (32KB Python, 1,150-line HTML template, 20+ routes) with zero additional code.

**Step 1: Find the current kiosk launcher**

```bash
ssh jeremy@192.168.1.197 "cat /home/jeremy/birdnet_display/kiosk_launcher.sh"
```

**Step 2: Create a simple kiosk launcher pointing at built-in UI**

```bash
ssh jeremy@192.168.1.197 "cat > /home/jeremy/birdnet-kiosk.sh << 'KIOSK'
#!/bin/bash
# Simple kiosk launcher for BirdNET-Go built-in web UI
# Waits for BirdNET-Go to be ready, then opens Chromium in kiosk mode

# Wait for BirdNET-Go web UI
for i in {1..60}; do
    curl -sf http://localhost:8080/ > /dev/null 2>&1 && break
    sleep 2
done

exec chromium-browser \
    --kiosk \
    --noerrdialogs \
    --disable-infobars \
    --disable-session-crashed-bubble \
    --disable-features=TranslateUI \
    --check-for-update-interval=31536000 \
    http://localhost:8080
KIOSK
chmod +x /home/jeremy/birdnet-kiosk.sh"
```

**Step 3: Update the autostart desktop entry**

```bash
ssh jeremy@192.168.1.197 "cat /home/jeremy/.config/autostart/bird-display-kiosk.desktop"
```

Update the `Exec=` line to point at the new launcher:

```bash
ssh jeremy@192.168.1.197 "sed -i 's|Exec=.*|Exec=/home/jeremy/birdnet-kiosk.sh|' /home/jeremy/.config/autostart/bird-display-kiosk.desktop"
```

**Step 4: Verify boot mode is still Desktop auto-login (B4)**

```bash
ssh jeremy@192.168.1.197 "sudo raspi-config nonint get_boot_cli"
```

Expected: `1` (desktop mode). If `0` (console mode), fix with: `sudo raspi-config nonint do_boot_behaviour B4`

**Step 5: Test by killing current Chromium and relaunching**

```bash
ssh jeremy@192.168.1.197 "pkill chromium; sleep 2; nohup /home/jeremy/birdnet-kiosk.sh &"
```

Expected: Chromium opens in kiosk mode showing BirdNET-Go's built-in detection dashboard.

---

## Task 5: Clean Up Backup Binaries

**Target:** `/usr/local/bin/birdnet-go.*` on Pi

**Why:** 11 backup binaries consuming ~1.3 GB. Keep one known-good backup, delete the rest.

**Step 1: List all binaries with sizes**

```bash
ssh jeremy@192.168.1.197 "ls -lh /usr/local/bin/birdnet-go*"
```

**Step 2: Keep only the vanilla upstream backup, remove the rest**

```bash
ssh jeremy@192.168.1.197 "sudo rm \
    /usr/local/bin/birdnet-go.backup-20260221 \
    /usr/local/bin/birdnet-go.backup-custom \
    /usr/local/bin/birdnet-go.backup-pre-163 \
    /usr/local/bin/birdnet-go.backup-pre-buffix \
    /usr/local/bin/birdnet-go.bak.20260310 \
    /usr/local/bin/birdnet-go.debug-build \
    /usr/local/bin/birdnet-go.nightly-20251012 \
    /usr/local/bin/birdnet-go.nightly-20260113 \
    /usr/local/bin/birdnet-go.nightly-20260118-35 \
    /usr/local/bin/birdnet-go.pre-discovery-fix"
```

Keep: `birdnet-go` (active, 49MB) and `birdnet-go.nightly-20260118-vanilla` (last clean upstream, 122MB).

**Step 3: Verify**

```bash
ssh jeremy@192.168.1.197 "ls -lh /usr/local/bin/birdnet-go*"
```

Expected: Only 2 files remain (~171MB total, freeing ~1.1GB).

---

## Task 6: Remove Dead ESP32 References from Config

**Target:** BirdNET-Go config and health check

**Why:** The ESP32-C6 RTSP mic is no longer in use (confirmed by user). Any references to it in config, health checks, or services should be cleaned up to avoid confusion.

**Step 1: Check for ESP32 references in config**

```bash
ssh jeremy@192.168.1.197 "grep -i 'esp32\|rtsp\|10.42.0\|mediamtx' /home/jeremy/.config/birdnet-go/config.yaml"
```

**Step 2: Check display config for ESP32 references**

```bash
ssh jeremy@192.168.1.197 "cat /home/jeremy/birdnet_display/display_config.json"
```

If it contains ESP32 IP/port entries, these are now dead references. Since the display service is already disabled (Task 1), this is informational only — no action needed unless the display is re-enabled later.

**Step 3: Check if mediamtx service exists**

```bash
ssh jeremy@192.168.1.197 "systemctl status mediamtx.service 2>&1 | head -3"
```

If it exists and is running, stop and disable it:
```bash
ssh jeremy@192.168.1.197 "sudo systemctl stop mediamtx.service && sudo systemctl disable mediamtx.service"
```

**Step 4: Check for RTSP stream entries in BirdNET config**

If `realtime.rtsp.streams` has any entries, clear them:
```yaml
realtime:
  rtsp:
    transport: tcp
    streams: []
```

---

## Task 7: Verify Stable Baseline

**Target:** Full system verification after all changes

**Why:** Confirm the simplified system is operational before declaring baseline.

**Step 1: Check service status**

```bash
ssh jeremy@192.168.1.197 "systemctl list-units --type=service --state=running | grep -E 'birdnet|bird-display|vicohome|watchdog|mediamtx'"
```

Expected: Only `birdnet-go-native.service`.

**Step 2: Check disabled services are actually disabled**

```bash
ssh jeremy@192.168.1.197 "systemctl is-enabled bird-display.service vicohome-bridge.service birdnet-watchdog.service 2>&1"
```

Expected: All report `disabled`.

**Step 3: Verify detection pipeline**

```bash
ssh jeremy@192.168.1.197 "curl -sf http://localhost:8080/api/v2/analytics/species/summary 2>/dev/null | python3 -m json.tool | head -20"
```

Expected: JSON response with species detection data (confirms DB + API working).

**Step 4: Verify MQTT publishing**

```bash
ssh jeremy@192.168.1.197 "timeout 15 mosquitto_sub -h 192.168.1.148 -u birdnet -P secret -t 'birdnet/#' -C 3 -v 2>/dev/null"
```

Expected: Messages on `birdnet/soundlevel` and/or `birdnet/detection` topics.

**Step 5: Verify HA entities**

Check Home Assistant for BirdNET entities — confidence sensor, sound level sensor should still be updating. This can be done via the HA web UI or:

```bash
ssh root@homeassistant.local "ha core check 2>/dev/null; echo 'HA OK'"
```

**Step 6: Check cron jobs are active**

```bash
ssh jeremy@192.168.1.197 "cat /etc/cron.d/daily-reboot /etc/cron.d/boya-reset /etc/cron.d/birdnet-health-check"
```

Expected: Three cron files:
- `daily-reboot` — 4AM reboot
- `boya-reset` — every 15 min BOYA USB reset check
- `birdnet-health-check` — every 15 min service health check

**Step 7: Check disk and memory**

```bash
ssh jeremy@192.168.1.197 "df -h / && free -h"
```

Expected: Disk usage should be lower (~1.1 GB freed from binary cleanup). Memory should be lower (3 fewer Python services).

**Step 8: Document the baseline state**

After verification passes, record the stable state. This is the "known good" to compare against.

```bash
ssh jeremy@192.168.1.197 "echo '=== BASELINE $(date) ===' && \
    echo '--- Services ---' && systemctl list-units --type=service --state=running --no-pager | grep -v '^\s' && \
    echo '--- Cron ---' && cat /etc/cron.d/daily-reboot /etc/cron.d/boya-reset /etc/cron.d/birdnet-health-check && \
    echo '--- Disk ---' && df -h / && \
    echo '--- Memory ---' && free -h && \
    echo '--- BirdNET Version ---' && /usr/local/bin/birdnet-go --version 2>&1 && \
    echo '--- Uptime ---' && uptime"
```

---

## Task 8: Commit Plan and Script to Repo

**Target:** `/home/dev/workspace/birdnet-gone/` (dev environment)

**Step 1: Stage the new files**

```bash
cd /home/dev/workspace/birdnet-gone
git add docs/plans/2026-03-26-simplification-baseline.md scripts/birdnet-boya-reset.sh
```

**Step 2: Commit**

```bash
git commit -m "docs: add simplification baseline plan and BOYA reset script

Strip Pi to single service (birdnet-go-native) + shell-based BOYA
watchdog. Disable Flask display, VicoHome bridge, Python watchdog.
Simplify config by removing BirdWeather, weather, dynamic threshold,
dog bark filter, and monitoring. Point kiosk at built-in web UI."
```

---

## What Was Removed vs Kept

| Component | Status | Re-enable path |
|-----------|--------|---------------|
| `birdnet-go-native.service` | **KEPT** | — |
| `bird-display.service` | Disabled | `sudo systemctl enable --now bird-display.service` |
| `vicohome-bridge.service` | Disabled | `sudo systemctl enable --now vicohome-bridge.service` |
| `birdnet-watchdog.service` | Disabled | Replaced by `boya-reset` cron |
| BirdWeather | Config disabled | Set `birdweather.enabled: true` |
| Weather (yr.no) | Config disabled | Set `weather.provider: "yr.no"` |
| Dynamic threshold | Config disabled | Set `dynamicthreshold.enabled: true` |
| Dog bark filter | Config disabled | Set `dogbarkfilter.enabled: true` |
| Monitoring | Config disabled | Set `monitoring.enabled: true` |
| Debug mode | Config disabled | Set `debug: true` |
| 10 backup binaries (~1.1 GB) | Deleted | Rebuild from source if needed |

## Re-Integration Order (After Proving Stability)

Once the baseline runs for 3+ days without manual intervention:

1. **BirdWeather** — low risk, config toggle only
2. **Weather** — low risk, cosmetic
3. **Dynamic threshold** — medium risk, changes detection behavior
4. **VicoHome bridge** — medium risk, adds cloud dependency
5. **Custom display** — high risk, should be rewritten if re-added (remove `sudo reboot`/`poweroff` endpoints)
