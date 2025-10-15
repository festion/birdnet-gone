# MediaMTX Configuration Examples

This directory contains example configuration files for using MediaMTX as an RTSP bridge between ESP32 microphones and BirdNET-Go.

## Files

- **mediamtx.yml** - MediaMTX server configuration
- **mediamtx.service** - Systemd service for MediaMTX
- **birdnet-go-rtsp.service** - BirdNET-Go systemd service with MediaMTX dependency
- **config.yaml.example** - BirdNET-Go configuration snippet for RTSP

## Quick Start

1. Install MediaMTX (see [ESP32-MediaMTX Setup Guide](../../doc/wiki/esp32-mediamtx-setup.md))
2. Copy and configure `mediamtx.yml` to `/etc/mediamtx.yml`
3. Install `mediamtx.service` to `/etc/systemd/system/`
4. Update your BirdNET-Go configuration with RTSP settings
5. Restart services

## Use Case

Use this setup when:
- Your microphone streams raw PCM audio over HTTP (like ESP32-S3)
- BirdNET-Go requires RTSP format
- You need audio format conversion (sample rate, channels)
- You want a stable, production-ready audio pipeline

## Documentation

See the complete guide: [ESP32-MediaMTX Setup](../../doc/wiki/esp32-mediamtx-setup.md)
