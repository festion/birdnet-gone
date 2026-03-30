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
