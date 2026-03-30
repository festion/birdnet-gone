# Project Index: birdnet-gone

## 1. Core Purpose
The `birdnet-gone` project appears to be a comprehensive system for bird sound detection and analysis. It likely involves real-time audio processing, data storage, a web-based frontend for visualization, and potentially integration with display systems and embedded devices. Its goal is to identify bird species from audio input, manage observations, and provide a platform for monitoring and analysis.

## 2. Architecture
The project exhibits a multi-component architecture:
*   **Go Backend:** The core application logic, API services, and data processing are implemented in Go, organized within `main.go`, `cmd/`, and `internal/` packages. This handles audio analysis (BirdNET integration), data management, and potentially MQTT communication.
*   **Python Display/Utility:** A Python-based display system (`display/`) suggests local visualization, kiosk mode operation, and location management, likely for physical installations. There's also `vicohome-bridge` which might integrate with smart home devices.
*   **Web Frontend:** A Svelte-based web interface (`frontend/`) provides user interaction, data display, and configuration capabilities.
*   **Deployment:** Docker and Podman configurations (`Docker/`, `Podman/`, `docker-compose.yml`, `Dockerfile`) indicate containerized deployment for the various services.
*   **Embedded Systems:** The `firmware/` directory suggests integration with ESP32 microcontrollers for specific functionalities like RTSP streaming or keepalives.
*   **VM Images:** The `vm-images/` directory indicates a system for building and managing virtual machine images of the application.

## 3. Key Files
*   `main.go`: Main entry point for the Go application.
*   `go.mod`, `go.sum`: Go module dependency definitions.
*   `Dockerfile`: Docker container definition for the main application.
*   `docker-compose.yml`: Docker Compose configuration for multi-service deployment.
*   `Podman/podman-compose.yml`: Podman Compose configuration.
*   `frontend/package.json`, `frontend/package-lock.json`: Frontend (Svelte) project dependencies and scripts.
*   `frontend/src/`: Frontend source code.
*   `display/birdnet_display.py`: Main script for the Python display component.
*   `display/requirements.txt`: Python dependencies for the display component.
*   `internal/birdnet/`: Go package for BirdNET integration and audio analysis.
*   `internal/api/`: Go package for API definitions and handlers.
*   `internal/datastore/`: Go package for data storage logic.
*   `internal/mqtt/`: Go package for MQTT communication.
*   `cmd/root.go`: Cobra CLI root command definition.
*   `ARCHITECTURE.md`: High-level architectural overview.
*   `README.md`: Project overview and setup instructions.
*   `CONTRIBUTING.md`: Guidelines for contributors.
*   `LICENSE`: Project license.
*   `PRIVACY.md`: Privacy policy document.
*   `.devcontainer/devcontainer.json`: Development container configuration.
*   `vm-images/build.sh`: Script to build VM images.
*   `scripts/collect-debug-data.sh`: Script for collecting debug information.
*   `watchdog/watchdog.py`: Python script for system monitoring/watchdog functionality.

## 4. Dependencies
*   **Go:** Go modules (`go.mod`).
*   **Python:** `requirements.txt` in `display/` and `vicohome-bridge/` define Python package dependencies.
*   **JavaScript/Node.js:** `package.json` in `frontend/` specifies npm packages (e.g., Svelte, Vite, Playwright).
*   **Containerization:** Docker, Podman.
*   **CLI Framework:** Cobra (Go).
*   **Web Framework:** Svelte (JavaScript).
*   **Embedded Frameworks:** ESP-IDF or similar for ESP32 firmware.
