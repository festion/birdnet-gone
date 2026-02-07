I have created the `PROJECT_INDEX.md` file in the `.claude` directory as you requested. Let me know if you have any other questions.
ct is a Go-based application for analyzing audio streams to identify bird species using the BirdNET model. It features a web-based frontend for real-time monitoring and visualization, supports various hardware (ESP32), and offers multiple deployment options including Docker and Podman.

## 2. Architecture

The system is composed of a Go backend and a Svelte frontend.

*   **Backend (`main.go`, `internal/`, `cmd/`)**: The Go application handles audio processing, BirdNET analysis, and serves a web API. It's structured into `cmd` for command-line interfaces and `internal` for core logic.
*   **Frontend (`frontend/`)**: A Svelte application provides the user interface, which communicates with the Go backend. It is bundled and embedded into the Go binary for distribution.
*   **Firmware (`firmware/`)**: Contains code for ESP32 microcontrollers, likely for audio capture.
*   **Deployment (`Docker/`, `Podman/`, `vm-images/`)**: The project supports containerized deployments with Docker and Podman, as well as virtual machine images.

## 3. Key Files

*   `main.go`: The main entry point for the backend application.
*   `go.mod`: Defines the Go module and its dependencies.
*   `frontend/svelte.config.js`: Configuration for the Svelte frontend.
*   `frontend/package.json`: Lists frontend dependencies and scripts.
*   `docker-compose.yml`: Defines the Docker services for development and production.
*   `ARCHITECTURE.md`: Provides a detailed overview of the project's architecture.

## 4. Dependencies

*   **Backend (Go)**: The primary Go dependencies are defined in `go.mod`. Key libraries include modules for audio processing, web serving, and interacting with the BirdNET model.
*   **Frontend (JavaScript)**: The frontend dependencies are managed via npm and listed in `frontend/package.json`. The core framework is Svelte, with various plugins and libraries for UI components and development tools.
