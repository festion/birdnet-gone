# Learnings — BirdNET-Go

> Project-specific knowledge accumulated from past sessions.
> When this file exceeds ~50 entries, split into mistakes.md, anti-patterns.md, validated.md, environment.md.

---

### BirdNET-Go nightly regression broke RTSP — 2026-02-06
- **Context:** Updated BirdNET-Go from nightly-20260113 to nightly-20260118
- **Wrong:** Assumed newer nightly would be stable — it had an upstream RTSP regression (issue #1870)
- **Right:** Always check GitHub issues before upgrading to nightly builds. Keep the previous binary as a backup at `/usr/local/bin/birdnet-go.<version>-backup`
- **Applies to:** BirdNET-Go (192.168.1.197), any service using nightly/pre-release builds

### BirdNET-Go stable vs nightly config format
- Stable releases (v0.6.x) use an incompatible config format from nightly builds. Don't attempt rollback from nightly to stable without config migration. SSH: `jeremy@192.168.1.197`, service: `birdnet-go-native.service`.
