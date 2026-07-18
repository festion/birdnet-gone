# Learnings — BirdNET-Go

> Project-specific knowledge accumulated from past sessions.
> When this file exceeds ~50 entries, split into mistakes.md, anti-patterns.md, validated.md, environment.md.

---

### Pi host + birdnet.db access (CURRENT) — 2026-06-08
- **The Pi moved to the IoT VLAN.** Host is `birdnet-pi5` at **192.168.10.246** (`jeremy@192.168.10.246`). The `192.168.1.197` in older entries below is **stale** — it no longer pings (consistent with the homelab IoT-VLAN migration). Service still `birdnet-go-native.service`; binary `/usr/local/bin/birdnet-go realtime`.
- **Detection DB:** `/home/jeremy/birdnet.db` (SQLite, GORM). Main table `notes` (one row per detection): `common_name, scientific_name, confidence, clip_name, date, time, threshold, sensitivity, begin_time, end_time, ...`; `results` table holds per-detection score rows (note_id→species,confidence).
- **`sqlite3` CLI is NOT installed on the Pi** — query via python3: `sqlite3.connect("file:/home/jeremy/birdnet.db?mode=ro",uri=True)`. Use `mode=ro` so a read query never creates root/owner-mismatched WAL/shm files while birdnet-go holds the DB open.
- **Clips:** every detection saves a clip (`notes.clip_name`, 1:1 — 0/8059 empty as of 06-08); files under `/home/jeremy/clips/<clip_name>` (e.g. `2026/06/baeolophus_bicolor_91p_…Z.wav`), full retention back to the DB's start, ~11 GB. Relevant for the second-level-reviewer evaluation (Vikunja #1710).

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

### BOYA desync has two failure modes — zero-amplitude vs I/O error — 2026-03-25
- **Context:** BOYA wireless desync. Two distinct failure modes observed.
- **Mode 1 — Zero-amplitude (soft desync):** USB receiver streams silence (-200 dB). Software USB unbind/rebind fixes it:
  ```bash
  sudo systemctl stop birdnet-go-native.service
  # Kill any orphaned birdnet-go processes first!
  sudo kill -9 $(pgrep -f 'birdnet-go realtime') 2>/dev/null
  echo '1-1.2' | sudo tee /sys/bus/usb/drivers/usb/unbind
  sleep 5
  echo '1-1.2' | sudo tee /sys/bus/usb/drivers/usb/bind
  sleep 3
  sudo systemctl start birdnet-go-native.service
  ```
- **Mode 2 — I/O error (hard desync):** USB receiver enumerates but `arecord` returns `pcm_read: read error: Input/output error`. Software USB reset does NOT fix this. **Requires physical power cycle of the BOYA transmitter.** The transmitter's 2.4GHz radio is fully desynced.
- **Critical:** The `Wants=sys-subsystem-sound-devices-card2.device` in the service unit causes systemd to auto-restart BirdNET on USB rebind, spawning orphan processes that hold `/dev/snd/pcmC2D0c`. **Always kill orphans before USB reset** or the watchdog/manual reset will fail with "device busy".
- **Diagnosis:** `arecord -D hw:2,0 -d 1 -f S24_3LE -r 48000 -c 2 /dev/null` — if "I/O error" → hard desync (need physical access). If silence → soft desync (USB reset may work).
- **Detection:** Watchdog `mean=-200.0 dB` is the reliable indicator. `events_received=0` is NOT (see separate learning).
- **Applies to:** Any BOYA desync investigation

### Range filter threshold is the highest-impact tuning lever — 2026-03-25
- **Context:** 1,848 detections in 4 days, 34 species, many questionable for north TX. `rangefilter.threshold` was 0.05 (5% occurrence probability — too loose).
- **Wrong:** Only adjusting per-species thresholds and global confidence. Leaves the candidate species pool wide open.
- **Right:** Tighten `rangefilter.threshold` first (0.05 → 0.01). This filters at the model level — species with <1% occurrence probability at the configured coordinates aren't even considered. Then layer on global threshold (0.6 → 0.65) and species-specific thresholds for volume control and habitat mismatches.
- **Full config:** See Serena memory `birdnet_detection_tuning_mar2026` for all 11 species thresholds.
- **Applies to:** Any future tuning session — start with range filter, then species-specific, then global.

### BOYA hard desync requires manual re-pair, not just power cycle — 2026-03-28
- **Context:** BOYA hard desync since Mar 26. Pi reboots, USB unbind/rebind, and even transmitter power cycling did NOT restore pairing.
- **Wrong:** Assuming power cycling the transmitter is sufficient. The BOYA's pairing table can get corrupted during hard desync, preventing auto-pair.
- **Right:** Use the manual re-pair procedure: (1) Power off transmitter, (2) On USB-C receiver: hold power 2s to off, then hold 5s to enter pairing mode, (3) On transmitter: hold power 5s while off to enter pairing mode, (4) Both LEDs blink blue quickly → solid blue = paired.
- **LED guide:** Slow blue blink = unpaired/idle (NOT actively pairing). Fast blue blink = pairing mode (5 min timeout). Solid blue = paired. Fast red blink = low battery.
- **Also required:** Physical USB reseat of receiver + Pi reboot before re-pair worked. USB unbind/rebind alone was insufficient.
- **Applies to:** Any BOYA hard desync where USB reset + power cycle fails

### Health check v5 had broken journal grep — replaced with MQTT-based v6 — 2026-03-28
- **Context:** `check_inference_activity()` grepped journalctl for `"Published sound level data"` — a string that never appears in the journal (logged to file via `analysis` module logger, not console)
- **Wrong:** Searching journalctl for module-logged messages. Also had bash syntax error from `grep -c` returning multiline output.
- **Right:** Health check v6 uses MQTT sound level check (same as boya-reset.sh) — reads `birdnet/soundlevel` topic, checks timestamp freshness (< 2 min). Deployed to `/usr/local/bin/birdnet-health-check.sh`.
- **Also fixed:** Double-logging (tee + cron redirect to same file), cron race condition (health check staggered to :03/:18/:33/:48, BOYA reset stays at :00/:15/:30/:45)
- **Applies to:** Any future health check modifications

### DSI display brightness resets to 15/255 on reboot — 2026-03-28
- **Context:** After Pi reboot, display appeared blank — brightness was 15/255 (nearly off)
- **Right:** Created `display-brightness.service` (oneshot, multi-user.target) that writes 200 to `/sys/class/backlight/10-0045/brightness` on boot
- **Applies to:** Any Pi reboot or display troubleshooting

### ALSA device is exclusive — arecord fails with "Device busy" while BirdNET-Go runs — 2026-03-26
- **Context:** BOYA reset script tried `arecord -D hw:2,0` as a smoke test for hard desync while birdnet-go-native was running
- **Wrong:** Assumed BOYA's ALSA driver supports concurrent reads. arecord returned "audio open error: Device or resource busy"
- **Right:** BirdNET-Go holds the ALSA device exclusively. Cannot use arecord while service is running. Use MQTT sound level (-200 dB = silence) as the sole desync indicator.
- **Applies to:** Any diagnostic that tries to read the audio device while BirdNET-Go is active

### Fork's 3 MQTT patches still needed as of upstream nightly-20260322 — 2026-03-26
- **Context:** Evaluated whether fork could be replaced with stock upstream
- **Verified:** Upstream still uses random UUIDs for source IDs (not deterministic), has no stale discovery cleanup, and uses bare `this.state` in measurement sensor templates
- **Right:** Fork patches a953b18d (deterministic IDs), e3308370 (stale cleanup), 64f7799a (HA measurement fix) are all still required
- **Applies to:** Any future consideration of dropping the fork or rebasing onto upstream

### Flask display exposed unauthenticated sudo reboot — caused unscheduled reboot — 2026-03-26
- **Context:** Pi rebooted at 07:15 from `POST /reboot` to Flask app from localhost (127.0.0.1)
- **Wrong:** Flask display app exposed `sudo reboot`, `sudo poweroff`, `sudo systemctl restart` as unauthenticated HTTP POST endpoints accessible from the kiosk Chromium browser
- **Right:** If custom display is ever re-enabled, remove all system control endpoints or add authentication. The built-in BirdNET-Go web UI (port 8080) has no such endpoints.
- **Applies to:** Any future display/kiosk development

### MQTT discovery source IDs regenerate on restart — stale configs accumulate — 2026-02-17
- **Context:** After deploying a new binary, HA still showed errors from old sensors
- **Root cause:** Audio source IDs (e.g., `audio_card_6a7886c1`) are regenerated each time BirdNET starts. Old retained MQTT discovery configs with old source IDs persist on the broker, creating ghost sensors in HA
- **Fix:** Clear stale retained discovery messages: `mosquitto_pub -h BROKER -u USER -P PASS -t 'homeassistant/sensor/NODE/STALE_TOPIC/config' -r -n` (empty retained message clears the topic)
- **Prevention:** BirdNET-Go should ideally clean up old discovery configs on startup or use stable source IDs
- **Applies to:** Any BirdNET-Go deployment where audio sources change between restarts

### Repo CI/hygiene facts (Go 1.26 adoption + container build) — 2026-07-18
- **Go base image lives in the Dockerfile, NOT setup-go** — `Dockerfile:4 FROM golang:<ver>-trixie`. Bumping go.mod's `go` directive requires bumping this too (and it's easy to miss: golangci/unit-tests use setup-go and pass regardless). Now `golang:1.26-trixie` (floating minor). Dependabot ecosystems here = **gomod, github-actions, npm — NOT docker**, so the Dockerfile pin drifts from go.mod unless hand-tracked (a floating `1.26-trixie` self-tracks). Consider adding `docker` to `.github/dependabot.yml`.
- **`docker-build.yml` `build-and-push-docker-image` job runs ONLY on push-to-main** (`if: (push || workflow_dispatch) && ref_name==main` after #151; was push-only). It's SKIPPED on PRs — so a Dockerfile/CI change can NEVER be verified on a PR; and the push trigger has `paths-ignore: .github/workflows/**` so a workflow-only change won't re-trigger it. To verify: **`gh workflow run "Build dev container images" --ref main`** (dispatch enabled in #151), or a code push to main.
- **Container build+push needs `packages: write`** — the job had no `permissions:` block, so the repo's read-only `default_workflow_permissions` token got `denied: installation not allowed to Create organization package` pushing to ghcr.io. Fixed with job-scoped `permissions: {contents: read, packages: write}` (#151). No org-level package-creation-policy change was needed.
- **`Update Licenses` workflow** — `peter-evans/create-pull-request` fails `GitHub Actions is not permitted to create or approve pull requests` because the repo's `can_approve_pull_request_reviews` is false. main is UNPROTECTED, so the fix (#149) commits `LICENSES.md` directly to main instead (guarded on a diff, `[skip ci]`, `fetch-depth: 0` for the rebase) — avoids enabling the org-wide "Actions can create PRs" setting.
- **Dependabot auto-merge is in-workflow (Template B)** and only merges patch/minor; the gate maps directory `/` → `golangci/lint`, which is path-filtered off workflows-only changes — so **actions-group** PRs never satisfy the required check and must be merged manually. Old frontend PRs stuck CLEAN-but-unmerged just need a `@dependabot rebase` to re-trigger the (now-green) gate.
- **Node is managed by `n`** (`/usr/local/bin/node`, dev-writable so no sudo). Bumped 22.17.0 → 22.23.1 for lint-staged 17 (`engines.node >=22.22.1`). Gotcha: the bundled npm under `/usr/local/lib/node_modules` is root-owned, so `n` can't replace it (prints permission errors) — cosmetic, npm still works and matches the node version.
