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

### nightly-20260113 has RTSP→analysis buffer bug — 2026-02-06
- **Context:** Upgraded to nightly-20260113 expecting native RTSP support
- **Wrong:** Sound levels publish (separate code path via `soundLevelChan`) but `WriteToAnalysisBuffer` never receives data from RTSP streams. Zero detections despite audio flowing.
- **Right:** Rolled back to `nightly-20251012` which uses malgo/ALSA input. Created `rtsp-audio-bridge.service` to bridge ESP32 RTSP→ALSA loopback.
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
