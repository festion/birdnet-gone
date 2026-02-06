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
- **Current working combo:** `nightly-20260113` + upstream Sukecz v1.3.0 firmware

### BirdNET-Go stable vs nightly config format
- Stable releases (v0.6.x) use an incompatible config format from nightly builds. Don't attempt rollback from nightly to stable without config migration. SSH: `jeremy@192.168.1.197`, service: `birdnet-go-native.service`.
