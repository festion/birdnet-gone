# ESP32-CAM Wireless Microphone - Quick Start Guide

Get your ESP32-CAM streaming audio to BirdNET-Gone in 30 minutes!

## ⚡ Prerequisites

- ✅ ESP32-CAM board
- ✅ ESP32-CAM-MB programmer board
- ✅ INMP441 I2S microphone module
- ✅ 6 jumper wires (10cm) or soldering supplies
- ✅ USB-A to USB-C cable
- ✅ Computer with PlatformIO installed

## 🔧 Step 1: Hardware Assembly (10 minutes)

### Wiring

Connect microphone to ESP32-CAM using this table:

| ESP32-CAM | Wire Color | INMP441 |
|-----------|------------|---------|
| 3V3       | Red        | VDD     |
| GND       | Black      | GND     |
| GND       | Black      | L/R     |
| GPIO14    | Yellow     | WS      |
| GPIO15    | Green      | SCK     |
| GPIO13    | Blue       | SD      |

**Tip:** If using DuPont jumper wires, no soldering needed for initial testing!

### Insert into Programmer

1. Orient ESP32-CAM with antenna facing away from USB port
2. Align pins carefully and press firmly into ESP32-CAM-MB
3. Connect USB cable to programmer board
4. Computer should detect CH340 USB serial device

## 💻 Step 2: Software Configuration (5 minutes)

### Edit WiFi Credentials

```bash
cd /home/dev/workspace/birdnet-gone/firmware/esp32-cam-microphone
nano src/main.cpp
```

Update lines 53-54:
```cpp
const char* WIFI_SSID = "YourActualWiFiName";
const char* WIFI_PASSWORD = "YourActualPassword";
```

Save and exit (Ctrl+X, Y, Enter)

### Optional: Set Static IP

Uncomment lines 57-61 in `src/main.cpp`:
```cpp
IPAddress local_IP(192, 168, 1, 210);     // Choose available IP
IPAddress gateway(192, 168, 1, 1);        // Your router IP
IPAddress subnet(255, 255, 255, 0);       // Standard subnet
IPAddress primaryDNS(8, 8, 8, 8);         // Google DNS
IPAddress secondaryDNS(8, 8, 4, 4);       // Google DNS
```

## 🚀 Step 3: Build and Flash (5 minutes)

```bash
# Build firmware
pio run

# Flash to ESP32-CAM
pio run --target upload

# Monitor serial output
pio device monitor --baud 115200
```

**Expected output:**
```
========================================
  BirdNET-Gone ESP32-CAM Microphone
  Version 1.0.0
========================================

[I2S] Testing microphone...
[I2S] ✅ Microphone is working!

[WiFi] Connecting to WiFi...
[OK] WiFi connected!
IP address: 192.168.1.210

System Ready!
Stream URL: http://192.168.1.210:8080/stream
========================================
```

## ✅ Step 4: Verify Operation (5 minutes)

### Test 1: Web Interface

Open browser to: `http://192.168.1.210:8080`

Should show device status page with:
- Active clients: 0
- Uptime: increasing
- WiFi RSSI: -30 to -70 dBm

### Test 2: Audio Stream

```bash
# Download 10 seconds of audio
timeout 10 curl http://192.168.1.210:8080/stream > test.wav

# Check file size (should be ~320KB)
ls -lh test.wav

# Play audio
ffplay test.wav
```

**What you should hear:**
- Room ambient noise
- Your voice if you speak
- Environmental sounds

**What indicates problems:**
- Silence = wiring issue
- Loud hum = power supply noise
- Clicking/popping = I2S configuration issue

### Test 3: Status API

```bash
curl http://192.168.1.210:8080/status | jq '.'
```

Expected output:
```json
{
  "activeClients": 0,
  "totalConnections": 1,
  "uptime": 127,
  "freeHeap": 234896,
  "wifiRSSI": -52,
  "sampleRate": 16000,
  "channels": 1
}
```

## 🎯 Step 5: BirdNET-Gone Integration (5 minutes)

### Configure BirdNET-Go

```bash
# SSH to Raspberry Pi
ssh jeremy@192.168.1.197

# Edit configuration
echo 'redflower805' | sudo -S nano /root/birdnet-go-app/config/config.yaml
```

Update audio source:
```yaml
realtime:
  audio:
    source: http://192.168.1.210:8080/stream
```

Save and restart:
```bash
echo 'redflower805' | sudo -S systemctl restart birdnet-go.service
```

### Verify Connection

```bash
# Check BirdNET-Go logs
docker logs birdnet-go --tail 50

# Should see:
# "Starting analyzer in realtime mode"
# "Audio source: http://192.168.1.210:8080/stream"
```

On ESP32-CAM serial monitor:
```
[HTTP] New streaming client #1 (active: 1, total: 1)
```

### Test Detection

Wait 15-30 seconds, then check for birds:
```bash
curl http://192.168.1.197:8080/api/v2/detections/recent | jq '.[] | {species: .commonName, confidence: .confidence}'
```

**Note:** Detections require actual bird sounds. Play bird calls near microphone or wait for real birds!

## 🎉 Success Criteria

You're done when:
- ✅ ESP32-CAM connects to WiFi (shows IP address)
- ✅ Microphone test passes (>90% non-zero samples)
- ✅ Web interface loads (http://ESP32-IP:8080)
- ✅ Audio stream downloads and plays real sound
- ✅ BirdNET-Go connects (activeClients: 1)
- ✅ Bird detections appear (when birds sing)

## 🐛 Common Issues

### Problem: WiFi won't connect

**Check:**
1. SSID and password are correct (case-sensitive!)
2. WiFi is 2.4GHz (not 5GHz - ESP32 doesn't support)
3. Router allows new device connections
4. Signal strength (move closer to router)

**Serial monitor shows:**
```
[WiFi] Connecting to WiFi...
....................
[ERROR] WiFi connection failed!
```

**Fix:** Double-check credentials in `src/main.cpp` lines 53-54

---

### Problem: Microphone silent (>99% zeros)

**Serial monitor shows:**
```
[I2S Test] Attempt 1: 256 samples, 2 non-zero (0.8%), range: -1 to 1
[WARN] Microphone appears silent (>99% zeros)
```

**Check:**
1. All 6 wires connected (especially GND and VDD)
2. L/R pin connected to GND or VDD (not floating)
3. Microphone facing outward (not blocked)
4. Speak loudly near microphone during test

**Advanced fix:** Try pin swap (some modules have mislabeled pins)
```cpp
// In src/main.cpp, swap lines 68-70:
#define I2S_WS_PIN      13  // Was 14
#define I2S_SD_PIN      14  // Was 13
```

---

### Problem: Upload failed

**Error:**
```
serial.serialutil.SerialException: [Errno 2] could not open port /dev/ttyUSB0
```

**Fix:**
1. Ensure ESP32-CAM is **fully inserted** into ESP32-CAM-MB
2. Press **Reset button** on programmer board
3. Try different USB cable (some are power-only)
4. Check USB device appears: `ls -l /dev/ttyUSB* /dev/ttyACM*`

---

### Problem: BirdNET-Go can't connect to stream

**BirdNET-Go logs show:**
```
[ERROR] Failed to connect to audio source
```

**Check:**
1. ESP32-CAM is reachable from Pi:
   ```bash
   ssh jeremy@192.168.1.197 "ping -c 3 192.168.1.210"
   ```

2. Stream URL is correct in config:
   ```bash
   ssh jeremy@192.168.1.197 "sudo cat /root/birdnet-go-app/config/config.yaml | grep source"
   ```

3. Test stream from Pi:
   ```bash
   ssh jeremy@192.168.1.197 "curl --max-time 5 http://192.168.1.210:8080/status"
   ```

---

### Problem: No bird detections

**This is normal if:**
- No birds are singing nearby
- Confidence threshold too high (>0.8)
- Wrong time of day (birds most active at dawn/dusk)

**Test with known audio:**
1. Download bird call MP3 (Northern Mockingbird, American Robin)
2. Play near ESP32-CAM microphone
3. Wait 30 seconds
4. Check detections: `curl http://192.168.1.197:8080/api/v2/detections/recent`

**Adjust sensitivity** in BirdNET-Go config:
```yaml
birdnet:
  threshold: 0.3  # Lower = more detections (0.0-1.0)
```

## 📖 Next Steps

**Optional Enhancements:**

1. **Weatherproof Enclosure** - See [HARDWARE.md](HARDWARE.md) for outdoor installation
2. **Static IP via DHCP** - Configure router to always assign same IP
3. **External Antenna** - Improve WiFi range for outdoor placement
4. **Solar Power** - Battery + solar panel for off-grid operation
5. **Multiple Microphones** - Deploy several ESP32-CAMs for different locations

**Advanced Configuration:**

- **Adjust sample rate** - 16kHz (default), 32kHz, or 48kHz
- **Change I2S pins** - If conflicts with other hardware
- **Add authentication** - HTTP Basic Auth for security
- **Enable HTTPS** - TLS encryption (requires certificate)

## 📚 Documentation

- **[README.md](README.md)** - Complete documentation
- **[HARDWARE.md](HARDWARE.md)** - Detailed hardware guide
- **[src/main.cpp](src/main.cpp)** - Firmware source code (commented)

## ❓ Getting Help

**Check logs:**
```bash
# ESP32-CAM serial monitor
pio device monitor --baud 115200

# BirdNET-Go logs
ssh jeremy@192.168.1.197 "docker logs birdnet-go --tail 100"

# BirdNET Display logs
ssh jeremy@192.168.1.197 "journalctl -u bird-display.service -n 50"
```

**Test connectivity:**
```bash
# Ping ESP32-CAM
ping 192.168.1.210

# Test web interface
curl http://192.168.1.210:8080

# Test audio stream
timeout 5 curl http://192.168.1.210:8080/stream | head -c 10000
```

**Still stuck?**
1. Re-read troubleshooting in [README.md](README.md)
2. Check BirdNET-Go GitHub issues
3. Review [HARDWARE.md](HARDWARE.md) wiring diagrams
4. Create GitHub issue with serial monitor output

---

**Total time:** 30 minutes
**Difficulty:** ⭐⭐⚪⚪⚪ (Beginner-Intermediate)
**Cost:** ~$24 (if buying all new components)

**Happy birding! 🐦🎤**
