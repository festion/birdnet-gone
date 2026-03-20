# Project Index: birdnet-gone

## 1. Core Purpose
The `birdnet-gone` project appears to be a system for real-time bird sound analysis and identification, likely integrating a Go backend with a Svelte frontend for display and management. It includes components for audio processing, data storage, notifications, and various deployment mechanisms (Docker, Podman, VM images).

## 2. Architecture
The project follows a modular architecture:
-   **Backend (Go):** Core logic resides in `main.go` and the `internal/` directory, handling audio analysis (`birdnet`, `myaudio`), API services (`api`), data storage (`datastore`), notifications (`notification`), and other system-level functionalities. Commands are defined in `cmd/`.
-   **Frontend (Svelte):** Located in `frontend/`, this provides a web-based user interface for interacting with the backend, potentially displaying analysis results and system status.
-   **Display (Python):** The `display/` directory contains Python scripts (`birdnet_display.py`, `location_manager.py`) and associated services, suggesting a separate display application or kiosk mode, possibly for Raspberry Pi or similar devices.
-   **Containerization:** `Docker/` and `Podman/` directories, along with `docker-compose.yml` and `Dockerfile`, indicate containerized deployment options for the entire application.
-   **Firmware:** The `firmware/` directory suggests integration with embedded devices (ESP32) for tasks like RTSP streaming or keepalives.
-   **VM Images:** `vm-images/` contains Packer configurations and scripts for building virtual machine images, indicating a deployment strategy for virtualized environments.

## 3. Key Files
-   `ARCHITECTURE.md`: Provides an overview of the system's design.
-   `main.go`: The main entry point for the Go backend application.
-   `go.mod`, `go.sum`: Go module dependency definitions.
-   `frontend/package.json`, `frontend/package-lock.json`: Frontend (Svelte) project dependencies and scripts.
-   `frontend/src/`: Contains the source code for the Svelte frontend.
-   `display/birdnet_display.py`: Main script for the birdnet display application.
-   `display/requirements.txt`: Python dependencies for the display application.
-   `internal/`: Houses the core Go modules and business logic.
-   `cmd/`: Defines the command-line interface structure for the Go application.
-   `Dockerfile`, `docker-compose.yml`: Docker configuration for containerized deployment.
-   `Podman/podman-compose.yml`: Podman configuration for containerized deployment.
-   `vm-images/build.sh`, `vm-images/*.pkr.hcl`: Scripts and configurations for building virtual machine images.
-   `vicohome-bridge/vicohome_bridge.py`: Python script for Vicohome camera integration.
-   `soundscape.wav`, `tawnyowl.wav`: Example audio files for testing or demonstration.

## 4. Dependencies
-   **Go:** `go.mod` defines Go language dependencies.
-   **Node.js/npm:** `frontend/package.json` defines JavaScript/TypeScript dependencies for the Svelte frontend (e.g., Svelte, Vite, Playwright).
-   **Python:** `display/requirements.txt` and `vicohome-bridge/requirements.txt` define Python dependencies (e.g., for `birdnet_display.py`, `location_manager.py`, `vicohome_bridge.py`).
-   **Docker/Podman:** Used for containerization and orchestration.
-   **Packer:** Used in `vm-images/` for building VM images.
-   **Shell scripts:** Various `.sh` files for installation, deployment, and utility tasks.
