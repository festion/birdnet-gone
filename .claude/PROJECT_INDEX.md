# Project Index: birdnet-gone

## 1. Core Purpose
The `birdnet-gone` project appears to be an application focused on bird sound detection and analysis. It integrates a Go backend with a web-based frontend, potentially including real-time audio processing, data storage, and visualization. The project also seems to support embedded systems (ESP32) for audio capture or streaming and offers command-line tools for various functionalities.

## 2. Architecture
The codebase exhibits a multi-component architecture:
- **Go Backend:** The core logic, API services, and CLI commands are implemented in Go, organized under `cmd/` and `internal/` directories. This includes modules for analysis, API, birdnet integration, data storage, monitoring, MQTT, audio processing, notifications, and spectrogram generation.
- **Svelte/Vite Frontend:** A web-based user interface is developed using Svelte and Vite, located in the `frontend/` directory, providing interactive data visualization and control.
- **Python Components:** Auxiliary Python scripts are found in `display/` (possibly for visualization or post-processing) and `vicohome-bridge/` (suggesting integration with smart home devices).
- **Embedded Firmware:** The `firmware/` directory contains code for ESP32 devices, indicating hardware integration for audio acquisition or related tasks.
- **Containerization:** The project leverages Docker and Podman (`Docker/`, `Podman/`, `Dockerfile`, `docker-compose.yml`) for consistent deployment and environment management.
- **Documentation & Tools:** Comprehensive documentation (`doc/`, `docs/`) and various scripts (`scripts/`) support development, debugging, and deployment.

## 3. Key Files
- `main.go`: The main entry point for the Go application.
- `go.mod`, `go.sum`: Go module dependency definitions.
- `frontend/index.html`: The main entry point for the web frontend.
- `frontend/package.json`, `frontend/package-lock.json`: Frontend dependency management.
- `Dockerfile`: Defines the Docker image for the application.
- `docker-compose.yml`: Docker Compose configuration for multi-container deployment.
- `Podman/podman-compose.yml`: Podman Compose configuration.
- `ARCHITECTURE.md`: Provides high-level architectural overview.
- `CONTRIBUTING.md`: Guidelines for contributing to the project.
- `README.md`: General project overview and setup instructions.
- `internal/`: Contains core Go packages and business logic.
- `cmd/`: Defines the command-line interface structure and commands.
- `display/birdnet_display_original.py`: A Python script likely for displaying BirdNET results.
- `vicohome-bridge/vicohome_bridge.py`: Python script for integrating with VicoHome.
- `vm-images/`: Contains resources for building virtual machine images.
- `scripts/`: Various utility scripts for analysis, health checks, and data collection.

## 4. Dependencies
- **Go Modules:** Managed by `go.mod` and `go.sum`.
- **NPM Packages:** For the frontend, managed by `frontend/package.json` and `frontend/package-lock.json`.
- **Python Packages:** For `vicohome-bridge/`, managed by `vicohome-bridge/requirements.txt`.
- **Container Runtimes:** Docker or Podman for deployment.
- **System Dependencies:** Potentially various system libraries and tools, depending on the Go and Python components.
