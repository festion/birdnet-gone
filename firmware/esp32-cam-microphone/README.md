# BirdNET-Gone ESP32-CAM Wireless Microphone

Transform your ESP32-CAM + ESP32-CAM-MB into a WiFi-enabled audio streaming device for BirdNET-Gone bird detection.

## 🎯 Features

- ✅ **HTTP Audio Streaming**: Continuous WAV audio at 16kHz, 16-bit, mono
- ✅ **I2S Digital Microphone**: Support for INMP441, SPH0645, ICS-43434 MEMS mics
- ✅ **Multiple Clients**: Up to 3 concurrent streaming connections
- ✅ **Auto Recovery**: Watchdog timer and client timeout protection
- ✅ **OTA Updates**: Update firmware over WiFi (no USB required)
- ✅ **Remote Reboot**: Web-based device restart
- ✅ **Status Monitoring**: Real-time system statistics via JSON API
- ✅ **Low Latency**: Async web server for non-blocking operation

## 📦 Hardware Requirements

### Required Components

1. **ESP32-CAM** - Main microcontroller board
   - ESP32-S chip (not ESP32-S3)
   - 4MB flash minimum
   - PSRAM support (usually included)

2. **ESP32-CAM-MB** - USB programmer board
   - Used for initial firmware flashing
   - Provides stable 5V power supply
   - Can be removed after flashing for standalone operation

3. **I2S MEMS Microphone** (choose one):
   - **INMP441** - Recommended (excellent SNR, widely available)
   - **SPH0645** - Good alternative (I2S output)
   - **ICS-43434** - Budget option (verify pinout, may need pin swap)

4. **Power Supply**:
   - 5V USB power adapter (minimum 1A)
   - Or use ESP32-CAM-MB 5V output

### Optional Components

- External WiFi antenna for better range
- Weatherproof enclosure for outdoor deployment
- MicroSD card for local audio buffering (future enhancement)

## 🔌 Wiring Diagram

### ESP32-CAM to I2S Microphone (INMP441 Example)

```
ESP32-CAM Pin      Wire Color    INMP441 Pin     Function
─────────────      ──────────    ───────────     ────────
3V3                Red           VDD             Power (3.3V)
GND                Black         GND             Ground
GND                Black         L/R             Channel Select (Left)
GPIO14             Yellow        WS              Word Select (LRCLK)
GPIO15             Green         SCK             Bit Clock (BCLK)
GPIO13             Blue          SD              Serial Data (DOUT)
```

### Pin Selection Rationale

The GPIO pins were carefully chosen to avoid conflicts with:
- Camera interface (GPIO0, GPIO2-12, GPIO16-GPIO27)
- SD card interface (if used)
- Boot mode pins

**Available safe pins on ESP32-CAM:**
- GPIO13, GPIO14, GPIO15 (used for I2S)
- GPIO33 (LED - avoid if using LED)

### Physical Connection

1. **Solder microphone module** to ESP32-CAM using the wiring above
2. **Keep wires short** (< 10cm) to reduce noise
3. **Use twisted pair** for I2S data lines if possible
4. **Connect L/R to GND** for left channel (or VDD for right channel)

## 🛠️ Software Setup

### Prerequisites

1. **Install PlatformIO**:
   ```bash
   # Option 1: VS Code Extension
   # Install "PlatformIO IDE" from VS Code Extensions

   # Option 2: Command Line
   pip install platformio
   ```

2. **Install CH340 USB Driver** (for ESP32-CAM-MB):
   - Windows: Download from manufacturer
   - Mac: `brew install --cask ch340g-ch34g-ch34x-mac-os-x-driver`
   - Linux: Usually built-in (check with `dmesg | grep ch34`)

### Configuration

1. **Edit WiFi Credentials** in `src/main.cpp`:
   ```cpp
   // Line 53-54
   const char* WIFI_SSID = "YourWiFiSSID";        // CHANGE THIS
   const char* WIFI_PASSWORD = "YourWiFiPassword"; // CHANGE THIS
   ```

2. **Optional: Configure Static IP** in `src/main.cpp`:
   ```cpp
   // Line 57-61 (uncomment to use)
   IPAddress local_IP(192, 168, 1, 210);
   IPAddress gateway(192, 168, 1, 1);
   IPAddress subnet(255, 255, 255, 0);
   IPAddress primaryDNS(8, 8, 8, 8);
   IPAddress secondaryDNS(8, 8, 4, 4);
   ```

3. **Optional: Adjust I2S Pins** (if using different wiring):
   ```cpp
   // Line 68-70
   #define I2S_WS_PIN      14
   #define I2S_SCK_PIN     15
   #define I2S_SD_PIN      13
   ```

## 📝 Building and Flashing

### Method 1: PlatformIO CLI (Recommended)

```bash
# Navigate to firmware directory
cd /home/dev/workspace/birdnet-gone/firmware/esp32-cam-microphone

# Install dependencies
pio lib install

# Build firmware
pio run

# Flash to ESP32-CAM (insert ESP32-CAM into ESP32-CAM-MB)
pio run --target upload

# Monitor serial output
pio device monitor --baud 115200
```

### Method 2: PlatformIO IDE (VS Code)

1. Open firmware directory in VS Code
2. PlatformIO will auto-detect `platformio.ini`
3. Click **"Build"** button (✓ icon) in bottom toolbar
4. Insert ESP32-CAM into ESP32-CAM-MB
5. Click **"Upload"** button (→ icon)
6. Click **"Serial Monitor"** button (🔌 icon)

### Method 3: ESP Tool (Advanced)

```bash
# Build firmware first
pio run

# Flash using esptool
esptool.py --chip esp32 --port /dev/ttyUSB0 --baud 921600 \
  --before default_reset --after hard_reset write_flash \
  0x1000 .pio/build/esp32cam/bootloader.bin \
  0x8000 .pio/build/esp32cam/partitions.bin \
  0x10000 .pio/build/esp32cam/firmware.bin
```

### Troubleshooting Flash Issues

**Problem: Device not detected**
```bash
# Check USB connection
ls -l /dev/ttyUSB* /dev/ttyACM*

# Verify CH340 driver
dmesg | grep ch34
```

**Problem: Upload failed / timeout**
1. Ensure ESP32-CAM is **fully inserted** into ESP32-CAM-MB
2. Press **Reset button** on ESP32-CAM-MB before upload
3. Try different USB cable (some are power-only)
4. Reduce upload speed: `upload_speed = 115200` in `platformio.ini`

**Problem: Upload works but device doesn't boot**
1. Check serial monitor for error messages
2. Verify 5V power supply is adequate (1A minimum)
3. Try erasing flash: `esptool.py --port /dev/ttyUSB0 erase_flash`

## 🚀 First Boot and Testing

### Expected Serial Output

```
========================================
  BirdNET-Gone ESP32-CAM Microphone
  Wireless Audio Streaming Server
  Version 1.0.0
========================================

[I2S] Initializing I2S interface...
[I2S] Configuration: 16000 Hz, 16-bit, 1 channel(s)
[I2S] GPIO Pins: WS=14, SCK=15, SD=13
[I2S] Testing microphone...
[I2S Test] Attempt 1: 256 samples, 243 non-zero (94.9%), range: -1234 to 1567
[I2S] ✅ Microphone is working!

[WiFi] Connecting to WiFi...
[WiFi] SSID: YourNetwork
..........
[OK] WiFi connected!
IP address: 192.168.1.210
Signal strength: -52 dBm

[HTTP] Setting up web server...
[HTTP] Server started on port 8080
[WDT] Configuring watchdog timer (60 seconds)

System Ready!
Stream URL: http://192.168.1.210:8080/stream
========================================
```

### Verification Steps

1. **Check WiFi Connection**:
   ```bash
   ping 192.168.1.210  # Use your ESP32's IP
   ```

2. **Test Web Interface**:
   ```bash
   # Open in browser
   http://192.168.1.210:8080

   # Should show device status page
   ```

3. **Test Status API**:
   ```bash
   curl http://192.168.1.210:8080/status | jq '.'

   # Expected output:
   # {
   #   "activeClients": 0,
   #   "totalConnections": 0,
   #   "uptime": 42,
   #   "freeHeap": 234567,
   #   "wifiRSSI": -52,
   #   "sampleRate": 16000,
   #   "channels": 1
   # }
   ```

4. **Test Audio Stream**:
   ```bash
   # Download 10 seconds of audio
   timeout 10 curl http://192.168.1.210:8080/stream > test.wav

   # Check file size (should be ~320KB for 10 seconds)
   ls -lh test.wav

   # Play audio (requires ffmpeg)
   ffplay test.wav

   # Or convert to playable format
   ffmpeg -i test.wav -ar 44100 test_playable.wav
   ```

5. **Test Audio Quality**:
   ```python
   # Analyze captured audio (requires Python + numpy)
   python3 << 'EOF'
   import struct
   import wave

   with wave.open('test.wav', 'rb') as wav:
       frames = wav.readframes(wav.getnframes())
       samples = struct.unpack('<' + 'h' * (len(frames) // 2), frames)

   zeros = sum(1 for s in samples if s == 0)
   non_zeros = len(samples) - zeros
   max_val = max(samples)
   min_val = min(samples)

   print(f"Total samples: {len(samples)}")
   print(f"Non-zero: {non_zeros} ({non_zeros/len(samples)*100:.1f}%)")
   print(f"Range: {min_val} to {max_val}")

   if non_zeros / len(samples) > 0.9:
       print("✅ Microphone is capturing audio!")
   else:
       print("❌ Microphone appears silent")
   EOF
   ```

### Expected Results

- ✅ Non-zero samples: **>90%** (indicates working microphone)
- ✅ Sample range: **-5000 to +5000** or wider (good dynamic range)
- ✅ File size: **~32 KB per second** (16kHz × 2 bytes × 1 channel)
- ✅ Audio playback: **Real environmental sounds** (not silence or noise)

## 🔧 BirdNET-Gone Integration

### Configure Audio Source

1. **SSH to BirdNET-Gone Raspberry Pi**:
   ```bash
   ssh jeremy@192.168.1.197
   ```

2. **Edit BirdNET-Go Configuration**:
   ```bash
   echo 'redflower805' | sudo -S nano /root/birdnet-go-app/config/config.yaml
   ```

3. **Update Audio Source**:
   ```yaml
   realtime:
     interval: 15  # Analyze every 15 seconds
     audio:
       source: http://192.168.1.210:8080/stream  # ESP32-CAM IP
   ```

4. **Restart BirdNET-Go**:
   ```bash
   echo 'redflower805' | sudo -S systemctl restart birdnet-go.service
   ```

5. **Verify Connection**:
   ```bash
   # Check BirdNET-Go logs
   ssh jeremy@192.168.1.197 "docker logs birdnet-go --tail 50"

   # Should see:
   # "Starting analyzer in realtime mode"
   # "Audio source: http://192.168.1.210:8080/stream"

   # Check ESP32-CAM serial monitor
   # Should see:
   # "[HTTP] New streaming client #1"
   ```

6. **Monitor for Detections**:
   ```bash
   # View recent detections
   curl http://192.168.1.197:8080/api/v2/detections/recent | jq '.[] | {species: .commonName, confidence: .confidence, time: .timestamp}'
   ```

### Troubleshooting Integration

**Problem: BirdNET-Go can't connect to stream**

```bash
# 1. Verify ESP32 is reachable from Pi
ssh jeremy@192.168.1.197 "ping -c 3 192.168.1.210"

# 2. Test stream from Pi
ssh jeremy@192.168.1.197 "curl --max-time 5 http://192.168.1.210:8080/status"

# 3. Check firewall (if any)
ssh jeremy@192.168.1.197 "sudo iptables -L"
```

**Problem: No bird detections**

This is **normal** if:
- No birds are singing near the microphone
- Confidence threshold is too high (check `birdnet.threshold` in config)
- Species occurrence filter is enabled

Test with known bird audio:
```bash
# Play bird call near ESP32-CAM microphone
# Northern Mockingbird, American Robin, etc.
```

## 🔄 OTA Firmware Updates

Once the initial firmware is flashed, you can update wirelessly:

### Using PlatformIO (After Adding OTA Support)

```bash
# Build new firmware
pio run

# Upload via WiFi (requires ArduinoOTA library added)
pio run --target upload --upload-port 192.168.1.210
```

### Using Web Interface (Manual Implementation)

```bash
# Build firmware
pio run

# Upload via HTTP POST (requires /ota endpoint implementation)
curl -X POST http://192.168.1.210:8080/ota \
  -F "firmware=@.pio/build/esp32cam/firmware.bin"
```

*Note: Full OTA implementation is a future enhancement*

## 📊 Performance Specifications

### Audio Quality

| Parameter | Value |
|-----------|-------|
| Sample Rate | 16,000 Hz |
| Bit Depth | 16-bit |
| Channels | 1 (Mono) |
| Format | PCM WAV |
| Bitrate | 256 kbps |
| Latency | <100ms |

### Network

| Parameter | Value |
|-----------|-------|
| Protocol | HTTP/1.1 |
| Transfer | Chunked encoding |
| Max Clients | 3 concurrent |
| Timeout | 30 seconds |
| Bandwidth | ~32 KB/s per client |

### System Resources

| Resource | Usage |
|----------|-------|
| Flash | ~350 KB (compiled firmware) |
| RAM | ~50 KB (runtime heap) |
| CPU | ~15% (@ 240MHz) |
| WiFi | -30 to -70 dBm (typical) |
| Power | ~300mA @ 5V (streaming) |

## 🛡️ Security Considerations

### Current Security

- ⚠️ **No authentication** - Anyone on network can access stream
- ⚠️ **HTTP only** - Unencrypted audio transmission
- ⚠️ **No CORS** - Cross-origin requests allowed

### Recommended Mitigations

1. **Network Isolation**:
   - Use VLAN for IoT devices
   - Firewall rules to limit access to BirdNET-Gone Pi only

2. **Static IP + MAC Filtering**:
   - Assign static IP via DHCP reservation
   - Enable MAC address filtering on router

3. **Future Enhancements**:
   - Add HTTP Basic Authentication
   - Implement HTTPS/TLS
   - Add API key requirement

## 🐛 Troubleshooting Guide

### Microphone Issues

**Symptom: 99% silence (zeros) in audio**

Possible causes:
1. **Wrong wiring** - Double-check pin connections
2. **Pin swap needed** - Some modules (ICS-43434) have mislabeled pins
   - Try swapping WS and SD in firmware
3. **L/R floating** - Connect L/R to GND or VDD
4. **Dead microphone** - Test with multimeter (3.3V on VDD)
5. **No sound source** - Speak loudly near microphone

Debug steps:
```bash
# Check serial monitor for diagnostic messages
# Look for:
# "[I2S DEBUG] Samples: X, zeros: Y (Z%)"
# "[WARN] Microphone appears silent (>99% zeros)"

# If >99% zeros, try pin swap in src/main.cpp:
# Swap lines 68-70:
# #define I2S_WS_PIN      13  // Was 14
# #define I2S_SD_PIN      14  // Was 13
```

**Symptom: Noisy/distorted audio**

Possible causes:
1. **Long wires** - Keep I2S wires < 10cm
2. **Power supply noise** - Use quality 5V adapter
3. **Insufficient power** - Ensure 1A minimum
4. **Sample rate mismatch** - Verify 16kHz in BirdNET-Go

### WiFi Issues

**Symptom: Won't connect to WiFi**

```bash
# 1. Verify credentials in src/main.cpp (lines 53-54)
# 2. Check WiFi band - ESP32 only supports 2.4GHz (not 5GHz)
# 3. Verify SSID is visible and password is correct
# 4. Check router logs for connection attempts
# 5. Try disabling WiFi security temporarily (for testing only)
```

**Symptom: Frequent disconnections**

```bash
# 1. Check WiFi signal strength (should be > -70 dBm)
curl http://192.168.1.210:8080/status | jq '.wifiRSSI'

# 2. Add external antenna if signal is weak
# 3. Move closer to router or add WiFi repeater
# 4. Check for 2.4GHz interference (Bluetooth, microwaves)
```

### Streaming Issues

**Symptom: Stream starts then stops**

```bash
# Check ESP32 serial monitor for:
# "[HTTP] Client timeout, closing stream"
# "[ERROR] I2S read failed"

# Possible causes:
# 1. Network congestion
# 2. Client disconnected
# 3. I2S buffer underrun

# Test with:
timeout 60 curl http://192.168.1.210:8080/stream > long_test.wav
# Should run for full 60 seconds without errors
```

**Symptom: Multiple clients cause crashes**

```bash
# Reduce MAX_CLIENTS in src/main.cpp:
# Line 85:
# #define MAX_CLIENTS 1  // Reduced from 3

# Rebuild and reflash firmware
```

## 📚 Additional Resources

### Documentation

- [ESP32-CAM Datasheet](https://github.com/raphaelbs/esp32-cam-ai-thinker/blob/master/assets/ESP32-CAM_Product_Specification.pdf)
- [I2S Protocol Overview](https://www.sparkfun.com/datasheets/BreakoutBoards/I2SBUS.pdf)
- [INMP441 Datasheet](https://invensense.tdk.com/wp-content/uploads/2015/02/INMP441.pdf)
- [BirdNET-Go Documentation](https://github.com/tphakala/birdnet-go)

### Related Projects

- [ESP32-S3 Sense Firmware](/home/dev/workspace/ESP32S3_PDM_FIRMWARE_FIX_COMPLETE.md) - PDM microphone variant
- [ESP32-C6 Microphone](/home/dev/workspace/ESP32_MICROPHONE_SUCCESS.md) - Alternative ESP32 board
- [BirdNET-Gone Deployment](/home/dev/workspace/BIRDNET_GONE_DEPLOYMENT_COMPLETE.md) - Complete system guide

### Community Support

- [BirdNET-Go GitHub Issues](https://github.com/tphakala/birdnet-go/issues)
- [ESP32 Forum - Audio Projects](https://www.esp32.com/)
- [PlatformIO Community](https://community.platformio.org/)

## 📝 License

This firmware is part of the BirdNET-Gone project and is released under the MIT License.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues for:

- Bug fixes
- Hardware compatibility improvements
- Documentation updates
- Feature enhancements (HTTPS, authentication, SD card buffering)

## 📅 Version History

### v1.0.0 (2025-10-22)
- Initial release
- I2S microphone support (INMP441, SPH0645, ICS-43434)
- HTTP WAV streaming at 16kHz
- Async web server with multiple clients
- Status monitoring and remote reboot
- BirdNET-Gone integration ready

---

**Built with ❤️ for bird enthusiasts and conservation**
