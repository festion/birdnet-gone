# Project Index: birdnet-gone

## 1. Core Purpose
The `birdnet-gone` project appears to be a system for real-time or batch bird sound analysis, identification, and display. It likely processes audio input (e.g., from microphones, `.wav` files), identifies bird species using a BirdNET model, and then visualizes these observations, possibly on a local display or web interface. It also includes components for location management, data storage, and integration with external bird observation platforms like eBird.

## 2. Architecture
The project exhibits a multi-component architecture:
*   **Backend (Go):** The core logic, API services, and data processing are likely implemented in Go, indicated by `main.go`, `go.mod`, and the `internal/` package structure (e.g., `analysis`, `api`, `birdnet`, `datastore`, `mqtt`, `observation`, `serviceapi`). It handles audio processing, bird identification, and data management.
*   **Frontend (Svelte/TypeScript):** A web-based user interface is built using Svelte, Vite, and TypeScript (`frontend/`). This provides a dashboard or control panel for the system.
*   **Display/Client (Python):** Python scripts (`display/birdnet_display.py`, `display/location_manager.py`) handle local display functionalities, kiosk mode, and potentially interact with hardware.
*   **Containerization:** Docker and Podman configurations (`Docker/`, `Podman/`, `Dockerfile`, `docker-compose.yml`) suggest the application is designed for containerized deployment.
*   **Firmware:** The `firmware/` directory indicates support for embedded systems, possibly for audio capture devices (e.g., ESP32-C6).
*   **VM Images:** `vm-images/` suggests the project can be deployed on virtual machines with pre-configured environments.
*   **Utility Scripts:** `scripts/` contains various shell and Go scripts for debugging, data collection, and updates.

## 3. Key Files
*   `ARCHITECTURE.md`: Provides a high-level overview of the system's design.
*   `main.go`: The main entry point for the Go backend application.
*   `go.mod`: Go module definition, specifying dependencies.
*   `frontend/package.json`: Frontend project metadata and JavaScript/TypeScript dependencies.
*   `frontend/src/`: Source code for the Svelte frontend application.
*   `display/birdnet_display.py`: Main Python script for the local display component.
*   `display/location_manager.py`: Python script for managing location services for the display.
*   `Dockerfile`: Defines the Docker image for the main application.
*   `docker-compose.yml`: Docker Compose configuration for multi-service deployments.
*   `Podman/podman-compose.yml`: Podman Compose configuration.
*   `install.sh`: Installation script for the project.
*   `Taskfile.yml`: Task runner configuration.
*   `internal/birdnet/`: Contains BirdNET integration logic.
*   `internal/observation/`: Handles bird observation data.
*   `ARCHITECTURE.md`: Project architecture documentation.
*   `CHANGELOG.md`: Records changes and versions.
*   `CONTRIBUTING.md`: Guidelines for contributions.
*   `README.md`: Main project documentation.
*   `TESTING.md`: Information about testing procedures.
*   `PRIVACY.md`: Privacy policy document.
*   `LICENSES.md`: License information.

## 4. Dependencies
*   **Go Modules:** Specified in `go.mod` and `go.sum`.
*   **Node.js/npm:** Managed by `frontend/package.json` for frontend development (Svelte, Vite, Playwright, Vitest, ESLint, Prettier, PostCSS, Stylelint).
*   **Python Libraries:** Listed in `display/requirements.txt` and `vicohome-bridge/requirements.txt` for Python components (e.g., `numpy`, `tensorflow`, `pyaudio`, `pyqt5`).
*   **Docker/Podman:** For containerized deployment.
*   **Shell Utilities:** Various bash commands and tools used in `.sh` scripts.
