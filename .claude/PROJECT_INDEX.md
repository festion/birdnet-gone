# Project Index: birdnet-gone

## 1. Core Purpose

The `birdnet-gone` project appears to be a Go-based application for bird sound detection and analysis, likely leveraging the BirdNET platform. It includes components for real-time analysis, data display, and a web-based user interface. It aims to provide a comprehensive solution for environmental audio monitoring, specifically focusing on avian species.

## 2. Architecture

The project follows a multi-component architecture:

*   **Go Backend:** The core application (`main.go`, `cmd/`, `internal/`) written in Go, handling audio analysis, data processing, API services, and integrations (e.g., MQTT, Ebird).
*   **Python Display/Location Manager:** A Python application (`display/`) responsible for local display functionality, managing location, and potentially interacting with hardware.
*   **Web Frontend:** A Svelte-based web interface (`frontend/`) for user interaction, configuration, and visualization of analysis results.
*   **Containerization:** Utilizes Docker and Podman (`Dockerfile`, `docker-compose.yml`, `Podman/`) for packaging and deployment across various environments.
*   **Firmware:** Includes ESP32 firmware (`firmware/`) for potential custom hardware integrations or edge devices.

## 3. Key Files

*   `ARCHITECTURE.md`: High-level architectural overview.
*   `main.go`: Main entry point for the Go application.
*   `go.mod`, `go.sum`: Go module dependency management.
*   `cmd/`: Contains subcommands for the Go CLI.
*   `internal/`: Core business logic and internal Go packages.
*   `display/birdnet_display.py`: Main Python script for the display component.
*   `display/location_manager.py`: Python script for managing device location.
*   `display/requirements.txt`: Python dependencies for the display component.
*   `frontend/package.json`: Frontend (Svelte) project dependencies and scripts.
*   `frontend/svelte.config.js`: Svelte framework configuration.
*   `Dockerfile`: Docker build instructions for the main application.
*   `docker-compose.yml`: Docker Compose configuration for multi-service deployment.
*   `Podman/podman-compose.yml`: Podman Compose configuration.
*   `firmware/`: Directory for ESP32 firmware projects.
*   `data/latest.json`: Possibly stores latest analysis or configuration data.
*   `README.md`: Project overview and setup instructions.
*   `CLAUDE.md`: Claude-specific documentation or project notes.
*   `CONTRIBUTING.md`: Guidelines for contributing to the project.

## 4. Dependencies

*   **Go:** Managed via `go.mod` and `go.sum`. Includes various standard and third-party Go libraries for system interaction, networking, data handling, and potentially audio processing.
*   **Python:**
    *   `display/requirements.txt`: Dependencies for the display component (e.g., `Flask`, `numpy`, `scipy`, `pyaudio`, `BirdNET-Analyzer`).
    *   `vicohome-bridge/requirements.txt`: Specific dependencies for the Vicohome bridge.
*   **JavaScript/TypeScript (Frontend):** Managed via `frontend/package.json`. Includes `Svelte`, `Vite`, `Playwright`, `ESLint`, `Prettier`, and other frontend build and testing tools.
*   **Container Runtimes:** Docker or Podman for deploying the application services.
*   **System Libraries:** Potential reliance on system-level audio libraries or hardware interfaces depending on deployment environment.
