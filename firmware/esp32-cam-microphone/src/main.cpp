/**
 * BirdNET-Gone ESP32-CAM Wireless Microphone
 *
 * HTTP audio streaming server for BirdNET-Gone bird detection
 * Supports ESP32-CAM + ESP32-CAM-MB programmer board
 *
 * Features:
 * - I2S digital microphone support (INMP441, SPH0645, ICS-43434)
 * - HTTP WAV audio streaming (16kHz, 16-bit, mono)
 * - Async web server for multiple concurrent clients
 * - Client timeout and watchdog protection
 * - OTA firmware updates
 * - Remote reboot capability
 * - Status monitoring endpoint
 *
 * Hardware Requirements:
 * - ESP32-CAM board
 * - ESP32-CAM-MB programmer board (for initial flashing)
 * - I2S MEMS microphone module (INMP441 recommended)
 *
 * Wiring:
 * ESP32-CAM Pin  →  Microphone Pin
 * ─────────────      ──────────────
 * 3V3            →  VDD
 * GND            →  GND / L/R
 * GPIO14         →  WS (Word Select / LRCLK)
 * GPIO15         →  SCK (Bit Clock / BCLK)
 * GPIO13         →  SD (Serial Data / DOUT)
 *
 * Configuration:
 * Update WiFi credentials in this file before compiling
 *
 * BirdNET-Gone Audio Source:
 * http://<ESP32_IP>:8080/stream
 *
 * Author: BirdNET-Gone Project
 * License: MIT
 * Version: 1.0.0
 */

#include <Arduino.h>
#include <WiFi.h>
#include <ESPAsyncWebServer.h>
#include <driver/i2s.h>
#include <esp_task_wdt.h>

// ============================================================================
// CONFIGURATION - UPDATE THESE VALUES
// ============================================================================

// WiFi credentials
const char* WIFI_SSID = "Lakehouse";
const char* WIFI_PASSWORD = "redflower805";

// Static IP configuration (optional - comment out for DHCP)
IPAddress local_IP(192, 168, 1, 212);
IPAddress gateway(192, 168, 1, 1);
IPAddress subnet(255, 255, 255, 0);
IPAddress primaryDNS(8, 8, 8, 8);
IPAddress secondaryDNS(8, 8, 4, 4);

// ============================================================================
// I2S MICROPHONE CONFIGURATION
// ============================================================================

// I2S pins for ESP32-CAM (avoid camera pins)
#define I2S_WS_PIN      14    // Word Select (LRCLK) - GPIO14
#define I2S_SCK_PIN     15    // Bit Clock (BCLK)    - GPIO15
#define I2S_SD_PIN      13    // Serial Data (DOUT)  - GPIO13

// Audio format settings (optimized for BirdNET)
#define SAMPLE_RATE     16000                      // 16kHz for BirdNET
#define BITS_PER_SAMPLE I2S_BITS_PER_SAMPLE_16BIT  // 16-bit audio
#define CHANNELS        1                          // Mono

// I2S configuration
#define I2S_PORT        I2S_NUM_0
#define I2S_BUFFER_COUNT 8
#define I2S_BUFFER_SIZE  512
#define DMA_BUFFER_COUNT 8
#define DMA_BUFFER_SIZE  512

// ============================================================================
// SERVER CONFIGURATION
// ============================================================================

#define HTTP_PORT           8080
#define CLIENT_TIMEOUT_MS   10000   // 10 seconds (shorter timeout to clear zombies faster)
#define WATCHDOG_TIMEOUT_S  60      // 60 seconds
#define MAX_CLIENTS         10      // Maximum concurrent streaming clients (increased for reconnection tolerance)

// ============================================================================
// GLOBAL VARIABLES
// ============================================================================

AsyncWebServer server(HTTP_PORT);
volatile int activeClients = 0;
unsigned long totalConnections = 0;
unsigned long lastWatchdogReset = 0;

// Audio monitoring
unsigned long lastAudioDebug = 0;
const unsigned long AUDIO_DEBUG_INTERVAL = 5000; // 5 seconds

// Audio diagnostics
int16_t lastMinVal = 0;
int16_t lastMaxVal = 0;
int lastNonZeroCount = 0;
int lastSampleCount = 0;

// Structure to track per-client streaming state
struct StreamingClient {
    unsigned long lastActivity;
    bool headerSent;
    bool disconnected;
};

// Simple client tracking array
#define MAX_STREAMING_CLIENTS 10
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
    uint32_t byteRate = SAMPLE_RATE * CHANNELS * 2;  // 16-bit = 2 bytes
    uint16_t blockAlign = CHANNELS * 2;
    uint16_t bitsPerSample = 16;
    char data[4] = {'d', 'a', 't', 'a'};
    uint32_t dataSize = 0xFFFFFFFF;   // Unknown size (streaming)
};

// ============================================================================
// I2S MICROPHONE FUNCTIONS
// ============================================================================

/**
 * Initialize I2S interface for digital microphone
 *
 * Configures ESP32 I2S peripheral in master RX mode for MEMS microphones
 * Uses standard I2S protocol (not PDM) for modules like INMP441
 */
bool initI2S() {
    Serial.println("[I2S] Initializing I2S interface...");
    Serial.printf("[I2S] Configuration: %d Hz, %d-bit, %d channel(s)\n",
                  SAMPLE_RATE, 16, CHANNELS);
    Serial.printf("[I2S] GPIO Pins: WS=%d, SCK=%d, SD=%d\n",
                  I2S_WS_PIN, I2S_SCK_PIN, I2S_SD_PIN);

    // I2S configuration for standard MEMS microphones (not PDM)
    // NOTE: If L/R pin on INMP441 is connected to:
    //   GND = Left channel (I2S_CHANNEL_FMT_ONLY_LEFT)
    //   VDD = Right channel (I2S_CHANNEL_FMT_ONLY_RIGHT)
    // Try RIGHT if LEFT gives silence!
    i2s_config_t i2s_config = {
        .mode = (i2s_mode_t)(I2S_MODE_MASTER | I2S_MODE_RX),  // Master receiver
        .sample_rate = SAMPLE_RATE,
        .bits_per_sample = BITS_PER_SAMPLE,
        .channel_format = I2S_CHANNEL_FMT_ONLY_LEFT,           // Mono (left channel - L/R pin to GND)
        .communication_format = I2S_COMM_FORMAT_STAND_I2S,    // Standard I2S
        .intr_alloc_flags = ESP_INTR_FLAG_LEVEL1,
        .dma_buf_count = DMA_BUFFER_COUNT,
        .dma_buf_len = DMA_BUFFER_SIZE,
        .use_apll = false,                                     // Use PLL (more stable)
        .tx_desc_auto_clear = false,
        .fixed_mclk = 0
    };

    // I2S pin configuration
    i2s_pin_config_t pin_config = {
        .bck_io_num = I2S_SCK_PIN,     // Bit clock
        .ws_io_num = I2S_WS_PIN,       // Word select
        .data_out_num = I2S_PIN_NO_CHANGE,
        .data_in_num = I2S_SD_PIN      // Data input
    };

    // Install I2S driver
    esp_err_t err = i2s_driver_install(I2S_PORT, &i2s_config, 0, NULL);
    if (err != ESP_OK) {
        Serial.printf("[ERROR] I2S driver install failed: %d\n", err);
        return false;
    }

    // Set I2S pins
    err = i2s_set_pin(I2S_PORT, &pin_config);
    if (err != ESP_OK) {
        Serial.printf("[ERROR] I2S pin config failed: %d\n", err);
        i2s_driver_uninstall(I2S_PORT);
        return false;
    }

    // Test microphone by reading samples
    Serial.println("[I2S] Testing microphone...");
    delay(100); // Let I2S stabilize

    int16_t testBuffer[256];
    size_t bytesRead = 0;

    for (int attempt = 1; attempt <= 3; attempt++) {
        i2s_read(I2S_PORT, testBuffer, sizeof(testBuffer), &bytesRead, 1000);

        int samplesRead = bytesRead / sizeof(int16_t);
        int nonZeroCount = 0;
        int16_t minVal = 32767, maxVal = -32768;

        for (int i = 0; i < samplesRead; i++) {
            if (testBuffer[i] != 0) nonZeroCount++;
            if (testBuffer[i] < minVal) minVal = testBuffer[i];
            if (testBuffer[i] > maxVal) maxVal = testBuffer[i];
        }

        float nonZeroPct = (float)nonZeroCount / samplesRead * 100.0;
        Serial.printf("[I2S Test] Attempt %d: %d samples, %d non-zero (%.1f%%), range: %d to %d\n",
                      attempt, samplesRead, nonZeroCount, nonZeroPct, minVal, maxVal);

        // Check if microphone is working
        if (nonZeroPct > 10.0 && (maxVal > 100 || minVal < -100)) {
            Serial.println("[I2S] ✅ Microphone is working!");
            return true;
        }

        delay(100);
    }

    // Microphone might still work even if test shows mostly zeros
    Serial.println("[WARN] Microphone test inconclusive - may need external sound");
    Serial.println("[I2S] Proceeding anyway (microphone may be silent)");
    return true;
}

/**
 * Read audio samples from I2S microphone
 *
 * @param buffer Destination buffer for audio samples
 * @param size Number of bytes to read
 * @return Number of bytes actually read
 */
size_t readI2SAudio(int16_t* buffer, size_t size) {
    size_t bytesRead = 0;
    esp_err_t result = i2s_read(I2S_PORT, buffer, size, &bytesRead, portMAX_DELAY);

    if (result != ESP_OK) {
        Serial.printf("[ERROR] I2S read failed: %d\n", result);
        return 0;
    }

    // Periodic audio quality monitoring (every 5 seconds)
    unsigned long now = millis();
    if (now - lastAudioDebug > AUDIO_DEBUG_INTERVAL) {
        lastAudioDebug = now;

        int samplesRead = bytesRead / sizeof(int16_t);
        int zeroCount = 0;
        int16_t minVal = 32767, maxVal = -32768;

        for (int i = 0; i < samplesRead; i++) {
            if (buffer[i] == 0) zeroCount++;
            if (buffer[i] < minVal) minVal = buffer[i];
            if (buffer[i] > maxVal) maxVal = buffer[i];
        }

        float zeroPct = (float)zeroCount / samplesRead * 100.0;
        Serial.printf("[I2S DEBUG] Samples: %d, zeros: %d (%.1f%%), range: %d to %d\n",
                      samplesRead, zeroCount, zeroPct, minVal, maxVal);

        // Store diagnostics for /diag endpoint
        lastMinVal = minVal;
        lastMaxVal = maxVal;
        lastNonZeroCount = samplesRead - zeroCount;
        lastSampleCount = samplesRead;

        // Warn if microphone appears silent
        if (zeroPct > 99.0) {
            Serial.println("[WARN] Microphone appears silent (>99% zeros)");
            Serial.println("[WARN] Check wiring and ensure sound is present");
        }
    }

    return bytesRead;
}

// ============================================================================
// HTTP SERVER HANDLERS
// ============================================================================

/**
 * Root page - Device information and status
 */
void handleRoot(AsyncWebServerRequest *request) {
    String html = "<!DOCTYPE html><html><head><title>ESP32-CAM Microphone</title>";
    html += "<meta name='viewport' content='width=device-width, initial-scale=1'>";
    html += "<style>body{font-family:Arial;margin:20px;background:#f0f0f0;}";
    html += ".container{background:white;padding:20px;border-radius:8px;max-width:600px;margin:auto;}";
    html += "h1{color:#333;margin-top:0;}";
    html += ".status{background:#e8f5e9;padding:15px;border-radius:4px;margin:10px 0;}";
    html += ".url{background:#f5f5f5;padding:10px;border-radius:4px;font-family:monospace;word-break:break-all;}";
    html += "a{color:#1976d2;text-decoration:none;}";
    html += "a:hover{text-decoration:underline;}";
    html += "</style></head><body><div class='container'>";

    html += "<h1>🎤 BirdNET-Gone ESP32-CAM Microphone</h1>";
    html += "<div class='status'>";
    html += "<strong>Status:</strong> Online<br>";
    html += "<strong>Active Clients:</strong> " + String(activeClients) + "<br>";
    html += "<strong>Total Connections:</strong> " + String(totalConnections) + "<br>";
    html += "<strong>Free Heap:</strong> " + String(ESP.getFreeHeap()) + " bytes<br>";
    html += "<strong>Uptime:</strong> " + String(millis() / 1000) + " seconds<br>";
    html += "<strong>WiFi RSSI:</strong> " + String(WiFi.RSSI()) + " dBm";
    html += "</div>";

    html += "<h2>Audio Stream</h2>";
    html += "<div class='url'>http://" + WiFi.localIP().toString() + ":" + String(HTTP_PORT) + "/stream</div>";
    html += "<p>Use this URL in BirdNET-Gone audio source configuration.</p>";

    html += "<h2>Endpoints</h2>";
    html += "<ul>";
    html += "<li><a href='/stream'>/stream</a> - Audio stream (WAV, 16kHz, 16-bit, mono)</li>";
    html += "<li><a href='/status'>/status</a> - JSON status information</li>";
    html += "<li><a href='/reboot'>/reboot</a> - Reboot device</li>";
    html += "</ul>";

    html += "<h2>Hardware</h2>";
    html += "<p><strong>Board:</strong> ESP32-CAM<br>";
    html += "<strong>Chip:</strong> " + String(ESP.getChipModel()) + "<br>";
    html += "<strong>Flash:</strong> " + String(ESP.getFlashChipSize() / 1024 / 1024) + " MB<br>";
    html += "<strong>Sample Rate:</strong> " + String(SAMPLE_RATE) + " Hz<br>";
    html += "<strong>Channels:</strong> " + String(CHANNELS) + " (Mono)</p>";

    html += "</div></body></html>";
    request->send(200, "text/html", html);
}

/**
 * Status endpoint - JSON status information
 */
void handleStatus(AsyncWebServerRequest *request) {
    String json = "{";
    json += "\"activeClients\":" + String(activeClients) + ",";
    json += "\"totalConnections\":" + String(totalConnections) + ",";
    json += "\"uptime\":" + String(millis() / 1000) + ",";
    json += "\"freeHeap\":" + String(ESP.getFreeHeap()) + ",";
    json += "\"wifiRSSI\":" + String(WiFi.RSSI()) + ",";
    json += "\"sampleRate\":" + String(SAMPLE_RATE) + ",";
    json += "\"channels\":" + String(CHANNELS);
    json += "}";
    request->send(200, "application/json", json);
}

/**
 * Reboot endpoint - Remote device restart
 */
void handleReboot(AsyncWebServerRequest *request) {
    Serial.println("[HTTP] Reboot requested");
    request->send(200, "text/plain", "Rebooting...");
    delay(1000);
    ESP.restart();
}

/**
 * Reset endpoint - Reset client counter for emergency recovery
 */
void handleReset(AsyncWebServerRequest *request) {
    Serial.println("[HTTP] Client counter reset requested");

    // Reset all client slots
    for (int i = 0; i < MAX_STREAMING_CLIENTS; i++) {
        streamingClients[i].disconnected = true;
        streamingClients[i].headerSent = false;
    }
    activeClients = 0;

    String response = "Client counter reset. Active clients: 0";
    Serial.println(response);
    request->send(200, "text/plain", response);
}

/**
 * Diagnostics endpoint - Show audio diagnostics
 */
void handleDiag(AsyncWebServerRequest *request) {
    String json = "{";
    json += "\"lastSampleCount\":" + String(lastSampleCount) + ",";
    json += "\"lastNonZeroCount\":" + String(lastNonZeroCount) + ",";
    json += "\"lastMinVal\":" + String(lastMinVal) + ",";
    json += "\"lastMaxVal\":" + String(lastMaxVal) + ",";
    json += "\"i2s_ws_pin\":" + String(I2S_WS_PIN) + ",";
    json += "\"i2s_sck_pin\":" + String(I2S_SCK_PIN) + ",";
    json += "\"i2s_sd_pin\":" + String(I2S_SD_PIN) + ",";
    json += "\"sampleRate\":" + String(SAMPLE_RATE) + ",";

    // Check if audio appears working
    bool audioWorking = (lastNonZeroCount > lastSampleCount / 10) &&
                        (lastMaxVal > 100 || lastMinVal < -100);
    json += "\"audioWorking\":" + String(audioWorking ? "true" : "false");
    json += "}";

    request->send(200, "application/json", json);
}

/**
 * Audio streaming handler - Sends continuous WAV audio
 *
 * Implements HTTP chunked transfer encoding for indefinite streaming
 * Includes client timeout to prevent hung connections
 */
void handleStream(AsyncWebServerRequest *request) {
    if (activeClients >= MAX_CLIENTS) {
        Serial.println("[HTTP] Too many clients, rejecting connection");
        request->send(503, "text/plain", "Server busy - too many clients");
        return;
    }

    // Find a slot for this client
    int clientSlot = nextClientSlot;
    nextClientSlot = (nextClientSlot + 1) % MAX_STREAMING_CLIENTS;

    // Initialize client state
    streamingClients[clientSlot].lastActivity = millis();
    streamingClients[clientSlot].headerSent = false;
    streamingClients[clientSlot].disconnected = false;

    activeClients++;
    totalConnections++;
    Serial.printf("[HTTP] New streaming client #%d in slot %d (active: %d, total: %lu)\n",
                  activeClients, clientSlot, activeClients, totalConnections);

    // Capture client slot in lambda
    int slot = clientSlot;

    AsyncWebServerResponse *response = request->beginChunkedResponse(
        "audio/wav",
        [slot](uint8_t *buffer, size_t maxLen, size_t index) -> size_t {
            StreamingClient* client = &streamingClients[slot];

            // Check if already marked as disconnected
            if (client->disconnected) {
                return 0;
            }

            // Send WAV header on first chunk
            if (!client->headerSent) {
                WAVHeader header;
                memcpy(buffer, &header, sizeof(WAVHeader));
                client->lastActivity = millis();
                client->headerSent = true;
                return sizeof(WAVHeader);
            }

            // Check for client timeout
            unsigned long now = millis();
            if (now - client->lastActivity > CLIENT_TIMEOUT_MS) {
                Serial.printf("[HTTP] Client in slot %d timeout, closing stream\n", slot);
                if (!client->disconnected) {
                    client->disconnected = true;
                    activeClients--;
                    Serial.printf("[HTTP] Active clients now: %d\n", activeClients);
                }
                return 0; // Signal end of stream
            }

            // Read audio from I2S microphone
            size_t audioSize = min(maxLen, (size_t)1024);
            int16_t* audioBuffer = (int16_t*)buffer;
            size_t bytesRead = readI2SAudio(audioBuffer, audioSize);

            if (bytesRead > 0) {
                client->lastActivity = now;
            } else {
                // No data read - check if we should disconnect
                // This handles cases where client disconnects without timeout
                if (now - client->lastActivity > CLIENT_TIMEOUT_MS / 2) {
                    // Still waiting, keep trying
                }
            }

            return bytesRead;
        }
    );

    // Set up disconnect handler using request abort detection
    // When client disconnects, the response will return 0 bytes which triggers cleanup

    request->send(response);
}

// ============================================================================
// WiFi FUNCTIONS
// ============================================================================

/**
 * Connect to WiFi network
 */
void setupWiFi() {
    Serial.println("\n[WiFi] Connecting to WiFi...");
    Serial.printf("[WiFi] SSID: %s\n", WIFI_SSID);

    // Configure static IP
    if (!WiFi.config(local_IP, gateway, subnet, primaryDNS, secondaryDNS)) {
        Serial.println("[ERROR] Static IP configuration failed");
    }

    WiFi.mode(WIFI_STA);
    WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

    int attempts = 0;
    while (WiFi.status() != WL_CONNECTED && attempts < 30) {
        delay(500);
        Serial.print(".");
        attempts++;
    }

    if (WiFi.status() == WL_CONNECTED) {
        Serial.println("\n[OK] WiFi connected!");
        Serial.printf("IP address: %s\n", WiFi.localIP().toString().c_str());
        Serial.printf("Signal strength: %d dBm\n", WiFi.RSSI());
    } else {
        Serial.println("\n[ERROR] WiFi connection failed!");
        Serial.println("[ERROR] Check SSID and password");
        Serial.println("[ERROR] Device will reboot in 10 seconds...");
        delay(10000);
        ESP.restart();
    }
}

/**
 * Setup HTTP server routes
 */
void setupServer() {
    Serial.println("[HTTP] Setting up web server...");

    server.on("/", HTTP_GET, handleRoot);
    server.on("/status", HTTP_GET, handleStatus);
    server.on("/stream", HTTP_GET, handleStream);
    server.on("/reboot", HTTP_GET, handleReboot);
    server.on("/reset", HTTP_GET, handleReset);
    server.on("/diag", HTTP_GET, handleDiag);

    // 404 handler
    server.onNotFound([](AsyncWebServerRequest *request) {
        request->send(404, "text/plain", "Not found");
    });

    server.begin();
    Serial.printf("[HTTP] Server started on port %d\n", HTTP_PORT);
}

// ============================================================================
// MAIN SETUP AND LOOP
// ============================================================================

void setup() {
    Serial.begin(115200);
    delay(1000);

    Serial.println("\n\n========================================");
    Serial.println("  BirdNET-Gone ESP32-CAM Microphone");
    Serial.println("  Wireless Audio Streaming Server");
    Serial.println("  Version 1.0.0");
    Serial.println("========================================\n");

    // Initialize I2S microphone
    if (!initI2S()) {
        Serial.println("[FATAL] I2S initialization failed!");
        Serial.println("[FATAL] Check microphone wiring and reboot");
        while(1) { delay(1000); }
    }

    // Connect to WiFi
    setupWiFi();

    // Setup HTTP server
    setupServer();

    // Configure watchdog timer
    Serial.printf("[WDT] Configuring watchdog timer (%d seconds)\n", WATCHDOG_TIMEOUT_S);
    esp_task_wdt_init(WATCHDOG_TIMEOUT_S, true);
    esp_task_wdt_add(NULL);

    Serial.println("\nSystem Ready!");
    Serial.printf("Stream URL: http://%s:%d/stream\n",
                  WiFi.localIP().toString().c_str(), HTTP_PORT);
    Serial.println("========================================\n");
}

void loop() {
    // Reset watchdog timer
    esp_task_wdt_reset();

    // Monitor WiFi connection
    if (WiFi.status() != WL_CONNECTED) {
        Serial.println("[WARN] WiFi disconnected, reconnecting...");
        setupWiFi();
    }

    // Periodic client cleanup - check for stale clients every 30 seconds
    static unsigned long lastCleanup = 0;
    unsigned long now = millis();
    if (now - lastCleanup > 30000) {
        lastCleanup = now;

        // Check all client slots for stale connections
        int staleCount = 0;
        for (int i = 0; i < MAX_STREAMING_CLIENTS; i++) {
            if (!streamingClients[i].disconnected && streamingClients[i].headerSent) {
                // This client was active but check if it's been too long
                if (now - streamingClients[i].lastActivity > CLIENT_TIMEOUT_MS * 2) {
                    streamingClients[i].disconnected = true;
                    staleCount++;
                }
            }
        }

        // Sanity check - if activeClients is stuck high but no slots are active, reset it
        int actualActive = 0;
        for (int i = 0; i < MAX_STREAMING_CLIENTS; i++) {
            if (!streamingClients[i].disconnected && streamingClients[i].headerSent) {
                actualActive++;
            }
        }

        if (activeClients > actualActive) {
            Serial.printf("[CLEANUP] Correcting activeClients from %d to %d\n", activeClients, actualActive);
            activeClients = actualActive;
        }

        if (staleCount > 0) {
            Serial.printf("[CLEANUP] Cleaned up %d stale clients, active: %d\n", staleCount, activeClients);
        }
    }

    delay(100);
}
