# Learnings — BirdNET-Go

> Project-specific knowledge accumulated from past sessions.
> When this file exceeds ~50 entries, split into mistakes.md, anti-patterns.md, validated.md, environment.md.

---

### BirdNET-Go nightly builds — keep previous version and don't mix with stable
- When upgrading nightlies, keep backups at `/usr/local/bin/birdnet-go.<version>`.
- Stable releases (v0.6.x) use an **incompatible config format** from nightly builds. Don't attempt rollback from nightly to stable without config migration.
- Backups on RPi: `/usr/local/bin/birdnet-go.nightly-20251012`, `/usr/local/bin/birdnet-go.nightly-20260113`, `/usr/local/bin/birdnet-go.nightly-20260118-35`
- SSH: `jeremy@192.168.1.197`, service: `birdnet-go-native.service`

### Sound levels ≠ detections — separate code paths — 2026-02-06
- Sound level MQTT publishing only confirms audio flows through FFmpeg, NOT that BirdNET inference is receiving data. Always verify with `birdnet-go file` on a captured WAV to confirm the analysis pipeline works.

### Rebasing fork onto upstream nightly tag — 2026-02-07
- **Context:** Fork's `main` had 51 commits: 32 fork-specific + merge commit + 17 upstream commits pulled via merge. Needed to rebase onto `nightly-20260118` tag to match production binary.
- **Approach:** Cherry-pick fork-only commits onto a new branch from the tag, skip merge commit + upstream PR commits + post-merge fixup commits.
- **Steps:** (1) `git branch backup-main-pre-rebase main` (2) `git checkout -b main-rebased nightly-20260118` (3) Cherry-pick fork commits oldest-first (4) `git branch -m main main-old && git branch -m main-rebased main` (5) `git push origin main --force-with-lease`
- **Gotcha: auto-generated files (PROJECT_INDEX.md) cause repeated conflicts.** Resolve with `git checkout --theirs <file> && git add <file> && git cherry-pick --continue`.
- **Gotcha: upstream API refactors cause build failures after cherry-pick.** The fork's `telemetry.go` used `slog` directly but upstream moved to `internal/logger` with custom `LogLevel`/`Field` types. The fork's `settings.go` had a `GetExcludedSpecies` duplicate already in `detections.go`. Fix these post-cherry-pick before building.
- **Verification:** `git log --oneline TAG..main | wc -l` for count, `git merge-base main TAG` must equal tag hash, `git log | grep "Merge pull request"` should return 0.
- **Applies to:** Any fork rebase where upstream was merged (not rebased) into the fork

### Deploy fork binary to RPi — 2026-02-07
- **Validated pattern:** `scp bin/birdnet-go jeremy@192.168.1.197:/tmp/birdnet-go-fork` then SSH to backup, stop, copy, start
- Backup naming: `/usr/local/bin/birdnet-go.<version-description>` (e.g., `nightly-20260118-vanilla`)
- MQTT broker may be transiently unreachable during restart — service restart after deploy resolves it
- **Current deployed version:** `nightly-20260118-47-g936f3159` (fork build with HA discovery fix)

### Logger module bug silently drops all [audio.ffmpeg] output — 2026-02-07
- **Context:** Debugging zero detections — no FFmpeg/RTSP error logs visible at all
- **Wrong:** Assumed logger was working. Spent hours thinking the pipeline code was silently failing.
- **Right:** `logger.Global().Module("audio").Module("ffmpeg")` creates a moduleLogger that drops ALL output (Info, Warn, Error). Known upstream issue [#1843](https://github.com/tphakala/birdnet-go/issues/1843). **Workaround:** Use `fmt.Fprintf(os.Stderr, ...)` to bypass the logger for debugging.
- **Key insight:** When all logs from a module are missing, suspect the logger itself before suspecting the code.
- **Applies to:** BirdNET-Go nightly-20260118, any module using nested `.Module().Module()` calls

### Pi kiosk boot mode: Desktop (B4) required for bird display — 2026-02-12
- **Context:** `pi-status-dashboard` kiosk-setup.sh was run on the BirdNET Pi, changing boot from Desktop auto-login (B4) to Console auto-login (B2) and launching Chromium with Grafana dashboard URLs instead of bird display
- **Symptom:** Pi display shows rotating Grafana dashboards instead of bird detection display
- **Fix:** `sudo raspi-config nonint do_boot_behaviour B4` + reboot. The existing `~/.config/autostart/bird-display-kiosk.desktop` autostart handles launching Chromium with `http://localhost:5000`
- **Key insight:** The original kiosk uses labwc (Wayland compositor) via Desktop session — switching to Console mode bypasses the desktop environment and its autostart `.desktop` files entirely
- **Applies to:** BirdNET RPi kiosk display at 192.168.1.197

### MQTT discovery `this.state` fallback breaks HA measurement sensors — 2026-02-17
- **Context:** HA logged `Value error while updating state` for all BirdNET confidence/sound_level sensors
- **Wrong:** Value templates used `else this.state` as fallback when `sourceId` didn't match. On first message (or after availability transition), `this.state` is `'unknown'` — violates `state_class: measurement` numeric requirement
- **Right:** Use `else (this.state if this.state not in ['unknown', 'unavailable'] else none)`. Returning `none` from value_template tells HA to skip the state update entirely
- **Applies to:** Any MQTT auto-discovery sensor with `state_class: measurement` that filters by source ID

### Event bus `events_received=0` is normal — only new species trigger events — 2026-03-25
- **Context:** Morning briefing showed `events_received=0` for 3.5+ hours. Assumed BOYA wireless desync (zero detections). Actually, detections were being stored to the DB the whole time.
- **Wrong:** Treating `events_received=0` as evidence of no detections / mic failure. Rebooted Pi unnecessarily.
- **Right:** `events_received` only counts events published to the event bus. **Only new species detections** publish events (via `publishNewSpeciesDetectionEvent()` in `actions.go:637`). All previously-seen species are saved to the database but never touch the event bus. Zero events is normal when all detected species are familiar.
- **Verification:** Use `/api/v2/analytics/species/summary` (checks DB aggregates) instead of event bus metrics to confirm detections are happening. The `/api/v2/detections` endpoint may also have query parameter quirks — species summary is more reliable for a quick health check.
- **Applies to:** BirdNET-Go briefing/monitoring, any future zero-detection investigation

### BOYA desync recovery requires USB receiver reset, not just transmitter power cycle — 2026-03-25
- **Context:** BOYA wireless desync (zero-amplitude audio, -200 dB). Physical transmitter power cycle + BirdNET service restart did NOT fix it. Watchdog continued reporting silence.
- **Wrong:** Only power cycling the transmitter and restarting BirdNET. The USB receiver holds a stale audio stream.
- **Right:** Software USB reset via unbind/rebind, then restart BirdNET:
  ```bash
  sudo systemctl stop birdnet-go-native.service
  echo '1-1.2' | sudo tee /sys/bus/usb/drivers/usb/unbind
  sleep 3
  echo '1-1.2' | sudo tee /sys/bus/usb/drivers/usb/bind
  sleep 2
  sudo systemctl start birdnet-go-native.service
  ```
  The USB port `1-1.2` is the BOYA receiver on the Pi. Unbind forces the kernel to drop the audio device; rebind triggers full re-enumeration. BOYA re-enumerates as card 2.
- **Detection:** Watchdog `mean=-200.0 dB` is the reliable indicator. `events_received=0` is NOT (see separate learning).
- **Applies to:** Any BOYA desync where transmitter power cycle alone doesn't resolve

### Range filter threshold is the highest-impact tuning lever — 2026-03-25
- **Context:** 1,848 detections in 4 days, 34 species, many questionable for north TX. `rangefilter.threshold` was 0.05 (5% occurrence probability — too loose).
- **Wrong:** Only adjusting per-species thresholds and global confidence. Leaves the candidate species pool wide open.
- **Right:** Tighten `rangefilter.threshold` first (0.05 → 0.01). This filters at the model level — species with <1% occurrence probability at the configured coordinates aren't even considered. Then layer on global threshold (0.6 → 0.65) and species-specific thresholds for volume control and habitat mismatches.
- **Full config:** See Serena memory `birdnet_detection_tuning_mar2026` for all 11 species thresholds.
- **Applies to:** Any future tuning session — start with range filter, then species-specific, then global.

### MQTT discovery source IDs regenerate on restart — stale configs accumulate — 2026-02-17
- **Context:** After deploying a new binary, HA still showed errors from old sensors
- **Root cause:** Audio source IDs (e.g., `audio_card_6a7886c1`) are regenerated each time BirdNET starts. Old retained MQTT discovery configs with old source IDs persist on the broker, creating ghost sensors in HA
- **Fix:** Clear stale retained discovery messages: `mosquitto_pub -h BROKER -u USER -P PASS -t 'homeassistant/sensor/NODE/STALE_TOPIC/config' -r -n` (empty retained message clears the topic)
- **Prevention:** BirdNET-Go should ideally clean up old discovery configs on startup or use stable source IDs
- **Applies to:** Any BirdNET-Go deployment where audio sources change between restarts
