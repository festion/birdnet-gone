# Project Index: birdnet-gone

## 1. Core Purpose
The `birdnet-gone` project appears to be a system for BirdNET soundscape analysis, likely involving real-time audio processing, a web-based frontend for display and configuration, and potentially embedded device integration (firmware). It aims to identify bird species from audio input, manage locations, and visualize data, possibly with a focus on ease of deployment and monitoring.

## 2. Architecture
The project follows a modular architecture:
- **Backend (Go):** The `main.go` and `cmd/` directory suggest a Go application handling core logic, command-line interfaces, and potentially API services. The `internal/` directory contains various packages for analysis, API, BirdNET integration, data storage, monitoring, MQTT communication, audio processing, and security.
- **Frontend (Svelte/JavaScript):** The `frontend/` directory contains a web application built with Svelte (indicated by `svelte.config.js` and `vite.config.js`), likely providing a user interface for configuration, data visualization, and real-time monitoring.
- **Display/Kiosk (Python):** The `display/` directory contains Python scripts (`birdnet_display.py`, `location_manager.py`) and service files (`birdnet-location-manager.service`, `Relaunch-Kiosk.desktop`), suggesting a dedicated display component, possibly for a kiosk-like setup, handling visual output and location management.
- **Firmware (ESP32):** The `firmware/` directory with `esp32-c6-rtsp` and `esp32-keepalive` indicates integration with ESP32 microcontrollers, likely for audio acquisition, streaming (RTSP), or system health monitoring on edge devices.
- **Containerization:** `Dockerfile`, `docker-compose.yml`, `Podman/` files suggest containerized deployment options for various components.
- **Logging:** The `logs/` directory indicates comprehensive logging for different parts of the application.
- **VM Images:** `vm-images/` implies pre-configured virtual machine images for deployment.

## 3. Key Files
- `ARCHITECTURE.md`: Provides a high-level overview of the system design.
- `main.go`: The main entry point for the Go backend application.
- `go.mod`, `go.sum`: Go module definition and dependency lock file.
- `frontend/package.json`, `frontend/package-lock.json`: Frontend (JavaScript) dependency management.
- `frontend/svelte.config.js`, `frontend/vite.config.js`: Frontend build and configuration files.
- `display/birdnet_display.py`, `display/location_manager.py`: Python scripts for the display and location management.
- `display/requirements.txt`: Python dependencies for the display component.
- `Dockerfile`, `docker-compose.yml`: Docker configurations for building and running the application.
- `Podman/podman-compose.yml`: Podman configurations for container orchestration.
- `internal/`: Contains core Go packages for various functionalities (analysis, API, birdnet integration, data, security, etc.).
- `firmware/`: Contains code for embedded systems (ESP32).
- `vm-images/`: Contains scripts and templates for building VM images.
- `PRIVACY.md`: Details the project's privacy policy.
- `TESTING.md`: Documentation on testing procedures.
- `CONTRIBUTING.md`: Guidelines for contributors.

## 4. Dependencies
- **Go:** Managed by `go.mod` and `go.sum`.
- **JavaScript/Node.js:** Managed by `frontend/package.json` and `frontend/package-lock.json`.
- **Python:** Managed by `display/requirements.txt` and `vicohome-bridge/requirements.txt`.
- **Docker/Podman:** Used for containerization and orchestration.
- **Svelte/Vite:** Frontend framework and build tool.
- **ESP-IDF (implied):** For ESP32 firmware development within the `firmware/` directory.
I have generated the `PROJECT_INDEX.md` file and saved it in the root directory of the project.
