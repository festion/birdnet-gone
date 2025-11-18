#!/bin/bash
#
# ESP32-CAM Firmware Flashing Script
# This script flashes the compiled firmware to an ESP32-CAM device
#
# Usage:
#   ./flash.sh [port]
#
# Example:
#   ./flash.sh /dev/ttyUSB0
#

set -e  # Exit on error

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}ESP32-CAM Firmware Flash Tool${NC}"
echo -e "${BLUE}========================================${NC}"
echo

# Check for port argument
if [ -z "$1" ]; then
    echo -e "${YELLOW}No port specified. Detecting available ports...${NC}"
    echo

    # Auto-detect USB serial ports
    if [ -e /dev/ttyUSB0 ]; then
        PORT="/dev/ttyUSB0"
        echo -e "${GREEN}Found: $PORT${NC}"
    elif [ -e /dev/ttyACM0 ]; then
        PORT="/dev/ttyACM0"
        echo -e "${GREEN}Found: $PORT${NC}"
    else
        echo -e "${RED}ERROR: No USB serial device found!${NC}"
        echo
        echo "Available /dev/tty* devices:"
        ls -la /dev/tty* 2>/dev/null | grep -E "USB|ACM" || echo "  None found"
        echo
        echo "Usage: $0 <port>"
        echo "Example: $0 /dev/ttyUSB0"
        exit 1
    fi
else
    PORT="$1"
fi

# Verify port exists
if [ ! -e "$PORT" ]; then
    echo -e "${RED}ERROR: Port $PORT does not exist!${NC}"
    exit 1
fi

# Verify firmware exists
FIRMWARE_PATH=".pio/build/esp32cam/firmware.bin"
BOOTLOADER_PATH=".pio/build/esp32cam/bootloader.bin"
PARTITIONS_PATH=".pio/build/esp32cam/partitions.bin"

echo -e "${BLUE}Verifying firmware files...${NC}"
if [ ! -f "$FIRMWARE_PATH" ]; then
    echo -e "${RED}ERROR: Firmware not found at $FIRMWARE_PATH${NC}"
    echo "Run 'pio run' to build firmware first."
    exit 1
fi

if [ ! -f "$BOOTLOADER_PATH" ]; then
    echo -e "${RED}ERROR: Bootloader not found at $BOOTLOADER_PATH${NC}"
    exit 1
fi

if [ ! -f "$PARTITIONS_PATH" ]; then
    echo -e "${RED}ERROR: Partitions not found at $PARTITIONS_PATH${NC}"
    exit 1
fi

echo -e "${GREEN}✓ All firmware files present${NC}"
echo

# Display configuration
echo -e "${BLUE}Flash Configuration:${NC}"
echo "  Port:       $PORT"
echo "  Chip:       ESP32"
echo "  Baud:       921600"
echo "  Flash Mode: DIO"
echo "  Flash Freq: 80MHz"
echo "  Flash Size: 4MB"
echo

# Firmware details
FIRMWARE_SIZE=$(stat -f%z "$FIRMWARE_PATH" 2>/dev/null || stat -c%s "$FIRMWARE_PATH")
FIRMWARE_SIZE_KB=$((FIRMWARE_SIZE / 1024))
echo -e "${BLUE}Firmware Details:${NC}"
echo "  Size:       ${FIRMWARE_SIZE_KB}KB"
echo "  WiFi SSID:  Jeremy's Wi-Fi Network"
echo "  Static IP:  192.168.1.210"
echo

# Prompt for confirmation
echo -e "${YELLOW}Ready to flash ESP32-CAM.${NC}"
echo -e "${YELLOW}Make sure:${NC}"
echo "  1. ESP32-CAM is inserted in ESP32-CAM-MB programmer"
echo "  2. USB cable is connected"
echo "  3. IO0 is pulled low (programming mode switch ON)"
echo
read -p "Continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Flash cancelled.${NC}"
    exit 0
fi

echo
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Flashing Firmware...${NC}"
echo -e "${BLUE}========================================${NC}"
echo

# Try using PlatformIO first
if command -v pio &> /dev/null; then
    echo -e "${GREEN}Using PlatformIO...${NC}"
    pio run --target upload --upload-port "$PORT"
    FLASH_RESULT=$?
elif command -v esptool.py &> /dev/null; then
    echo -e "${GREEN}Using esptool.py...${NC}"
    esptool.py --chip esp32 --port "$PORT" --baud 921600 \
        --before default_reset --after hard_reset write_flash -z \
        --flash_mode dio --flash_freq 80m --flash_size detect \
        0x1000 "$BOOTLOADER_PATH" \
        0x8000 "$PARTITIONS_PATH" \
        0x10000 "$FIRMWARE_PATH"
    FLASH_RESULT=$?
else
    echo -e "${RED}ERROR: Neither 'pio' nor 'esptool.py' found!${NC}"
    echo "Install one of the following:"
    echo "  - PlatformIO: https://platformio.org/install/cli"
    echo "  - esptool: pip install esptool"
    exit 1
fi

echo
echo -e "${BLUE}========================================${NC}"

if [ $FLASH_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Flash Successful!${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo
    echo -e "${GREEN}Next Steps:${NC}"
    echo "  1. Remove ESP32-CAM from programmer board"
    echo "  2. Power cycle the device (or press RST button)"
    echo "  3. Connect via serial monitor to verify boot:"
    echo "     pio device monitor"
    echo
    echo "  4. Wait for WiFi connection"
    echo "     Expected IP: 192.168.1.210"
    echo
    echo "  5. Test web interface:"
    echo "     curl http://192.168.1.210:8080"
    echo
    echo "  6. Test audio stream:"
    echo "     timeout 10 curl http://192.168.1.210:8080/stream > test.wav"
    echo "     ffplay test.wav"
    echo
else
    echo -e "${RED}✗ Flash Failed!${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo
    echo -e "${YELLOW}Troubleshooting:${NC}"
    echo "  1. Ensure ESP32-CAM is in programming mode (IO0 low)"
    echo "  2. Check USB cable connection"
    echo "  3. Try a different USB port"
    echo "  4. Verify correct port ($PORT)"
    echo "  5. Check CH340 driver installation (for ESP32-CAM-MB)"
    exit 1
fi
