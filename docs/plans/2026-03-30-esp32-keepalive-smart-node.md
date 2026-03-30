# ESP32-C6 Smart Keepalive Node Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Upgrade the ESP32-C6 keepalive buzzer from a dumb PWM tone generator to a networked IoT node with WiFi, MQTT heartbeat, HTTP control API, and OTA firmware updates.

**Architecture:** Arduino framework on XIAO ESP32-C6. WiFi connects to Lakehouse IoT (2.4GHz). MQTT publishes heartbeat every 60s to `birdnet/keepalive/status` with LWT for offline detection. HTTP server exposes status/control/OTA endpoints. Tone generation runs independently of network — if WiFi drops, buzzer keeps buzzing. BirdNET health check v7 gains an additional MQTT check for the ESP32 heartbeat.

**Tech Stack:** PlatformIO, Arduino framework, ESP32-C6 (pioarduino platform), PubSubClient (MQTT), ArduinoOTA, WebServer (built-in ESP32), ArduinoJson

---

## Context for Implementer

### Hardware

- **Board:** Seeed XIAO ESP32-C6 (ESP32-C6 SoC, WiFi 6, BLE 5.0)
- **Piezo:** Passive piezo element on GPIO0 (PWM via LEDC)
- **LED:** Onboard LED on GPIO8
- **Power:** USB from Pi or charger
- **Placement:** Within 30cm of BOYA transmitter

### Current firmware

`firmware/esp32-keepalive/src/main.cpp` — 91 lines. Just LEDC PWM at 15.5kHz on GPIO0 and LED blink. No WiFi, no network.

### Network

- **SSID:** `Lakehouse` (IoT network, 2.4GHz)
- **MQTT broker:** 192.168.1.148:1883 (user: `birdnet`, pass: `secret`)
- **MQTT topic:** `birdnet/keepalive/status` (retained)
- **MQTT LWT:** Same topic, payload `{"alive":false}` (retained)

### Credentials strategy

WiFi password goes in `config.h` (gitignored). `config.h.example` checked in with placeholders. The WiFi password is the Lakehouse IoT PSK — retrieve from Infisical or get from user.

---

## Task 1: Firmware scaffold (platformio.ini, config, gitignore)

> **Parallelizable:** Yes (independent of Task 3)

**Files:**
- Modify: `firmware/esp32-keepalive/platformio.ini`
- Create: `firmware/esp32-keepalive/src/config.h.example`
- Modify: `.gitignore`

**Step 1: Update platformio.ini with library dependencies**

Replace contents of `firmware/esp32-keepalive/platformio.ini`:

```ini
; PlatformIO Project Configuration File
;
; BirdNET-Gone Smart Keepalive Node v2
; ESP32-C6 with WiFi, MQTT heartbeat, HTTP control, OTA updates.
; Emits ultrasonic tone via piezo to prevent BOYA wireless mic auto-sleep.

[env:esp32c6]
platform = https://github.com/pioarduino/platform-espressif32/releases/download/51.03.07/platform-espressif32.zip
board = esp32-c6-devkitm-1
framework = arduino
monitor_speed = 115200

board_build.mcu = esp32c6
board_build.f_cpu = 160000000L

build_flags =
    -DARDUINO_USB_MODE=1
    -DARDUINO_USB_CDC_ON_BOOT=1

lib_deps =
    knolleary/PubSubClient@^2.8
    bblanchon/ArduinoJson@^7.0.0

upload_speed = 921600

; OTA upload (use after initial USB flash):
; pio run -t upload --upload-port <ESP32_IP>
upload_protocol = esptool
```

**Step 2: Create config.h.example**

Create `firmware/esp32-keepalive/src/config.h.example`:

```cpp
#ifndef CONFIG_H
#define CONFIG_H

// WiFi
#define WIFI_SSID     "Lakehouse"
#define WIFI_PASSWORD "your_wifi_password_here"

// MQTT Broker
#define MQTT_BROKER   "192.168.1.148"
#define MQTT_PORT     1883
#define MQTT_USER     "birdnet"
#define MQTT_PASSWORD "secret"
#define MQTT_TOPIC    "birdnet/keepalive/status"
#define MQTT_CLIENT   "birdnet-keepalive"

#endif
```

**Step 3: Add config.h to .gitignore**

Append to `.gitignore`:

```
# ESP32 keepalive credentials
firmware/esp32-keepalive/src/config.h
```

**Step 4: Verify no syntax issues**

Run: `head -5 firmware/esp32-keepalive/src/config.h.example`
Expected: `#ifndef CONFIG_H`

**Step 5: Commit**

```bash
git add firmware/esp32-keepalive/platformio.ini firmware/esp32-keepalive/src/config.h.example .gitignore
git commit -m "feat: scaffold ESP32 keepalive v2 — add deps, config template, gitignore"
```

---

## Task 2: Firmware implementation (main.cpp rewrite)

> **Parallelizable:** No (depends on Task 1 for config.h structure and lib_deps)

**Files:**
- Modify: `firmware/esp32-keepalive/src/main.cpp`

**Step 1: Write the v2 firmware**

Replace contents of `firmware/esp32-keepalive/src/main.cpp`:

```cpp
/**
 * BirdNET-Gone Smart Keepalive Node v2
 *
 * Emits a 15.5kHz tone through a piezo element to prevent the BOYA Magic
 * wireless microphone transmitter from entering auto-sleep during overnight
 * silence. The tone is just above BirdNET's 15kHz analysis ceiling.
 *
 * v2 adds: WiFi, MQTT heartbeat, HTTP control API, OTA firmware updates.
 *
 * Hardware:
 *   XIAO ESP32-C6 + passive piezo element (disc or buzzer)
 *
 * Wiring:
 *   GPIO0 → Piezo (+)
 *   GND   → Piezo (-)
 *
 * Network:
 *   WiFi: Lakehouse IoT (2.4GHz)
 *   MQTT: birdnet/keepalive/status (retained, 60s heartbeat)
 *   HTTP: port 80 (status, tone control, OTA trigger)
 *   OTA:  ArduinoOTA on default port 3232
 */

#include <Arduino.h>
#include <WiFi.h>
#include <ArduinoOTA.h>
#include <WebServer.h>
#include <PubSubClient.h>
#include <ArduinoJson.h>
#include "config.h"

// ============================================================================
// CONFIGURATION
// ============================================================================

// Piezo
constexpr uint8_t  PIEZO_PIN       = 0;
constexpr uint8_t  LEDC_CHANNEL    = 0;
constexpr uint8_t  LEDC_RESOLUTION = 8;
constexpr uint32_t DEFAULT_DUTY    = 128;

// Tone defaults
constexpr uint32_t DEFAULT_FREQ_HZ = 15500;
constexpr uint32_t MIN_FREQ_HZ     = 1000;
constexpr uint32_t MAX_FREQ_HZ     = 20000;

// LED
constexpr uint8_t STATUS_LED = 8;

// Timing
constexpr unsigned long MQTT_HEARTBEAT_MS   = 60000;
constexpr unsigned long MQTT_RECONNECT_MS   = 5000;
constexpr unsigned long WIFI_RECONNECT_MS   = 10000;
constexpr unsigned long LED_BLINK_MS        = 5000;

// Firmware version
#define FIRMWARE_VERSION "2.0.0"

// ============================================================================
// GLOBALS
// ============================================================================

WiFiClient       wifiClient;
PubSubClient     mqtt(wifiClient);
WebServer        server(80);

uint32_t      currentFreq   = DEFAULT_FREQ_HZ;
bool          toneEnabled   = true;
unsigned long lastHeartbeat = 0;
unsigned long lastMqttRetry = 0;
unsigned long lastWifiRetry = 0;
unsigned long lastLedToggle = 0;
unsigned long bootTime      = 0;

// ============================================================================
// TONE CONTROL
// ============================================================================

void toneStart() {
    ledcAttachChannel(PIEZO_PIN, currentFreq, LEDC_RESOLUTION, LEDC_CHANNEL);
    ledcWrite(PIEZO_PIN, DEFAULT_DUTY);
    toneEnabled = true;
}

void toneStop() {
    ledcWrite(PIEZO_PIN, 0);
    toneEnabled = false;
}

void toneSetFreq(uint32_t freq) {
    if (freq < MIN_FREQ_HZ) freq = MIN_FREQ_HZ;
    if (freq > MAX_FREQ_HZ) freq = MAX_FREQ_HZ;
    currentFreq = freq;
    if (toneEnabled) {
        ledcAttachChannel(PIEZO_PIN, currentFreq, LEDC_RESOLUTION, LEDC_CHANNEL);
        ledcWrite(PIEZO_PIN, DEFAULT_DUTY);
    }
}

// ============================================================================
// JSON STATUS
// ============================================================================

String buildStatusJson() {
    JsonDocument doc;
    doc["alive"]   = true;
    doc["freq"]    = currentFreq;
    doc["tone_on"] = toneEnabled;
    doc["uptime_s"] = (millis() - bootTime) / 1000;
    doc["rssi"]    = WiFi.RSSI();
    doc["ip"]      = WiFi.localIP().toString();
    doc["version"] = FIRMWARE_VERSION;

    String output;
    serializeJson(doc, output);
    return output;
}

// ============================================================================
// MQTT
// ============================================================================

void mqttConnect() {
    if (mqtt.connected()) return;
    if (millis() - lastMqttRetry < MQTT_RECONNECT_MS) return;
    lastMqttRetry = millis();

    // LWT: if we disconnect, broker publishes alive:false
    String lwt = "{\"alive\":false}";

    if (mqtt.connect(MQTT_CLIENT, MQTT_USER, MQTT_PASSWORD,
                     MQTT_TOPIC, 1, true, lwt.c_str())) {
        Serial.println("MQTT connected");
        // Publish immediate heartbeat on connect
        String status = buildStatusJson();
        mqtt.publish(MQTT_TOPIC, status.c_str(), true);
    } else {
        Serial.printf("MQTT failed, rc=%d\n", mqtt.state());
    }
}

void mqttHeartbeat() {
    if (!mqtt.connected()) return;
    if (millis() - lastHeartbeat < MQTT_HEARTBEAT_MS) return;
    lastHeartbeat = millis();

    String status = buildStatusJson();
    mqtt.publish(MQTT_TOPIC, status.c_str(), true);
}

// ============================================================================
// HTTP HANDLERS
// ============================================================================

void handleStatus() {
    server.send(200, "application/json", buildStatusJson());
}

void handleTone() {
    bool changed = false;

    if (server.hasArg("enabled")) {
        String val = server.arg("enabled");
        if (val == "true") {
            toneStart();
            changed = true;
        } else if (val == "false") {
            toneStop();
            changed = true;
        }
    }

    if (server.hasArg("freq")) {
        uint32_t freq = server.arg("freq").toInt();
        if (freq > 0) {
            toneSetFreq(freq);
            changed = true;
        }
    }

    if (changed) {
        server.send(200, "application/json", buildStatusJson());
    } else {
        server.send(400, "application/json",
                     "{\"error\":\"use ?enabled=true|false or ?freq=N\"}");
    }
}

void handleNotFound() {
    server.send(404, "application/json", "{\"error\":\"not found\"}");
}

// ============================================================================
// WIFI
// ============================================================================

void wifiConnect() {
    if (WiFi.status() == WL_CONNECTED) return;
    if (millis() - lastWifiRetry < WIFI_RECONNECT_MS) return;
    lastWifiRetry = millis();

    Serial.printf("WiFi connecting to %s...\n", WIFI_SSID);
    WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
}

// ============================================================================
// OTA
// ============================================================================

void otaSetup() {
    ArduinoOTA.setHostname("birdnet-keepalive");

    ArduinoOTA.onStart([]() {
        toneStop();  // Stop tone during OTA to avoid interference
        Serial.println("OTA start");
    });
    ArduinoOTA.onEnd([]() {
        Serial.println("OTA complete");
    });
    ArduinoOTA.onProgress([](unsigned int progress, unsigned int total) {
        Serial.printf("OTA: %u%%\r", (progress / (total / 100)));
    });
    ArduinoOTA.onError([](ota_error_t error) {
        Serial.printf("OTA error[%u]\n", error);
    });

    ArduinoOTA.begin();
}

// ============================================================================
// SETUP
// ============================================================================

void setup() {
    Serial.begin(115200);
    delay(1000);

    bootTime = millis();

    Serial.println();
    Serial.println("=== BirdNET-Gone Smart Keepalive v2 ===");
    Serial.printf("Firmware: %s\n", FIRMWARE_VERSION);
    Serial.printf("Piezo: GPIO%d @ %lu Hz\n", PIEZO_PIN, DEFAULT_FREQ_HZ);

    // LED
    pinMode(STATUS_LED, OUTPUT);

    // Start tone immediately (network-independent)
    toneStart();
    Serial.println("Tone active.");

    // WiFi
    WiFi.mode(WIFI_STA);
    WiFi.setAutoReconnect(true);
    WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

    // Wait up to 10s for initial WiFi connection
    Serial.print("WiFi connecting");
    int attempts = 0;
    while (WiFi.status() != WL_CONNECTED && attempts < 20) {
        // Fast LED blink during WiFi connect
        digitalWrite(STATUS_LED, attempts % 2);
        delay(500);
        Serial.print(".");
        attempts++;
    }
    Serial.println();

    if (WiFi.status() == WL_CONNECTED) {
        Serial.printf("WiFi connected: %s\n", WiFi.localIP().toString().c_str());

        // MQTT
        mqtt.setServer(MQTT_BROKER, MQTT_PORT);
        mqtt.setBufferSize(512);
        mqttConnect();

        // HTTP
        server.on("/status", HTTP_GET, handleStatus);
        server.on("/tone", HTTP_POST, handleTone);
        server.onNotFound(handleNotFound);
        server.begin();
        Serial.println("HTTP server started on port 80");

        // OTA
        otaSetup();
        Serial.println("OTA ready");
    } else {
        Serial.println("WiFi failed — running in offline mode (tone only)");
    }
}

// ============================================================================
// LOOP
// ============================================================================

void loop() {
    unsigned long now = millis();

    // WiFi reconnect (non-blocking)
    if (WiFi.status() != WL_CONNECTED) {
        wifiConnect();
    } else {
        // Only run network services when WiFi is up
        ArduinoOTA.handle();
        server.handleClient();

        // MQTT
        if (!mqtt.connected()) {
            mqttConnect();
        }
        mqtt.loop();
        mqttHeartbeat();
    }

    // LED heartbeat — brief flash every 5s
    if (now - lastLedToggle >= LED_BLINK_MS) {
        lastLedToggle = now;
        digitalWrite(STATUS_LED, HIGH);
    }
    if (now - lastLedToggle >= 100) {
        digitalWrite(STATUS_LED, LOW);
    }
}
```

**Step 2: Verify syntax (basic check)**

Run: `head -5 firmware/esp32-keepalive/src/main.cpp`
Expected: `/**` (comment start)

**Step 3: Commit**

```bash
git add firmware/esp32-keepalive/src/main.cpp
git commit -m "feat: ESP32 keepalive v2 — WiFi, MQTT heartbeat, HTTP control, OTA"
```

---

## Task 3: Health check v7 ESP32 heartbeat integration

> **Parallelizable:** Yes (independent of Tasks 1-2)

**Files:**
- Modify: `scripts/birdnet-health-check.sh`

**Step 1: Add the ESP32 heartbeat check function**

Add the following function after the `check_all_bands_silent()` function (after its closing `}`), before the `# DESYNC RECOVERY` section header:

```bash
# Check ESP32 keepalive heartbeat via MQTT.
# Reads birdnet/keepalive/status retained message.
# Returns 0 if alive and fresh, 1 if stale or offline.
# Sets ESP32_STATUS for inclusion in alerts.
ESP32_STATUS=""
check_esp32_keepalive() {
    local msg
    msg=$(mosquitto_sub \
        -h "$MQTT_BROKER" \
        -u "$MQTT_USER" \
        -P "$MQTT_PASS" \
        -t "birdnet/keepalive/status" \
        -C 1 \
        -W 10 2>/dev/null) || {
        ESP32_STATUS="unreachable (MQTT timeout)"
        log "WARNING: ESP32 keepalive not responding (MQTT timeout)"
        return 1
    }

    if [[ -z "$msg" ]]; then
        ESP32_STATUS="unreachable (empty message)"
        log "WARNING: ESP32 keepalive not responding (empty message)"
        return 1
    fi

    local alive
    alive=$(echo "$msg" | python3 -c "
import json, sys
try:
    d = json.load(sys.stdin)
    print('true' if d.get('alive') else 'false')
except Exception:
    print('error')
" 2>/dev/null)

    if [[ "$alive" != "true" ]]; then
        ESP32_STATUS="offline (alive=$alive)"
        log "WARNING: ESP32 keepalive reports offline (alive=$alive)"
        return 1
    fi

    log "INFO: ESP32 keepalive is alive"
    ESP32_STATUS=""
    return 0
}
```

**Step 2: Add ESP32 check call to main()**

In the `main()` function, add the ESP32 check AFTER the detection stats line (`report_detection_stats`) and BEFORE the normal restart block. Insert:

```bash
    # Check: ESP32 keepalive (informational — does not trigger restart)
    check_esp32_keepalive || true
```

**Step 3: Modify the Pushover alert in the desync path to include ESP32 status**

Find the `send_pushover_alert` call in the desync recovery block and change the message to include ESP32 status when relevant. Replace:

```bash
                send_pushover_alert "Soft recovery failed (USB rebind). Manual re-pair needed. Cooldown until ${cooldown_until}."
```

With:

```bash
                local alert_msg="Soft recovery failed (USB rebind). Manual re-pair needed. Cooldown until ${cooldown_until}."
                if [[ -n "$ESP32_STATUS" ]]; then
                    alert_msg="${alert_msg} Keepalive buzzer: ${ESP32_STATUS}."
                fi
                send_pushover_alert "$alert_msg"
```

**Step 4: Verify syntax**

Run: `bash -n scripts/birdnet-health-check.sh`
Expected: No output (no errors)

**Step 5: Commit**

```bash
git add scripts/birdnet-health-check.sh
git commit -m "feat: health check v7 — add ESP32 keepalive heartbeat monitoring"
```

---

## Task 4: Compile verification

> **Parallelizable:** No (depends on Tasks 1-2)

**Files:**
- Create: `firmware/esp32-keepalive/src/config.h` (gitignored, real credentials)

**Step 1: Create real config.h**

Create `firmware/esp32-keepalive/src/config.h` with actual WiFi credentials. The WiFi password for the Lakehouse IoT SSID must be obtained from the user or Infisical. The MQTT credentials are known:

```cpp
#ifndef CONFIG_H
#define CONFIG_H

// WiFi
#define WIFI_SSID     "Lakehouse"
#define WIFI_PASSWORD "REPLACE_WITH_REAL_PASSWORD"

// MQTT Broker
#define MQTT_BROKER   "192.168.1.148"
#define MQTT_PORT     1883
#define MQTT_USER     "birdnet"
#define MQTT_PASSWORD "secret"
#define MQTT_TOPIC    "birdnet/keepalive/status"
#define MQTT_CLIENT   "birdnet-keepalive"

#endif
```

**IMPORTANT:** Ask the user for the WiFi password before creating this file. Do NOT commit this file (it's gitignored).

**Step 2: Compile firmware**

Run: `cd firmware/esp32-keepalive && pio run 2>&1`
Expected: `SUCCESS` — binary builds without errors

If compilation fails, read the error output carefully:
- Missing library → check `lib_deps` in platformio.ini
- API mismatch → check pioarduino platform version supports the function
- `config.h` not found → verify file is in `src/` directory

**Step 3: Verify binary size**

Run: `ls -la firmware/esp32-keepalive/.pio/build/esp32c6/firmware.bin`
Expected: Binary exists, size ~200KB-1MB (typical for ESP32 Arduino with WiFi)

**Step 4: Report binary ready for flashing**

No commit needed for this task (config.h is gitignored, binary is build artifact).

---

## Task 5: Deploy to Pi and flash ESP32

> **Parallelizable:** No (depends on Tasks 3-4)
> **NOTE:** This task requires physical USB access to the ESP32 for initial flash. After that, OTA handles future updates.

**Step 1: Deploy updated health check to Pi**

```bash
scp scripts/birdnet-health-check.sh jeremy@192.168.1.197:/tmp/birdnet-health-check.sh
ssh jeremy@192.168.1.197 'sudo cp /tmp/birdnet-health-check.sh /usr/local/bin/birdnet-health-check.sh && sudo chmod 755 /usr/local/bin/birdnet-health-check.sh'
```

**Step 2: Flash ESP32 via USB**

The ESP32 must be connected to a USB port. If connected to the dev machine:

```bash
cd firmware/esp32-keepalive && pio run -t upload
```

If connected to the Pi, the binary needs to be copied and flashed there:

```bash
# From dev machine
scp .pio/build/esp32c6/firmware.bin jeremy@192.168.1.197:/tmp/firmware.bin
# On Pi (requires esptool.py)
ssh jeremy@192.168.1.197 'esptool.py --chip esp32c6 --port /dev/ttyACM1 write_flash 0x0 /tmp/firmware.bin'
```

**Step 3: Verify ESP32 connects to WiFi and MQTT**

Wait 15 seconds after flash, then:

```bash
# Check MQTT heartbeat
mosquitto_sub -h 192.168.1.148 -u birdnet -P secret -t 'birdnet/keepalive/status' -C 1 -W 15
```

Expected: JSON with `{"alive":true,"freq":15500,"tone_on":true,...}`

**Step 4: Verify HTTP API**

```bash
curl http://<ESP32_IP>/status
```

Expected: Same JSON as MQTT heartbeat

**Step 5: Test tone control**

```bash
# Disable tone
curl -X POST 'http://<ESP32_IP>/tone?enabled=false'
# Re-enable
curl -X POST 'http://<ESP32_IP>/tone?enabled=true'
# Change frequency
curl -X POST 'http://<ESP32_IP>/tone?freq=14000'
# Reset to default
curl -X POST 'http://<ESP32_IP>/tone?freq=15500'
```

**Step 6: Verify health check sees ESP32**

```bash
ssh jeremy@192.168.1.197 'sudo /usr/local/bin/birdnet-health-check.sh'
ssh jeremy@192.168.1.197 'tail -5 /var/log/birdnet-health-check.log | grep keepalive'
```

Expected: `INFO: ESP32 keepalive is alive`

**Step 7: Test OTA (verify future remote updates work)**

```bash
cd firmware/esp32-keepalive && pio run -t upload --upload-port <ESP32_IP>
```

Expected: Upload succeeds, ESP32 reboots, MQTT heartbeat resumes within 15s.
