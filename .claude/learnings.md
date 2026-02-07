# Learnings — BirdNET-Go

> Project-specific knowledge accumulated from past sessions.
> When this file exceeds ~50 entries, split into mistakes.md, anti-patterns.md, validated.md, environment.md.

---

### ESP32 ring buffer refactoring broke RTSP streaming — 2026-02-06
- **Context:** Added HTTP audio streaming to ESP32 firmware by refactoring I2S→RTP into I2S→ring buffer→RTP/HTTP
- **Wrong:** Commit `03de8697` broke RTSP — ring buffer consumer (`xRingbufferReceive`) never returned data during RTSP streaming, and first connection even crashed the ESP32. Blamed upstream BirdNET-Go (issue #1870) for days instead of testing our own firmware change
- **Right:** (1) Always test the ORIGINAL functionality after refactoring. (2) When a popular project "doesn't work", check your own code first. (3) Use raw TCP RTSP handshake tests (Python socket) to isolate protocol issues from application issues. (4) Don't write custom firmware code when upstream works
- **Applies to:** ESP32 firmware in `festion/birdnet-gone`, any firmware refactoring

### BirdNET-Go nightly builds — keep previous version
- When upgrading nightlies, keep backups at `/usr/local/bin/birdnet-go.<version>`. The config format differs between stable (v0.6.x) and nightly, so don't try stable rollback without config migration.
- **Current working combo:** `nightly-20260118` + upstream Sukecz v1.3.0 firmware + ALSA loopback bridge
- Backups on RPi: `/usr/local/bin/birdnet-go.nightly-20251012`, `/usr/local/bin/birdnet-go.nightly-20260113`

### BirdNET-Go stable vs nightly config format
- Stable releases (v0.6.x) use an incompatible config format from nightly builds. Don't attempt rollback from nightly to stable without config migration. SSH: `jeremy@192.168.1.197`, service: `birdnet-go-native.service`.

### nightly-20260113+ has RTSP→analysis buffer bug — 2026-02-06
- **Context:** Upgraded to nightly-20260113 expecting native RTSP support
- **Wrong:** Sound levels publish (separate code path via `soundLevelChan`) but `WriteToAnalysisBuffer` never receives data from RTSP streams. Zero detections despite audio flowing.
- **Update (2026-02-07):** The RTSP pipeline DOES work in nightly-20260118 — confirmed `handleAudioData()` calls `WriteToAnalysisBuffer()` with data. The zero-detections issue was caused by ESP32 thermal shutdown disabling RTSP server (Connection refused), hidden by logger bug #1843.
- **Applies to:** BirdNET-Go nightly-20260113 and nightly-20260118

### Sound levels ≠ detections — separate code paths — 2026-02-06
- Sound level MQTT publishing only confirms audio flows through FFmpeg, NOT that BirdNET inference is receiving data. Always verify with `birdnet-go file` on a captured WAV to confirm the analysis pipeline works.

### ESP32 ICS-43434 MEMS mic insufficient for outdoor bird detection — 2026-02-06
- **Context:** ESP32 mic placed at same outdoor location as old USB mic that produced 1,644 detections
- **Wrong:** Assumed ESP32 audio quality was equivalent. Spent hours on gain adjustment (-24dB, -8dB, raw) and highpass filtering
- **Right:** Frequency analysis revealed ESP32 audio is 53% sub-1kHz noise, only 8% in 2-4kHz bird range. USB mic clips were 86-100% in 2-4kHz. ICS-43434 (65 dBA SNR) drowns bird signals in ambient noise at garden distance. **Gain/filtering can't fix fundamental SNR deficiency.**
- **Applies to:** Any MEMS mic selection for outdoor audio capture

### ALSA loopback bridge for BirdNET-Go — 2026-02-06
- **Validated pattern:** ESP32 RTSP → FFmpeg (with filters) → `plughw:0,1` → BirdNET-Go reads `Loopback` via malgo
- malgo device matching uses `strings.Contains(info.Name(), audioSource)` — use device NAME substring (e.g., `Loopback`), not ALSA PCM name
- Bridge service: `/etc/systemd/system/rtsp-audio-bridge.service`
- Current FFmpeg filters: `highpass=f=500:p=2,highpass=f=1000:p=1,volume=-8dB`

### Rebasing fork onto upstream nightly tag — 2026-02-07
- **Context:** Fork's `main` had 51 commits: 32 fork-specific + merge commit + 17 upstream commits pulled via merge. Needed to rebase onto `nightly-20260118` tag to match production binary.
- **Approach:** Cherry-pick fork-only commits onto a new branch from the tag, skip merge commit + upstream PR commits + post-merge fixup commits.
- **Steps:** (1) `git branch backup-main-pre-rebase main` (2) `git checkout -b main-rebased nightly-20260118` (3) Cherry-pick fork commits oldest-first (4) `git branch -m main main-old && git branch -m main-rebased main` (5) `git push origin main --force-with-lease`
- **Gotcha: `git cherry-pick --abort` reverts ALL commits in a multi-commit sequence**, even already-committed ones. If cherry-picking `A B C D` fails on C, aborting undoes A and B too. Fix: cherry-pick one at a time near conflict points, or note which commits applied before aborting.
- **Gotcha: auto-generated files (PROJECT_INDEX.md) cause repeated conflicts.** Resolve with `git checkout --theirs <file> && git add <file> && git cherry-pick --continue`.
- **Gotcha: upstream API refactors cause build failures after cherry-pick.** The fork's `telemetry.go` used `slog` directly but upstream moved to `internal/logger` with custom `LogLevel`/`Field` types. The fork's `settings.go` had a `GetExcludedSpecies` duplicate already in `detections.go`. Fix these post-cherry-pick before building.
- **Verification:** `git log --oneline TAG..main | wc -l` for count, `git merge-base main TAG` must equal tag hash, `git log | grep "Merge pull request"` should return 0.
- **Applies to:** Any fork rebase where upstream was merged (not rebased) into the fork

### Deploy fork binary to RPi — 2026-02-07
- **Validated pattern:** `scp bin/birdnet-go jeremy@192.168.1.197:/tmp/birdnet-go-fork` then SSH to backup, stop, copy, start
- Backup naming: `/usr/local/bin/birdnet-go.<version-description>` (e.g., `nightly-20260118-vanilla`)
- MQTT broker may be transiently unreachable during restart — service restart after deploy resolves it
- **Current deployed version:** `nightly-20260118-35-g9317d8bb` (fork build with debug stderr logging)

### Logger module bug silently drops all [audio.ffmpeg] output — 2026-02-07
- **Context:** Debugging zero detections — no FFmpeg/RTSP error logs visible at all
- **Wrong:** Assumed logger was working. Spent hours thinking the pipeline code was silently failing.
- **Right:** `logger.Global().Module("audio").Module("ffmpeg")` creates a moduleLogger that drops ALL output (Info, Warn, Error). Known upstream issue [#1843](https://github.com/tphakala/birdnet-go/issues/1843). **Workaround:** Use `fmt.Fprintf(os.Stderr, ...)` to bypass the logger for debugging.
- **Key insight:** When all logs from a module are missing, suspect the logger itself before suspecting the code.
- **Applies to:** BirdNET-Go nightly-20260118, any module using nested `.Module().Module()` calls

### ESP32 thermal protection disables RTSP silently — 2026-02-07
- **Context:** ESP32 Sukecz v1.3.0 firmware at 192.168.1.183 hit 95.8°C
- **Symptom:** RTSP port 8554 refuses connections. ESP32 pingable, web UI works. BirdNET-Go FFmpeg retries every ~30s with "Connection refused" but errors hidden by logger bug.
- **Diagnosis:** `curl -s http://192.168.1.183/api/status` shows `rtsp_server_enabled: false`. `curl -s http://192.168.1.183/api/thermal` shows `latched_persist: true`.
- **Fix:** `curl -X POST http://192.168.1.183/api/thermal/clear` re-enables RTSP after cooldown.
- **Problem:** ESP32 runs at ~95°C continuously and will keep hitting the thermal limit. Needs hardware cooling or higher limit.
- **Applies to:** ESP32 RTSP mic firmware with thermal protection enabled
