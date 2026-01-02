/**
 * BirdNET-Gone XIAO ESP32-C6 Wireless Microphone
 *
 * HTTP audio streaming server for BirdNET-Go bird detection
 * Supports XIAO ESP32-C6 + ICS-43434 I2S Microphone
 *
 * Features:
 * - I2S digital microphone support (ICS-43434)
 * - HTTP WAV audio streaming (48kHz, 16-bit, mono)
 * - Raw PCM streaming endpoint for ffmpeg compatibility
 * - Async web server for multiple concurrent clients
 * - Client timeout and watchdog protection
 * - OTA firmware updates
 * - Remote reboot capability
 * - Status monitoring endpoint
 *
 * Hardware: XIAO ESP32-C6
 * Microphone: ICS-43434 I2S MEMS
 *
 * Wiring (XIAO ESP32-C6 to ICS-43434):
 * ─────────────────────────────────────
 * 3V3  → VDD
 * GND  → GND
 * D4   → WS (Word Select / LRCLK) - GPIO4
 * D5   → SCK (Bit Clock / BCLK)   - GPIO5
 * D6   → SD (Serial Data / DOUT)  - GPIO6
 * GND  → L/R (Left channel select)
 *
 * Stream URLs:
 * - WAV:  http://<IP>:8080/stream
 * - RAW:  http://<IP>:8080/stream_raw (16-bit PCM, no header)
 *
 * Author: BirdNET-Gone Project
 * License: MIT
 * Version: 2.2.0
 */

#include <Arduino.h>
#include <WiFi.h>
#include <ESPAsyncWebServer.h>
#include <ArduinoOTA.h>
#include <driver/i2s.h>
#include <esp_task_wdt.h>

// ============================================================================
// CONFIGURATION - UPDATE THESE VALUES
// ============================================================================

// WiFi credentials
const char* WIFI_SSID = "Lakehouse";
const char* WIFI_PASSWORD = "redflower805";

// Use DHCP (comment out static IP section to use DHCP)
// Static IP configuration (optional)
// IPAddress local_IP(192, 168, 1, 131);
// IPAddress gateway(192, 168, 1, 1);
// IPAddress subnet(255, 255, 255, 0);
// IPAddress primaryDNS(8, 8, 8, 8);
// IPAddress secondaryDNS(8, 8, 4, 4);
// #define USE_STATIC_IP

// ============================================================================
// I2S MICROPHONE CONFIGURATION - XIAO ESP32-C6 + ICS-43434
// ============================================================================

// I2S pins for XIAO ESP32-C6
#define I2S_WS_PIN      4     // Word Select (LRCLK) - D4/GPIO4
#define I2S_SCK_PIN     5     // Bit Clock (BCLK)    - D5/GPIO5
#define I2S_SD_PIN      6     // Serial Data (DOUT)  - D6/GPIO6

// Audio format settings
#define SAMPLE_RATE     48000                     // 48kHz for high quality
#define BITS_PER_SAMPLE I2S_BITS_PER_SAMPLE_16BIT // 16-bit audio
#define CHANNELS        1                         // Mono

// I2S configuration
#define I2S_PORT        I2S_NUM_0
#define DMA_BUFFER_COUNT 8
#define DMA_BUFFER_SIZE  1024

// ============================================================================
// SERVER CONFIGURATION
// ============================================================================

#define HTTP_PORT           8080
#define CLIENT_TIMEOUT_MS   30000   // 30 seconds
#define WATCHDOG_TIMEOUT_S  60      // 60 seconds
#define MAX_CLIENTS         4       // Maximum concurrent streaming clients

// ============================================================================
// GLOBAL VARIABLES
// ============================================================================

AsyncWebServer server(HTTP_PORT);
volatile int activeClients = 0;
unsigned long totalConnections = 0;

// Audio monitoring
unsigned long lastAudioDebug = 0;
const unsigned long AUDIO_DEBUG_INTERVAL = 10000; // 10 seconds

// Audio diagnostics
int16_t lastMinVal = 0;
int16_t lastMaxVal = 0;
int lastNonZeroCount = 0;
int lastSampleCount = 0;

// Client tracking
struct StreamingClient {
    unsigned long lastActivity;
    bool headerSent;
    bool disconnected;
    bool isRaw;  // true for raw PCM, false for WAV
};

#define MAX_STREAMING_CLIENTS 8
static StreamingClient streamingClients[MAX_STREAMING_CLIENTS];
static int nextClientSlot = 0;

// WAV header structure
struct WAVHeader {
    char riff[4] = {'R', 'I', 'F', 'F'};
    uint32_t fileSize = 0xFFFFFFFF;  // Unknown size (streaming)
    char wave[4] = {'W', 'A', 'V', 'E'};
    char fmt[4] = {'f', 'm', 't', ' '};
    uint32_t fmtSize = 16;
    uint16_t audioFormat = 1;         // PCM
    uint16_t numChannels = CHANNELS;
    uint32_t sampleRate = SAMPLE_RATE;
    uint32_t byteRate = SAMPLE_RATE * CHANNELS * 2;
    uint16_t blockAlign = CHANNELS * 2;
    uint16_t bitsPerSample = 16;
    char data[4] = {'d', 'a', 't', 'a'};
    uint32_t dataSize = 0xFFFFFFFF;   // Unknown size (streaming)
};

// ============================================================================
// I2S MICROPHONE FUNCTIONS
// ============================================================================

bool initI2S() {
    Serial.println("\nConfiguring I2S microphone...");
    Serial.printf("  Pins: WS=%d, SCK=%d, SD=%d\n", I2S_WS_PIN, I2S_SCK_PIN, I2S_SD_PIN);
    Serial.printf("  Format: %d Hz, %d-bit, %d channel\n", SAMPLE_RATE, 16, CHANNELS);

    // I2S configuration for ICS-43434
    i2s_config_t i2s_config = {
        .mode = (i2s_mode_t)(I2S_MODE_MASTER | I2S_MODE_RX),
        .sample_rate = SAMPLE_RATE,
        .bits_per_sample = BITS_PER_SAMPLE,
        .channel_format = I2S_CHANNEL_FMT_RIGHT_LEFT,  // Try both channels to detect any signal
        .communication_format = I2S_COMM_FORMAT_STAND_I2S,
        .intr_alloc_flags = ESP_INTR_FLAG_LEVEL1,
        .dma_buf_count = DMA_BUFFER_COUNT,
        .dma_buf_len = DMA_BUFFER_SIZE,
        .use_apll = false,
        .tx_desc_auto_clear = false,
        .fixed_mclk = 0
    };

    i2s_pin_config_t pin_config = {
        .bck_io_num = I2S_SCK_PIN,
        .ws_io_num = I2S_WS_PIN,
        .data_out_num = I2S_PIN_NO_CHANGE,
        .data_in_num = I2S_SD_PIN
    };

    esp_err_t err = i2s_driver_install(I2S_PORT, &i2s_config, 0, NULL);
    if (err != ESP_OK) {
        Serial.printf("[ERROR] I2S driver install failed: %d\n", err);
        return false;
    }

    err = i2s_set_pin(I2S_PORT, &pin_config);
    if (err != ESP_OK) {
        Serial.printf("[ERROR] I2S pin config failed: %d\n", err);
        i2s_driver_uninstall(I2S_PORT);
        return false;
    }

    // Test microphone
    delay(100);
    int16_t testBuffer[512];
    size_t bytesRead = 0;

    Serial.print("[I2S] Testing microphone");
    for (int attempt = 1; attempt <= 3; attempt++) {
        Serial.print(".");
        i2s_read(I2S_PORT, testBuffer, sizeof(testBuffer), &bytesRead, 500);

        int samplesRead = bytesRead / sizeof(int16_t);
        int nonZeroCount = 0;
        int16_t minVal = 32767, maxVal = -32768;

        for (int i = 0; i < samplesRead; i++) {
            if (testBuffer[i] != 0) nonZeroCount++;
            if (testBuffer[i] < minVal) minVal = testBuffer[i];
            if (testBuffer[i] > maxVal) maxVal = testBuffer[i];
        }

        if (nonZeroCount > samplesRead / 10) {
            Serial.println("\n[OK] I2S microphone initialized");
            Serial.printf("   Sample Rate: %d Hz\n", SAMPLE_RATE);
            Serial.printf("   Bit Depth: 16 bits\n");
            Serial.printf("   Channels: 1 (Mono)\n");
            return true;
        }
        delay(100);
    }

    Serial.println("\n[WARN] Microphone may be silent - proceeding anyway");
    return true;
}

size_t readI2SAudio(int16_t* buffer, size_t size) {
    size_t bytesRead = 0;
    esp_err_t result = i2s_read(I2S_PORT, buffer, size, &bytesRead, 100);

    if (result != ESP_OK) {
        return 0;
    }

    // Periodic audio monitoring
    unsigned long now = millis();
    if (now - lastAudioDebug > AUDIO_DEBUG_INTERVAL) {
        lastAudioDebug = now;

        int samplesRead = bytesRead / sizeof(int16_t);
        int nonZeroCount = 0;
        int16_t minVal = 32767, maxVal = -32768;

        for (int i = 0; i < samplesRead; i++) {
            if (buffer[i] != 0) nonZeroCount++;
            if (buffer[i] < minVal) minVal = buffer[i];
            if (buffer[i] > maxVal) maxVal = buffer[i];
        }

        lastMinVal = minVal;
        lastMaxVal = maxVal;
        lastNonZeroCount = nonZeroCount;
        lastSampleCount = samplesRead;

        Serial.printf("[Audio] samples=%d, non-zero=%d, range=[%d, %d]\n",
                      samplesRead, nonZeroCount, minVal, maxVal);
    }

    return bytesRead;
}

// ============================================================================
// HTTP SERVER HANDLERS
// ============================================================================

void handleRoot(AsyncWebServerRequest *request) {
    String html = "<!DOCTYPE html><html><head><title>BirdNET Audio Streamer</title>";
    html += "<style>";
    html += "body{font-family:Arial;margin:40px;background:#f0f0f0}";
    html += ".container{background:white;padding:30px;border-radius:10px;max-width:600px;margin:auto}";
    html += "h1{color:#2c3e50}h3{color:#34495e}";
    html += ".info{background:#e8f4f8;padding:15px;border-radius:5px;margin:20px 0}";
    html += ".status{background:#d4edda;padding:15px;border-radius:5px;margin:20px 0}";
    html += ".url{background:#333;color:#0f0;padding:10px;border-radius:5px;font-family:monospace;word-break:break-all;margin:10px 0}";
    html += "button{background:#3498db;color:white;border:none;padding:12px 24px;border-radius:5px;cursor:pointer;font-size:16px;margin:10px 5px}";
    html += "button:hover{background:#2980b9}";
    html += ".reboot-button{background:#e74c3c}.reboot-button:hover{background:#c0392b}";
    html += "</style></head><body><div class='container'>";

    html += "<h1>BirdNET Audio Streamer</h1>";
    html += "<div class='info'>";
    html += "<p><strong>Device:</strong> XIAO ESP32-C6 + ICS-43434</p>";
    html += "<p><strong>Sample Rate:</strong> " + String(SAMPLE_RATE) + " Hz</p>";
    html += "<p><strong>Format:</strong> 16-bit Mono PCM</p>";
    html += "<p><strong>Active Clients:</strong> <span id='clients'>" + String(activeClients) + "</span></p>";
    html += "<p><strong>Uptime:</strong> " + String(millis() / 1000) + " seconds</p>";
    html += "<p><strong>Free Heap:</strong> " + String(ESP.getFreeHeap()) + " bytes</p>";
    html += "<p><strong>WiFi RSSI:</strong> " + String(WiFi.RSSI()) + " dBm</p>";
    html += "</div>";

    html += "<h3>WAV Stream (for browsers):</h3>";
    html += "<div class='url'>http://" + WiFi.localIP().toString() + ":8080/stream</div>";

    html += "<h3>Raw PCM Stream (for ffmpeg/BirdNET-Go):</h3>";
    html += "<div class='url'>http://" + WiFi.localIP().toString() + ":8080/stream_raw</div>";
    html += "<p><small>Format: 16-bit signed LE, " + String(SAMPLE_RATE) + " Hz, mono</small></p>";

    html += "<h3>ffmpeg command:</h3>";
    html += "<div class='url' style='font-size:12px'>ffmpeg -f s16le -ar " + String(SAMPLE_RATE) + " -ac 1 -i http://" + WiFi.localIP().toString() + ":8080/stream_raw -f alsa hw:0,0</div>";

    html += "<h3>Actions:</h3>";
    html += "<button onclick=\"window.location='/status'\">Status JSON</button>";
    html += "<button onclick=\"window.location='/diag'\">Diagnostics</button>";
    html += "<button class='reboot-button' onclick=\"if(confirm('Reboot?'))window.location='/reboot'\">Reboot</button>";

    html += "</div></body></html>";
    request->send(200, "text/html", html);
}

void handleStatus(AsyncWebServerRequest *request) {
    String json = "{";
    json += "\"activeClients\":" + String(activeClients) + ",";
    json += "\"uptime\":" + String(millis() / 1000) + ",";
    json += "\"freeHeap\":" + String(ESP.getFreeHeap()) + ",";
    json += "\"wifiRSSI\":" + String(WiFi.RSSI()) + ",";
    json += "\"sampleRate\":" + String(SAMPLE_RATE) + ",";
    json += "\"bitsPerSample\":16,";
    json += "\"channels\":1";
    json += "}";
    request->send(200, "application/json", json);
}

void handleDiag(AsyncWebServerRequest *request) {
    // Do a fresh I2S read right now for real-time diagnostics
    int16_t testBuffer[1024];
    size_t bytesRead = 0;

    esp_err_t result = i2s_read(I2S_PORT, testBuffer, sizeof(testBuffer), &bytesRead, 500);

    int samplesRead = bytesRead / sizeof(int16_t);
    int nonZeroCount = 0;
    int leftNonZero = 0;
    int rightNonZero = 0;
    int16_t minVal = 32767, maxVal = -32768;
    int16_t leftMin = 32767, leftMax = -32768;
    int16_t rightMin = 32767, rightMax = -32768;

    for (int i = 0; i < samplesRead; i++) {
        if (testBuffer[i] != 0) nonZeroCount++;
        if (testBuffer[i] < minVal) minVal = testBuffer[i];
        if (testBuffer[i] > maxVal) maxVal = testBuffer[i];

        // Check left channel (even indices) and right channel (odd indices) separately
        if (i % 2 == 0) {
            if (testBuffer[i] != 0) leftNonZero++;
            if (testBuffer[i] < leftMin) leftMin = testBuffer[i];
            if (testBuffer[i] > leftMax) leftMax = testBuffer[i];
        } else {
            if (testBuffer[i] != 0) rightNonZero++;
            if (testBuffer[i] < rightMin) rightMin = testBuffer[i];
            if (testBuffer[i] > rightMax) rightMax = testBuffer[i];
        }
    }

    String json = "{";
    json += "\"i2s_result\":" + String(result) + ",";
    json += "\"bytesRead\":" + String(bytesRead) + ",";
    json += "\"samplesRead\":" + String(samplesRead) + ",";
    json += "\"nonZeroCount\":" + String(nonZeroCount) + ",";
    json += "\"minVal\":" + String(minVal) + ",";
    json += "\"maxVal\":" + String(maxVal) + ",";
    json += "\"leftChannel\":{\"nonZero\":" + String(leftNonZero) + ",\"min\":" + String(leftMin) + ",\"max\":" + String(leftMax) + "},";
    json += "\"rightChannel\":{\"nonZero\":" + String(rightNonZero) + ",\"min\":" + String(rightMin) + ",\"max\":" + String(rightMax) + "},";
    json += "\"i2s_ws_pin\":" + String(I2S_WS_PIN) + ",";
    json += "\"i2s_sck_pin\":" + String(I2S_SCK_PIN) + ",";
    json += "\"i2s_sd_pin\":" + String(I2S_SD_PIN) + ",";
    json += "\"audioWorking\":" + String((nonZeroCount > samplesRead / 10) ? "true" : "false");
    json += "}";
    request->send(200, "application/json", json);
}

void handleReboot(AsyncWebServerRequest *request) {
    request->send(200, "text/plain", "Rebooting...");
    delay(500);
    ESP.restart();
}

void handleReset(AsyncWebServerRequest *request) {
    for (int i = 0; i < MAX_STREAMING_CLIENTS; i++) {
        streamingClients[i].disconnected = true;
    }
    activeClients = 0;
    request->send(200, "text/plain", "Client counter reset");
}

// Generic stream handler for both WAV and raw PCM
void handleStreamGeneric(AsyncWebServerRequest *request, bool isRaw) {
    if (activeClients >= MAX_CLIENTS) {
        request->send(503, "text/plain", "Too many clients");
        return;
    }

    int clientSlot = nextClientSlot;
    nextClientSlot = (nextClientSlot + 1) % MAX_STREAMING_CLIENTS;

    streamingClients[clientSlot].lastActivity = millis();
    streamingClients[clientSlot].headerSent = false;
    streamingClients[clientSlot].disconnected = false;
    streamingClients[clientSlot].isRaw = isRaw;

    activeClients++;
    totalConnections++;

    Serial.printf("[HTTP] Client #%d connected (%s), active: %d\n",
                  totalConnections, isRaw ? "raw" : "wav", activeClients);

    int slot = clientSlot;

    const char* contentType = isRaw ? "application/octet-stream" : "audio/wav";

    AsyncWebServerResponse *response = request->beginChunkedResponse(
        contentType,
        [slot](uint8_t *buffer, size_t maxLen, size_t index) -> size_t {
            StreamingClient* client = &streamingClients[slot];

            if (client->disconnected) {
                return 0;
            }

            // Send WAV header on first chunk (only for WAV streams)
            if (!client->headerSent) {
                client->headerSent = true;
                client->lastActivity = millis();

                if (!client->isRaw) {
                    WAVHeader header;
                    memcpy(buffer, &header, sizeof(WAVHeader));
                    return sizeof(WAVHeader);
                }
            }

            // Check timeout
            unsigned long now = millis();
            if (now - client->lastActivity > CLIENT_TIMEOUT_MS) {
                Serial.printf("[HTTP] Client timeout (slot %d)\n", slot);
                if (!client->disconnected) {
                    client->disconnected = true;
                    activeClients--;
                }
                return 0;
            }

            // Read audio
            size_t audioSize = min(maxLen, (size_t)2048);
            size_t bytesRead = readI2SAudio((int16_t*)buffer, audioSize);

            if (bytesRead > 0) {
                client->lastActivity = now;
            }

            return bytesRead;
        }
    );

    // Set headers for better streaming
    response->addHeader("Cache-Control", "no-cache, no-store, must-revalidate");
    response->addHeader("Connection", "close");

    request->send(response);
}

void handleStream(AsyncWebServerRequest *request) {
    handleStreamGeneric(request, false);  // WAV stream
}

void handleStreamRaw(AsyncWebServerRequest *request) {
    handleStreamGeneric(request, true);   // Raw PCM stream
}

// ============================================================================
// WiFi FUNCTIONS
// ============================================================================

void setupWiFi() {
    Serial.print("\nConnecting to WiFi");

#ifdef USE_STATIC_IP
    if (!WiFi.config(local_IP, gateway, subnet, primaryDNS, secondaryDNS)) {
        Serial.println("\n[WARN] Static IP failed, using DHCP");
    }
#endif

    WiFi.mode(WIFI_STA);
    WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

    int attempts = 0;
    while (WiFi.status() != WL_CONNECTED && attempts < 40) {
        delay(500);
        Serial.print(".");
        attempts++;
    }

    if (WiFi.status() == WL_CONNECTED) {
        Serial.println("\n[OK] WiFi connected!");
        Serial.printf("IP address: %s\n", WiFi.localIP().toString().c_str());
        Serial.printf("Signal: %d dBm\n", WiFi.RSSI());
    } else {
        Serial.println("\n[ERROR] WiFi failed! Rebooting in 10s...");
        delay(10000);
        ESP.restart();
    }
}

void setupOTA() {
    Serial.println("\nConfiguring OTA updates...");

    ArduinoOTA.setHostname("birdnet-mic");
    ArduinoOTA.setPort(3232);

    ArduinoOTA.onStart([]() {
        String type = (ArduinoOTA.getCommand() == U_FLASH) ? "sketch" : "filesystem";
        Serial.println("[OTA] Start updating " + type);
        // Disable watchdog during OTA to prevent resets
        esp_task_wdt_delete(NULL);
        Serial.println("[OTA] Watchdog disabled for update");
    });

    ArduinoOTA.onEnd([]() {
        Serial.println("\n[OTA] Update complete!");
    });

    ArduinoOTA.onProgress([](unsigned int progress, unsigned int total) {
        Serial.printf("[OTA] Progress: %u%%\r", (progress / (total / 100)));
    });

    ArduinoOTA.onError([](ota_error_t error) {
        Serial.printf("[OTA] Error[%u]: ", error);
        if (error == OTA_AUTH_ERROR) Serial.println("Auth Failed");
        else if (error == OTA_BEGIN_ERROR) Serial.println("Begin Failed");
        else if (error == OTA_CONNECT_ERROR) Serial.println("Connect Failed");
        else if (error == OTA_RECEIVE_ERROR) Serial.println("Receive Failed");
        else if (error == OTA_END_ERROR) Serial.println("End Failed");
    });

    ArduinoOTA.begin();
    Serial.println("[OK] OTA configured on port 3232");
}

void setupServer() {
    // Configure OTA first
    setupOTA();

    Serial.println("\nStarting HTTP server...");

    // Give the network stack time to fully initialize
    delay(1000);

    server.on("/", HTTP_GET, handleRoot);
    server.on("/status", HTTP_GET, handleStatus);
    server.on("/diag", HTTP_GET, handleDiag);
    server.on("/stream", HTTP_GET, handleStream);
    server.on("/stream_raw", HTTP_GET, handleStreamRaw);
    server.on("/reboot", HTTP_GET, handleReboot);
    server.on("/reset", HTTP_GET, handleReset);

    server.onNotFound([](AsyncWebServerRequest *request) {
        request->send(404, "text/plain", "Not found");
    });

    // Start server directly - pioarduino platform handles TCPIP locking properly
    Serial.println("Initializing TCP server...");
    server.begin();
    Serial.println("[OK] HTTP server started on port 8080");
}

// ============================================================================
// MAIN SETUP AND LOOP
// ============================================================================

void setup() {
    Serial.begin(115200);
    delay(2000);  // Wait for serial monitor

    Serial.println("\n\n========================================");
    Serial.println("  BirdNET Async Audio Streamer");
    Serial.println("  XIAO ESP32-C6 + ICS-43434");
    Serial.println("  Version 2.2.0 - OTA Support");
    Serial.println("========================================");

    // WiFi
    setupWiFi();

    // I2S
    if (!initI2S()) {
        Serial.println("[FATAL] I2S init failed!");
        while(1) { delay(1000); }
    }

    // OTA and HTTP - do this BEFORE watchdog
    setupServer();

    // Watchdog - configure AFTER server starts to avoid timeout during init
    Serial.println("\nConfiguring watchdog timer...");
    esp_task_wdt_config_t wdt_config = {
        .timeout_ms = WATCHDOG_TIMEOUT_S * 1000,
        .idle_core_mask = (1 << 0),  // Watch core 0
        .trigger_panic = true
    };
    esp_task_wdt_init(&wdt_config);
    esp_task_wdt_add(NULL);
    Serial.printf("[OK] Watchdog: %d seconds\n", WATCHDOG_TIMEOUT_S);

    Serial.println("\nAudio streaming configured via HTTP chunked responses");
    Serial.println("\n========================================");
    Serial.println("System Ready!");
    Serial.println("========================================");
    Serial.printf("Web Interface: http://%s:%d\n", WiFi.localIP().toString().c_str(), HTTP_PORT);
    Serial.printf("WAV Stream:    http://%s:%d/stream\n", WiFi.localIP().toString().c_str(), HTTP_PORT);
    Serial.printf("Raw Stream:    http://%s:%d/stream_raw\n", WiFi.localIP().toString().c_str(), HTTP_PORT);
    Serial.println("========================================\n");
}

void loop() {
    esp_task_wdt_reset();

    // Handle OTA updates
    ArduinoOTA.handle();

    // Monitor WiFi
    if (WiFi.status() != WL_CONNECTED) {
        Serial.println("[WARN] WiFi lost, reconnecting...");
        setupWiFi();
    }

    // Periodic cleanup
    static unsigned long lastCleanup = 0;
    unsigned long now = millis();
    if (now - lastCleanup > 30000) {
        lastCleanup = now;

        int actualActive = 0;
        for (int i = 0; i < MAX_STREAMING_CLIENTS; i++) {
            if (!streamingClients[i].disconnected && streamingClients[i].headerSent) {
                if (now - streamingClients[i].lastActivity > CLIENT_TIMEOUT_MS * 2) {
                    streamingClients[i].disconnected = true;
                } else {
                    actualActive++;
                }
            }
        }

        if (activeClients != actualActive) {
            Serial.printf("[CLEANUP] Corrected clients: %d -> %d\n", activeClients, actualActive);
            activeClients = actualActive;
        }
    }

    delay(100);
}
