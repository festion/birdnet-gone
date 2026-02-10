# BirdNET-Go Configuration Tuning Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Tune the production BirdNET-Go configuration on the Raspberry Pi (192.168.1.197) to reduce false positives, lower CPU overhead, and improve detection quality.

**Architecture:** All changes are to the remote YAML config file at `/home/jeremy/.config/birdnet-go/config.yaml` on `jeremy@192.168.1.197`. Changes are applied via SSH `sed` commands, validated by grep, and activated by restarting the `birdnet-go-native.service`. A pre-change backup is taken first. After all changes, a validation query runs against the SQLite database to establish a baseline.

**Tech Stack:** SSH, sed, systemd, SQLite3, YAML config

**Remote Host:** `jeremy@192.168.1.197`
**Config Path:** `/home/jeremy/.config/birdnet-go/config.yaml`
**Service:** `birdnet-go-native.service`
**Database:** `/home/jeremy/birdnet.db`

---

## Pre-Requisites

- SSH access to `jeremy@192.168.1.197` (key-based, no password)
- `sudo` access for systemctl operations
- The config file is a flat YAML file edited in-place via sed (BirdNET-Go rewrites it on settings save, so formatting is stable)

---

### Task 1: Backup Current Configuration

**Why:** Safety net before making any changes. If detections degrade, we can restore.

**Files:**
- Read: `jeremy@192.168.1.197:/home/jeremy/.config/birdnet-go/config.yaml`
- Create: `jeremy@192.168.1.197:/home/jeremy/.config/birdnet-go/config.yaml.bak.2026-02-10`

**Step 1: Create timestamped backup**

```bash
ssh jeremy@192.168.1.197 "cp /home/jeremy/.config/birdnet-go/config.yaml /home/jeremy/.config/birdnet-go/config.yaml.bak.2026-02-10"
```

**Step 2: Verify backup exists and matches**

```bash
ssh jeremy@192.168.1.197 "diff /home/jeremy/.config/birdnet-go/config.yaml /home/jeremy/.config/birdnet-go/config.yaml.bak.2026-02-10"
```

Expected: No output (files identical)

**Step 3: Record baseline detection stats**

```bash
ssh jeremy@192.168.1.197 "sqlite3 /home/jeremy/birdnet.db \"SELECT COUNT(*) as total, ROUND(AVG(confidence),3) as avg_conf, MIN(confidence) as min_conf FROM notes WHERE date >= date('now', '-7 days');\""
```

Save the output for comparison after tuning.

**Step 4: Commit (local docs only)**

```bash
# No commit needed - remote-only operation
```

---

### Task 2: Raise Range Filter Threshold (0.03 -> 0.05)

**Why:** The range filter uses BirdNET's species occurrence model to filter out-of-season/out-of-range species. At 0.03 it's essentially disabled, allowing false positives like Purple Martin in February (8 detections on Feb 9 despite species not arriving in North TX until mid-March). Raising to 0.05 enables meaningful geographic/seasonal filtering while remaining permissive for early arrivals.

**Files:**
- Modify: `jeremy@192.168.1.197:/home/jeremy/.config/birdnet-go/config.yaml` (line ~133)

**Step 1: Apply the change**

```bash
ssh jeremy@192.168.1.197 "sed -i '/rangefilter:/,/threshold:/{s/threshold: 0.03/threshold: 0.05/}' /home/jeremy/.config/birdnet-go/config.yaml"
```

**Step 2: Verify the change**

```bash
ssh jeremy@192.168.1.197 "grep -A4 'rangefilter:' /home/jeremy/.config/birdnet-go/config.yaml | head -5"
```

Expected output should show `threshold: 0.05`

---

### Task 3: Raise Dynamic Threshold Minimum (0.4 -> 0.5)

**Why:** 29 detections below 0.5 confidence in the last month are all species also detected at higher confidence — they're noise. Raising the floor from 0.4 to 0.5 eliminates these without losing any real species. Also extend validhours from 24 to 48 to give more time for threshold establishment.

**Files:**
- Modify: `jeremy@192.168.1.197:/home/jeremy/.config/birdnet-go/config.yaml` (lines ~206-207)

**Step 1: Raise min threshold**

```bash
ssh jeremy@192.168.1.197 "sed -i '/dynamicthreshold:/,/validhours:/{s/min: 0.4/min: 0.5/}' /home/jeremy/.config/birdnet-go/config.yaml"
```

**Step 2: Extend valid hours**

```bash
ssh jeremy@192.168.1.197 "sed -i '/dynamicthreshold:/,/falsepositivefilter:/{s/validhours: 24/validhours: 48/}' /home/jeremy/.config/birdnet-go/config.yaml"
```

**Step 3: Verify both changes**

```bash
ssh jeremy@192.168.1.197 "grep -A6 'dynamicthreshold:' /home/jeremy/.config/birdnet-go/config.yaml | head -7"
```

Expected: `min: 0.5` and `validhours: 48`

---

### Task 4: Reduce Analysis Overlap (2.2 -> 1.5)

**Why:** Overlap of 2.2s on a 3s window means only 0.8s of new audio per inference — very aggressive. This causes higher CPU usage and more duplicate detections. 1.5s overlap (1.5s new audio per cycle) is a well-tested balance between catch rate and efficiency.

**Files:**
- Modify: `jeremy@192.168.1.197:/home/jeremy/.config/birdnet-go/config.yaml` (line ~124)

**Step 1: Apply the change**

```bash
ssh jeremy@192.168.1.197 "sed -i 's/overlap: 2.2/overlap: 1.5/' /home/jeremy/.config/birdnet-go/config.yaml"
```

**Step 2: Verify**

```bash
ssh jeremy@192.168.1.197 "grep 'overlap:' /home/jeremy/.config/birdnet-go/config.yaml"
```

Expected: `overlap: 1.5`

---

### Task 5: Disable Global Debug Mode

**Why:** `debug: true` is set at the top level plus ~15 individual modules. This adds CPU overhead from verbose logging and makes real issues harder to spot. Production should run at info level.

**Files:**
- Modify: `jeremy@192.168.1.197:/home/jeremy/.config/birdnet-go/config.yaml` (lines 1, 121, 130, 144, 150, 168, 187, 204, 218, 234, 238, 252, 315, 354, 358, 364, 384, 399)

**Step 1: Disable top-level debug**

```bash
ssh jeremy@192.168.1.197 "sed -i '1s/^debug: true$/debug: false/' /home/jeremy/.config/birdnet-go/config.yaml"
```

**Step 2: Disable all indented debug flags**

```bash
ssh jeremy@192.168.1.197 "sed -i 's/        debug: true/        debug: false/g' /home/jeremy/.config/birdnet-go/config.yaml"
```

This targets the 8-space-indented `debug: true` lines (all module-level debug flags).

**Step 3: Also disable 4-space-indented debug flags (birdnet section)**

```bash
ssh jeremy@192.168.1.197 "sed -i 's/    debug: true/    debug: false/g' /home/jeremy/.config/birdnet-go/config.yaml"
```

**Step 4: Verify no debug: true remains**

```bash
ssh jeremy@192.168.1.197 "grep 'debug: true' /home/jeremy/.config/birdnet-go/config.yaml"
```

Expected: No output (all debug flags disabled)

---

### Task 6: Disable Sentry Telemetry

**Why:** Sentry sends error telemetry to the upstream BirdNET-Go project. This is a fork build, so telemetry would be confusing/misleading upstream. Disable it.

**Files:**
- Modify: `jeremy@192.168.1.197:/home/jeremy/.config/birdnet-go/config.yaml` (line ~383)

**Step 1: Apply the change**

```bash
ssh jeremy@192.168.1.197 "sed -i '/^sentry:/,/^[a-z]/{s/enabled: true/enabled: false/}' /home/jeremy/.config/birdnet-go/config.yaml"
```

**Step 2: Verify**

```bash
ssh jeremy@192.168.1.197 "grep -A2 '^sentry:' /home/jeremy/.config/birdnet-go/config.yaml"
```

Expected: `enabled: false`

---

### Task 7: Add Purple Martin Species Threshold

**Why:** 8 Purple Martin false positives on Feb 9 (species doesn't arrive until mid-March). Add a high threshold so only very confident detections are logged during the off-season. The range filter fix (Task 2) helps long-term, but a species-specific threshold adds defense in depth.

**Files:**
- Modify: `jeremy@192.168.1.197:/home/jeremy/.config/birdnet-go/config.yaml` (after line ~311, inside species.config block)

**Step 1: Add purple martin config after northern mockingbird block**

```bash
ssh jeremy@192.168.1.197 "sed -i '/northern mockingbird:/,/actions: \[\]/{/actions: \[\]/a\\            purple martin:\\n                threshold: 0.85\\n                interval: 0\\n                actions: []}' /home/jeremy/.config/birdnet-go/config.yaml"
```

**Step 2: Verify**

```bash
ssh jeremy@192.168.1.197 "grep -A3 'purple martin:' /home/jeremy/.config/birdnet-go/config.yaml"
```

Expected:
```yaml
            purple martin:
                threshold: 0.85
                interval: 0
                actions: []
```

---

### Task 8: Raise Northern Mockingbird Threshold (0.35 -> 0.45)

**Why:** Mockingbirds mimic other species, so the model naturally has lower confidence. But 0.35 is very low and catches many marginal calls. 0.45 is still permissive for a mimic species while reducing noise.

**Files:**
- Modify: `jeremy@192.168.1.197:/home/jeremy/.config/birdnet-go/config.yaml` (line ~309)

**Step 1: Apply the change**

```bash
ssh jeremy@192.168.1.197 "sed -i '/northern mockingbird:/,/threshold:/{s/threshold: 0.35/threshold: 0.45/}' /home/jeremy/.config/birdnet-go/config.yaml"
```

**Step 2: Verify**

```bash
ssh jeremy@192.168.1.197 "grep -A2 'northern mockingbird:' /home/jeremy/.config/birdnet-go/config.yaml"
```

Expected: `threshold: 0.45`

---

### Task 9: Add Database Backup Cron Job

**Why:** The SQLite database at `/home/jeremy/birdnet.db` has no backup. A simple daily cron job copies it before the 4 AM reboot.

**Files:**
- Create: `jeremy@192.168.1.197:/etc/cron.d/birdnet-db-backup`

**Step 1: Create the cron job**

```bash
ssh jeremy@192.168.1.197 "echo '30 3 * * * jeremy cp /home/jeremy/birdnet.db /home/jeremy/birdnet.db.daily-backup' | sudo tee /etc/cron.d/birdnet-db-backup > /dev/null && sudo chmod 644 /etc/cron.d/birdnet-db-backup"
```

This runs at 3:30 AM daily (30 min before the 4 AM reboot), copying the DB as user `jeremy`.

**Step 2: Verify**

```bash
ssh jeremy@192.168.1.197 "cat /etc/cron.d/birdnet-db-backup"
```

Expected: `30 3 * * * jeremy cp /home/jeremy/birdnet.db /home/jeremy/birdnet.db.daily-backup`

---

### Task 10: Restart Service and Validate

**Why:** All config changes require a service restart. After restart, verify the service is healthy and detections are flowing.

**Step 1: Diff the config against backup to review all changes**

```bash
ssh jeremy@192.168.1.197 "diff /home/jeremy/.config/birdnet-go/config.yaml.bak.2026-02-10 /home/jeremy/.config/birdnet-go/config.yaml"
```

Review the diff carefully. Expected changes:
- `debug: true` -> `debug: false` (many lines)
- `overlap: 2.2` -> `overlap: 1.5`
- `rangefilter.threshold: 0.03` -> `0.05`
- `dynamicthreshold.min: 0.4` -> `0.5`
- `dynamicthreshold.validhours: 24` -> `48`
- `sentry.enabled: true` -> `false`
- `northern mockingbird.threshold: 0.35` -> `0.45`
- New `purple martin` species config block
- No other unexpected changes

**Step 2: Restart the service**

```bash
ssh jeremy@192.168.1.197 "sudo systemctl restart birdnet-go-native.service && sleep 5 && systemctl is-active birdnet-go-native.service"
```

Expected: `active`

**Step 3: Verify audio is flowing**

```bash
ssh jeremy@192.168.1.197 "journalctl -u birdnet-go-native.service --since '1 minute ago' --no-pager | tail -10"
```

Expected: Sound level and analysis log lines, no errors.

**Step 4: Verify API reports healthy state**

```bash
ssh jeremy@192.168.1.197 "curl -s http://localhost:8080/api/v2/system/audio/active | python3 -c 'import json,sys; d=json.load(sys.stdin); print(f\"verified={d[\"verified\"]} device={d[\"device\"][\"name\"]}\")'"
```

Expected: `verified=True device=BOYA Magic, USB Audio`

**Step 5: Check for any error logs**

```bash
ssh jeremy@192.168.1.197 "journalctl -u birdnet-go-native.service --since '2 minutes ago' --no-pager --priority=err"
```

Expected: No output (no errors).

---

## Rollback Procedure

If detections degrade significantly after these changes:

```bash
# Restore the backup config
ssh jeremy@192.168.1.197 "cp /home/jeremy/.config/birdnet-go/config.yaml.bak.2026-02-10 /home/jeremy/.config/birdnet-go/config.yaml && sudo systemctl restart birdnet-go-native.service"
```

## Success Criteria

After 24-48 hours with the new config:
1. Detection count per day should remain roughly the same (within 20% of baseline)
2. Average confidence should increase (fewer low-confidence detections)
3. No Purple Martin detections below 0.85 confidence
4. CPU usage should be slightly lower (reduced overlap + no debug logging)
5. No error logs in journalctl
