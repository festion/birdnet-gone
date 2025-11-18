@echo off
REM ESP32-CAM Firmware Flash Script for Windows
REM Run this from the firmware directory

echo ========================================
echo ESP32-CAM Firmware Flash Tool
echo ========================================
echo.

REM Check if firmware files exist
if not exist ".pio\build\esp32cam\firmware.bin" (
    echo ERROR: firmware.bin not found!
    echo Run 'pio run' to build first.
    pause
    exit /b 1
)

echo Firmware files found.
echo.

REM Find COM port
echo Looking for COM port...
for /f "tokens=1" %%p in ('mode ^| findstr "COM"') do (
    set PORT=%%p
)

if "%PORT%"=="" (
    set PORT=COM3
    echo No COM port detected, using default: %PORT%
) else (
    echo Found: %PORT%
)

echo.
echo Flash Configuration:
echo   Port: %PORT%
echo   Baud: 460800
echo.
echo IMPORTANT: Make sure ESP32-CAM is in programming mode!
echo   - IO0 pulled to GND (programming switch ON)
echo   - USB cable connected
echo.

set /p CONFIRM=Continue? (Y/N):
if /i not "%CONFIRM%"=="Y" (
    echo Flash cancelled.
    pause
    exit /b 0
)

echo.
echo Step 1: Erasing flash...
python -m esptool --chip esp32 --port %PORT% erase_flash
if errorlevel 1 (
    echo ERROR: Flash erase failed!
    pause
    exit /b 1
)

echo.
echo Step 2: Flashing firmware...
python -m esptool --chip esp32 --port %PORT% --baud 460800 ^
    --before default_reset --after hard_reset write_flash -z ^
    --flash_mode dio --flash_freq 80m --flash_size 4MB ^
    0x1000 .pio\build\esp32cam\bootloader.bin ^
    0x8000 .pio\build\esp32cam\partitions.bin ^
    0x10000 .pio\build\esp32cam\firmware.bin

if errorlevel 1 (
    echo.
    echo ERROR: Flash failed!
    pause
    exit /b 1
)

echo.
echo ========================================
echo SUCCESS! Firmware flashed.
echo ========================================
echo.
echo Next steps:
echo   1. Set programming switch to OFF (IO0 floating)
echo   2. Press RESET button or power cycle
echo   3. Device will connect to WiFi: Lakehouse
echo   4. Access at: http://192.168.1.212:8080
echo.
pause
