# ESP32 Microphone with MediaMTX RTSP Server Setup

This guide describes how to configure BirdNET-Go to receive audio from an ESP32-S3 microphone using MediaMTX as an RTSP intermediary server.

## Overview

ESP32 microcontrollers can stream raw PCM audio over HTTP, but BirdNET-Go requires RTSP streams. MediaMTX acts as a bridge, converting the HTTP raw PCM stream to RTSP format that BirdNET-Go can consume.

### Architecture

```
ESP32-S3 Microphone (HTTP Raw PCM)
  ↓ http://ESP32_IP:8080/stream
MediaMTX RTSP Server (FFmpeg conversion)
  ↓ rtsp://localhost:8554/birdnet_audio
BirdNET-Go (Realtime Analysis)
  ↓ Bird Detections
```

## Prerequisites

- ESP32-S3 microphone streaming raw PCM audio over HTTP
- BirdNET-Go installed and running
- Root access to the system running BirdNET-Go

## Step 1: Install MediaMTX

MediaMTX (formerly rtsp-simple-server) is a ready-to-use RTSP server.

### Download and Install

```bash
# For ARM64 (Raspberry Pi 4/5, etc.)
wget https://github.com/bluenviron/mediamtx/releases/download/v1.15.2/mediamtx_v1.15.2_linux_arm64.tar.gz
tar -xzf mediamtx_v1.15.2_linux_arm64.tar.gz
sudo mv mediamtx /usr/local/bin/
sudo chmod +x /usr/local/bin/mediamtx

# For AMD64 (x86_64 systems)
wget https://github.com/bluenviron/mediamtx/releases/download/v1.15.2/mediamtx_v1.15.2_linux_amd64.tar.gz
tar -xzf mediamtx_v1.15.2_linux_amd64.tar.gz
sudo mv mediamtx /usr/local/bin/
sudo chmod +x /usr/local/bin/mediamtx
```

## Step 2: Configure MediaMTX

Create the MediaMTX configuration file at `/etc/mediamtx.yml`:

```yaml
# MediaMTX Configuration for ESP32 Audio Streaming
logLevel: info
logDestinations: [stdout]
logFile: /tmp/mediamtx.log

# RTSP server configuration
rtspAddress: :8554
protocols: [tcp]

# Path for ESP32 audio stream
paths:
  birdnet_audio:
    runOnInit: ffmpeg -nostdin -hide_banner -f s16le -ar 16000 -ac 1 -i http://ESP32_IP:8080/stream -f rtsp -rtsp_transport tcp -ar 48000 -ac 2 -acodec pcm_s16be rtsp://localhost:8554/birdnet_audio
    runOnInitRestart: yes
```

**Important:** Replace `ESP32_IP` with your ESP32's actual IP address.

### Configuration Explanation

- **Input Format** (`-f s16le -ar 16000 -ac 1`): Raw PCM, 16kHz, mono (typical ESP32 output)
- **Output Format** (`-ar 48000 -ac 2 -acodec pcm_s16be`): 48kHz, stereo PCM (BirdNET-compatible)
- **runOnInitRestart: yes**: Automatically restart FFmpeg if it crashes

## Step 3: Create MediaMTX Systemd Service

Create the service file at `/etc/systemd/system/mediamtx.service`:

```ini
[Unit]
Description=MediaMTX RTSP Server for BirdNET Audio
Documentation=https://github.com/bluenviron/mediamtx
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/root
ExecStart=/usr/local/bin/mediamtx /etc/mediamtx.yml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable mediamtx.service
sudo systemctl start mediamtx.service
sudo systemctl status mediamtx.service
```

## Step 4: Configure BirdNET-Go

### Update BirdNET-Go Configuration

Edit your BirdNET-Go configuration file (typically `/root/birdnet-go-app/config/config.yaml`):

```yaml
realtime:
  audio:
    source: ""  # Leave empty, RTSP configured below

  rtsp:
    transport: tcp
    urls:
      - rtsp://localhost:8554/birdnet_audio
    health:
      healthydatathreshold: 60
      monitoringinterval: 30
```

### Update BirdNET-Go Systemd Service

If using Docker, update `/etc/systemd/system/birdnet-go.service`:

```ini
[Unit]
Description=BirdNET-Go
After=docker.service mediamtx.service
Requires=docker.service mediamtx.service

[Service]
Restart=always
ExecStartPre=-/usr/bin/docker rm -f birdnet-go
ExecStart=/usr/bin/docker run --rm --name birdnet-go \
  --network host \
  --env TZ="America/New_York" \
  --device /dev/snd \
  -v /root/birdnet-go-app/config:/config \
  -v /root/birdnet-go-app/data:/data \
  ghcr.io/tphakala/birdnet-go:nightly
ExecStopPost=-/usr/bin/docker rm -f birdnet-go

[Install]
WantedBy=multi-user.target
```

Note the `After=mediamtx.service` and `Requires=mediamtx.service` lines ensure MediaMTX starts before BirdNET-Go.

Restart BirdNET-Go:

```bash
sudo systemctl daemon-reload
sudo systemctl restart birdnet-go.service
```

## Step 5: Verify Setup

### Check MediaMTX Status

```bash
sudo systemctl status mediamtx.service
```

Expected output should show:
```
● mediamtx.service - MediaMTX RTSP Server for BirdNET Audio
   Active: active (running)

Oct 15 06:05:06 hostname mediamtx[3465]: 2025/10/15 06:05:06 INF [RTSP] [session xxx] is publishing to path 'birdnet_audio', 1 track (LPCM)
```

### Check BirdNET-Go Logs

```bash
sudo docker logs birdnet-go
```

Expected output should show:
```
✅ Started FFmpeg stream for rtsp://localhost:8554/birdnet_audio
✅ Stream rtsp://localhost:8554/birdnet_audio is healthy and receiving data (88.0 KB/s)
```

### Test RTSP Stream with VLC

```bash
vlc rtsp://localhost:8554/birdnet_audio
```

You should hear audio from your ESP32 microphone.

## Troubleshooting

### No Audio / Stream Not Working

1. **Verify ESP32 is streaming:**
   ```bash
   curl -I http://ESP32_IP:8080/stream
   ```
   Should return HTTP 200.

2. **Check MediaMTX logs:**
   ```bash
   sudo journalctl -u mediamtx.service -n 50
   ```
   Look for FFmpeg errors.

3. **Verify FFmpeg can read ESP32 stream:**
   ```bash
   ffmpeg -f s16le -ar 16000 -ac 1 -i http://ESP32_IP:8080/stream -t 5 test.wav
   ```
   Should create a 5-second audio file.

### BirdNET-Go Shows "Invalid or corrupted stream data"

This usually means:
- Wrong audio format specified in MediaMTX config
- ESP32 stopped streaming
- Network connectivity issue

Verify MediaMTX is publishing:
```bash
sudo journalctl -u mediamtx.service | grep "is publishing"
```

### High CPU Usage

If FFmpeg uses too much CPU, reduce the output sample rate:
```yaml
# In /etc/mediamtx.yml, change -ar 48000 to -ar 16000
runOnInit: ffmpeg -nostdin -hide_banner -f s16le -ar 16000 -ac 1 -i http://ESP32_IP:8080/stream -f rtsp -rtsp_transport tcp -ar 16000 -ac 1 -acodec pcm_s16be rtsp://localhost:8554/birdnet_audio
```

## Performance Considerations

- **Network Bandwidth**: ~96 KB/s (768 kbps) for 48kHz stereo stream
- **CPU Usage**: ~5-10% on Raspberry Pi 4 for FFmpeg conversion
- **Latency**: <100ms typical

## Alternative: Direct RTSP from ESP32

If your ESP32 has enough resources, you can skip MediaMTX by streaming RTSP directly from the ESP32 using libraries like Micro-RTSP-Audio. However, MediaMTX provides better stability and easier troubleshooting.

## See Also

- [RTSP Troubleshooting Guide](rtsp-troubleshooting.md)
- [Hardware Recommendations](hardware.md)
- [MediaMTX Documentation](https://github.com/bluenviron/mediamtx)
