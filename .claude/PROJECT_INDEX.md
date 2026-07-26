# Project Index: birdnet-gone

## 1. Core Purpose
The `birdnet-gone` project appears to be a comprehensive application focused on bird sound detection and analysis, likely leveraging the BirdNET model. Its core purpose involves:
*   **Audio Analysis:** Processing audio to identify bird species.
*   **Data Management:** Storing and managing analysis results and related data (e.g., eBird integration).
*   **User Interface:** Providing a web-based frontend for interacting with the analysis results.
*   **Deployment Versatility:** Supporting deployment across various environments, including Docker, Podman, and potentially embedded systems (firmware) or virtual machines.
*   **Monitoring and Notifications:** Offering features for system monitoring, telemetry, and various notification mechanisms.

## 2. Architecture
The codebase exhibits a modular, multi-component architecture, primarily written in Go for the backend and Svelte/JavaScript for the frontend.

*   **Go Backend:**
    *   **`cmd/`**: Contains the entry points for various commands and services.
    *   **`internal/`**: Houses the core business logic, including:
        *   `analysis`: For processing audio data and running detection models.
        *   `api`: Defines the RESTful API endpoints, likely including `v2`.
        *   `birdnet`: Specific integration and logic related to the BirdNET model.
        *   `conf`: Configuration management.
        *   `datastore`: Data persistence logic.
        *   `events`: Eventing system for inter-component communication.
        *   `logger`: Centralized logging.
        *   `mqtt`: MQTT integration for messaging.
        *   `notification`: Handles various notification types (e.g., webhooks, push).
        *   `observability`/`telemetry`: For metrics, tracing, and health checks.
        *   `security`: Authentication and authorization.
    *   **`main.go`**: The primary application entry point.
*   **Frontend (`frontend/`)**: A Svelte-based web application providing the user interface, built with Vite.
*   **Containerization**: `Docker/` and `Podman/` directories indicate support for containerized deployment using `docker-compose` or `podman-compose`.
*   **Firmware (`firmware/`)**: Suggests integration with embedded devices, specifically ESP32.
*   **VM Images (`vm-images/`)**: Tools and configurations for building virtual machine images.
*   **Utilities (`scripts/`)**: Various shell and Python scripts for debugging, health checks, and system management.
*   **Documentation (`doc/`, `docs/`)**: Extensive documentation covering architecture, profiling, and specific component details.

## 3. Key Files

*   `main.go`: Main application entry point for the Go backend.
*   `ARCHITECTURE.md`: High-level architectural overview of the project.
*   `go.mod`, `go.sum`: Go module dependency management.
*   `Dockerfile`: Defines the Docker image for the application.
*   `docker-compose.yml`, `Podman/podman-compose.yml`: Container orchestration definitions.
*   `frontend/package.json`, `frontend/package-lock.json`: Frontend (Node.js/Svelte) dependency management.
*   `frontend/src/`: Source code for the Svelte frontend application.
*   `internal/api/README.md`: Documentation for the backend API.
*   `internal/analysis/`: Contains logic for audio analysis.
*   `internal/birdnet/`: Contains logic specific to BirdNET integration.
*   `internal/conf/config.yaml`: Main configuration file.
*   `internal/logger/docs/LOGGER_DOCUMENTATION_INDEX.md`: Entry point for logging documentation.
*   `internal/security/README.md`: Documentation for security aspects.
*   `internal/telemetry/README.md`: Documentation for telemetry and observability.
*   `vicohome-bridge/vicohome_bridge.py`, `vicohome-bridge/requirements.txt`: Python script and dependencies for VicoHome integration.
*   `watchdog/watchdog.py`: Python watchdog script.

## 4. Dependencies
*   **Go Modules**: Managed by `go.mod` and `go.sum`.
*   **Node.js/NPM**: For the frontend, specified in `frontend/package.json` (Svelte, Vite, Playwright, Vitest).
*   **Python**: Used for scripts (e.g., `scripts/analyze-debug-data.py`, `vicohome-bridge/vicohome_bridge.py`, `watchdog/watchdog.py`) with dependencies potentially listed in `vicohome-bridge/requirements.txt`.
*   **Docker/Podman**: For containerization and deployment.
*   **Systemd**: For service management, as indicated by `scripts/systemd/` and recovery service files.
*   **eBird API**: For taxonomy and observation data, as indicated by `internal/ebird/`.
