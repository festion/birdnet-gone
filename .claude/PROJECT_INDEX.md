# Project Index: birdnet-gone

## 1. Core Purpose
The `birdnet-gone` project appears to be a bird sound analysis and detection system, likely capable of real-time processing and display of bird observations. It integrates a backend written in Go with a web-based frontend and potentially includes components for embedded systems or local displays.

## 2. Architecture
The project follows a modular architecture:
- **Go Backend**: The core application logic is implemented in Go, with `main.go` as the entry point and various modules organized under `cmd/` (CLI commands) and `internal/` (internal packages for analysis, API, data storage, etc.).
- **Frontend**: A web-based user interface is developed using JavaScript/TypeScript (Svelte/Vite based on `frontend/package.json`, `frontend/svelte.config.js`, `frontend/vite.config.js`).
- **Display Module**: A Python-based display system (`display/`) likely handles local visualization of analysis results.
- **Containerization**: Docker and Podman configurations (`Docker/`, `Podman/`, `Dockerfile`, `docker-compose.yml`) are used for deployment and environment setup.
- **Firmware**: The `firmware/` directory suggests integration with embedded devices (e.g., ESP32 for RTSP or keepalive functions).
- **CLI**: The `cmd/` directory indicates a command-line interface for various functionalities.

## 3. Key Files
- `main.go`: Main entry point for the Go application.
- `go.mod`: Go module definition, managing Go dependencies.
- `frontend/index.html`: Main HTML file for the web frontend.
- `frontend/package.json`: Manages frontend dependencies and scripts.
- `display/birdnet_display.py`: Script for the birdnet display system.
- `Dockerfile`: Defines the Docker image for the application.
- `docker-compose.yml`: Orchestrates multi-container Docker applications.
- `ARCHITECTURE.md`: Provides architectural overview of the project.
- `README.md`: General project information and setup instructions.
- `internal/birdnet/`: Contains core BirdNET analysis logic.
- `cmd/root.go`: Defines the root command for the CLI.

## 4. Dependencies
- **Go**: Dependencies are managed via `go.mod` and `go.sum`.
- **Frontend (JavaScript/TypeScript)**: Dependencies are managed via `frontend/package.json` and `frontend/package-lock.json`.
- **Python**: Dependencies for the display module are in `display/requirements.txt`, and for the VicoHome bridge in `vicohome-bridge/requirements.txt`.
- **Docker/Podman**: Utilizes Docker/Podman for containerization, as defined in `Dockerfile`, `docker-compose.yml`, `Podman/podman-compose.yml`.
