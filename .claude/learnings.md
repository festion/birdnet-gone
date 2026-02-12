# Learnings — BirdNET-Go

> Project-specific knowledge accumulated from past sessions.
> When this file exceeds ~50 entries, split into mistakes.md, anti-patterns.md, validated.md, environment.md.

---

### BirdNET-Go nightly builds — keep previous version and don't mix with stable
- When upgrading nightlies, keep backups at `/usr/local/bin/birdnet-go.<version>`.
- Stable releases (v0.6.x) use an **incompatible config format** from nightly builds. Don't attempt rollback from nightly to stable without config migration.
- Backups on RPi: `/usr/local/bin/birdnet-go.nightly-20251012`, `/usr/local/bin/birdnet-go.nightly-20260113`
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
- **Current deployed version:** `nightly-20260118-35-g9317d8bb` (fork build with debug stderr logging)

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
