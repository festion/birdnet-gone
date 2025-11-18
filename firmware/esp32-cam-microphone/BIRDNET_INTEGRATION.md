# BirdNET-Go Integration with ESP32-CAM Microphone

**Date:** November 18, 2025
**ESP32-CAM IP:** 192.168.1.212
**BirdNET-Go Host:** 192.168.1.197
**Status:** Ready for integration

---

## ESP32-CAM Status

✅ **Firmware flashed successfully**
✅ **Device online at 192.168.1.212:8080**
✅ **Audio stream verified** (WAV, 16kHz, 16-bit, mono)
✅ **WiFi connected** to "Lakehouse" network
✅ **Signal strength:** -61 dBm (Good)

### Verified Endpoints

```bash
# Web interface (status dashboard)
curl http://192.168.1.212:8080/

# JSON status API
curl http://192.168.1.212:8080/status
# Response: {"activeClients":2,"totalConnections":2,"uptime":283,"freeHeap":222172,"wifiRSSI":-60,"sampleRate":16000,"channels":1}

# Audio stream (continuous WAV)
curl http://192.168.1.212:8080/stream > test.wav
```

---

## Integration Steps

### 1. SSH to BirdNET-Go Raspberry Pi

```bash
ssh jeremy@192.168.1.197
# Password: redflower805
```

### 2. Backup Current Configuration

```bash
echo 'redflower805' | sudo -S cp /root/birdnet-go-app/config/config.yaml /root/birdnet-go-app/config/config.yaml.backup
```

### 3. Update Audio Source Configuration

```bash
# Edit the config file
echo 'redflower805' | sudo -S nano /root/birdnet-go-app/config/config.yaml
```

**Find and update the audio source section** (typically around line 75):

```yaml
realtime:
  audio:
    # OLD (USB microphone or ALSA):
    # source: hw:3,0

    # NEW (ESP32-CAM wireless microphone):
    source: http://192.168.1.212:8080/stream

    # Keep these settings unchanged:
    # - Sample rate should remain 16000 Hz (matches ESP32-CAM)
    # - Channels should remain 1 (mono)
```

**Complete realtime section should look like:**

```yaml
realtime:
  audio:
    source: http://192.168.1.212:8080/stream
  processingtime: true
  interval: 15
```

### 4. Restart BirdNET-Go Service

```bash
# Restart the service
echo 'redflower805' | sudo -S systemctl restart birdnet-go.service

# Check service status
echo 'redflower805' | sudo -S systemctl status birdnet-go.service

# Expected output:
# ● birdnet-go.service - BirdNET-Go
#      Loaded: loaded
#      Active: active (running)
```

### 5. Verify Integration

```bash
# Monitor BirdNET-Go logs (press Ctrl+C to exit)
echo 'redflower805' | sudo -S journalctl -u birdnet-go.service -f

# Expected log messages:
# "Audio source: http://192.168.1.212:8080/stream"
# "Connected to audio stream"
# "Analyzing audio..."
# "Detection: [species name] (confidence: XX%)"
```

### 6. Test from BirdNET-Go Web Interface

Open browser to: `http://192.168.1.197:8080` (or whatever port BirdNET-Go uses)

**Expected results:**
- Dashboard should show audio input active
- Spectrogram should display audio waveforms
- Detections should start appearing within minutes (depending on bird activity)

---

## Verification Checklist

After integration, verify:

- [ ] BirdNET-Go service is running (`systemctl status birdnet-go.service`)
- [ ] Logs show successful connection to ESP32-CAM stream
- [ ] Web dashboard shows audio input is active
- [ ] Spectrogram displays audio data
- [ ] ESP32-CAM status shows active client: `curl http://192.168.1.212:8080/status`
- [ ] No error messages in BirdNET-Go logs
- [ ] Detections appear within 15-30 minutes (if birds are present)

---

## Troubleshooting

### Issue: BirdNET-Go Can't Connect to ESP32-CAM

**Symptoms:** Logs show "Failed to connect to audio stream" or "Connection refused"

**Solutions:**

1. **Verify ESP32-CAM is online:**
   ```bash
   ping 192.168.1.212
   curl -I http://192.168.1.212:8080/stream
   ```

2. **Check firewall rules on Raspberry Pi:**
   ```bash
   echo 'redflower805' | sudo -S iptables -L
   # Should allow outbound HTTP connections
   ```

3. **Test stream from Raspberry Pi:**
   ```bash
   # On the Pi, test the stream directly
   timeout 10 curl http://192.168.1.212:8080/stream > /tmp/test.wav
   file /tmp/test.wav
   # Expected: /tmp/test.wav: RIFF (little-endian) data, WAVE audio
   ```

4. **Verify correct URL in config:**
   - Must include `/stream` path: `http://192.168.1.212:8080/stream`
   - NOT just `http://192.168.1.212:8080`

### Issue: No Audio Data / Silent Stream

**Symptoms:** Stream connects but no detections, spectrogram is flat

**Solutions:**

1. **Check ESP32-CAM microphone connection:**
   - Verify all 6 INMP441 wires are connected
   - L/R pin must be connected to GND (for left channel)

2. **Test with known loud sound:**
   - Clap near microphone
   - Play bird sounds on phone speaker near microphone
   - Check ESP32-CAM web dashboard for active clients

3. **Monitor ESP32-CAM serial output:**
   ```bash
   # On dev machine where ESP32-CAM is connected via USB
   pio device monitor
   # Look for I2S errors or audio buffer issues
   ```

### Issue: Frequent Disconnections

**Symptoms:** Stream works but drops every few minutes

**Solutions:**

1. **Check ESP32-CAM WiFi signal:**
   ```bash
   curl http://192.168.1.212:8080/status | jq .wifiRSSI
   # Should be > -70 dBm (more negative = weaker)
   ```

2. **Monitor ESP32-CAM uptime:**
   ```bash
   watch -n 5 'curl -s http://192.168.1.212:8080/status | jq .uptime'
   # If uptime resets, device is rebooting
   ```

3. **Check ESP32-CAM power supply:**
   - Minimum 5V @ 400mA required
   - Use quality USB power adapter
   - Avoid long/thin USB cables

4. **Move ESP32-CAM closer to WiFi access point**
   - Or add external antenna if ESP32-CAM has U.FL connector

### Issue: High CPU Usage on Raspberry Pi

**Symptoms:** BirdNET-Go using 90%+ CPU constantly

**Solutions:**

1. **Verify audio format matches expectations:**
   - ESP32-CAM streams 16kHz, 16-bit, mono
   - BirdNET-Go expects same format (no resampling needed)

2. **Check for multiple BirdNET-Go processes:**
   ```bash
   ps aux | grep birdnet
   # Should show only one process
   ```

3. **Monitor system resources:**
   ```bash
   htop
   # Check CPU, memory, and temperature
   ```

---

## Performance Expectations

### ESP32-CAM

- **Uptime:** Days to weeks (unless power cycled)
- **WiFi Connection:** Stable with RSSI > -70 dBm
- **Active Clients:** Typically 1 (BirdNET-Go)
- **Memory Usage:** ~220 KB free heap (stable)
- **Audio Latency:** <100ms (audio capture → network)

### BirdNET-Go

- **Detection Interval:** Every 15 seconds (configurable)
- **CPU Usage:** 30-60% during analysis (Raspberry Pi 4)
- **Detection Delay:** 0-15 seconds (based on interval)
- **Confidence Threshold:** Typically 0.7-0.9 (configurable)

### Network

- **Bandwidth:** ~32 KB/s continuous (256 kbps)
- **Latency:** <50ms on local network
- **Reliability:** No packet loss with good WiFi signal

---

## Reverting to Previous Configuration

If you need to switch back to the original microphone:

```bash
# Restore backup config
echo 'redflower805' | sudo -S cp /root/birdnet-go-app/config/config.yaml.backup /root/birdnet-go-app/config/config.yaml

# Restart service
echo 'redflower805' | sudo -S systemctl restart birdnet-go.service
```

---

## Additional Configuration Options

### Static IP Reservation (Optional)

The ESP32-CAM is configured with static IP 192.168.1.212 in firmware. To add a DHCP reservation as backup:

1. Log into router admin interface
2. Find DHCP settings
3. Add reservation:
   - **IP Address:** 192.168.1.212
   - **MAC Address:** (check ESP32-CAM web dashboard or router DHCP leases)
   - **Hostname:** esp32-cam-microphone

### Multiple Microphones (Future)

To use multiple ESP32-CAM microphones:

1. Flash additional ESP32-CAM devices with different static IPs
2. Configure BirdNET-Go to use multiple audio sources (if supported)
3. Or run multiple BirdNET-Go instances (one per microphone)

---

## Support Files

- **Hardware Wiring:** `HARDWARE.md`
- **Firmware Documentation:** `DEPLOYMENT_READY.md`
- **Troubleshooting:** `README.md` (ESP32-CAM section)
- **Source Code:** `src/main.cpp`

---

**Integration Status:** Ready to deploy
**Last Updated:** November 18, 2025
**Next Step:** Update BirdNET-Go config and restart service
