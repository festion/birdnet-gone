# Project Index: birdnet-gone

## 1. Core Purpose
The `birdnet-gone` project is an application designed for real-time bird sound detection and analysis. It processes audio input to identify bird species, offering features for data visualization and interaction through a web interface, and potentially integrating with IoT devices for edge processing.

## 2. Architecture
The project follows a modular architecture:
*   **Backend**: Primarily developed in Go, handling core logic, data processing (birdnet, audio, spectrogram), API services, and integrations (e.g., MQTT, eBird, weather).
*   **Frontend**: A web-based user interface built with Svelte (JavaScript/TypeScript), providing data visualization and user interaction.
*   **Python Components**: Includes scripts for display, analysis, and integration with specific home automation systems (VicoHome bridge) and a watchdog.
*   **Deployment**: Utilizes Docker and Podman for containerized deployment, with `docker-compose.yml` and `podman-compose.yml` defining the service orchestration.
*   **Firmware**: Contains code for ESP32 devices, suggesting potential for edge computing or specific hardware integrations.

## 3. Key Files
*   `ARCHITECTURE.md`: High-level architectural overview.
*   `main.go`: Main entry point for the Go backend application.
*   `go.mod`, `go.sum`: Go module dependency definitions.
*   `frontend/`: Directory containing the Svelte web frontend.
*   `frontend/package.json`: Frontend project dependencies and scripts.
*   `frontend/src/`: Frontend source code.
*   `internal/birdnet/`, `internal/myaudio/`, `internal/spectrogram/`: Core Go packages for bird sound analysis and audio processing.
*   `cmd/root.go`: Defines the root command for the CLI application.
*   `docker-compose.yml`, `Dockerfile`: Docker configuration for local development and deployment.
*   `Podman/podman-compose.yml`: Podman configuration for deployment.
*   `display/birdnet_display.py.backup`: Python script for display functionality.
*   `vicohome-bridge/vicohome_bridge.py`: Python script for VicoHome integration.
*   `firmware/esp32-c6-rtsp/`, `firmware/esp32-keepalive/`: Embedded firmware projects for ESP32 devices.
*   `scripts/`: Contains various utility and maintenance scripts.
*   `README.md`: General project overview and instructions.
*   `CONTRIBUTING.md`: Guidelines for contributing to the project.
*   `TESTING.md`: Information regarding project testing.
*   `PRIVACY.md`: Project privacy policy.
*   `.devcontainer/devcontainer.json`: Configuration for development containers.
*   `vm-images/build.sh`: Script for building virtual machine images.

## 4. Dependencies
*   **Go**: Primary language for the backend services and CLI tools.
*   **Svelte**: JavaScript framework used for the web frontend, along with associated tools like Vite, Playwright, and Vitest.
*   **Python**: Used for various scripts, including display utilities and integrations (e.g., `vicohome-bridge`), with dependencies managed via `requirements.txt`.
*   **Docker/Podman**: Containerization platforms used for packaging and deploying the application components.
*   **ESP-IDF (or similar)**: Expected toolchain for building the ESP32 firmware (inferred).
