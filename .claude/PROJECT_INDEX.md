# Project Index: vicohome-poller

## 1. Core Purpose
The `vicohome-poller` worktree, within the broader `birdnet-gone` project, appears to be focused on real-time bird sound analysis and detection. It likely integrates with a `vicohome` system, potentially acting as a data polling or bridge component to ingest audio or environmental data for analysis, and providing a web-based user interface for interaction and visualization of observations.

## 2. Architecture
The project employs a polyglot architecture:
*   **Backend**: Primarily Go-based (`main.go`, `cmd/`, `internal/`) handling core logic, data processing (e.g., `birdnet`, `myaudio`, `spectrogram`), API services (`internal/api`, `internal/serviceapi`), and various integrations (e.g., `mqtt`, `ebird`, `notification`).
*   **Frontend**: A Svelte-based web application (`frontend/`) managed with Node.js/npm, providing the user interface for monitoring and control.
*   **Containerization**: Utilizes Docker (`Docker/`, `Dockerfile`, `docker-compose.yml`) and Podman (`Podman/`) for deployment and environment management.
*   **VicoHome Integration**: A Python-based bridge (`vicohome-bridge/`) likely handles specific interactions or data fetching from `vicohome` devices.
*   **Firmware**: Includes embedded code for ESP32 devices (`firmware/esp32-keepalive`), suggesting potential hardware interaction or data acquisition at the edge.
*   **Scripting & Automation**: Various shell and Python scripts (`scripts/`) for maintenance, debugging, and system interactions.
*   **VM Images**: Packer (`vm-images/`) is used to build virtual machine images, indicating planned deployment to VMs.

## 3. Key Files
*   `main.go`: Main entry point for the Go application.
*   `go.mod`, `go.sum`: Go module dependency management.
*   `frontend/package.json`, `frontend/package-lock.json`: Frontend (Node.js/Svelte) dependency management and project configuration.
*   `vicohome-bridge/requirements.txt`: Python dependencies for the `vicohome-bridge`.
*   `Dockerfile`, `docker-compose.yml`: Docker build instructions and multi-container application definitions.
*   `Podman/podman-compose.yml`: Podman equivalent for container orchestration.
*   `ARCHITECTURE.md`, `README.md`, `CONTRIBUTING.md`, `TESTING.md`: Core project documentation.
*   `internal/birdnet/`: Contains core BirdNET integration logic.
*   `internal/vicohome/`: Contains core VicoHome integration logic.
*   `cmd/`: CLI command definitions for the Go application.
*   `Taskfile.yml`: Task runner configuration.
*   `frontend/src/`: Frontend source code.

## 4. Dependencies
*   **Go Modules**: Managed by `go.mod` (e.g., various internal packages, external Go libraries).
*   **Node.js/NPM**: For frontend development, defined in `frontend/package.json` (e.g., Svelte, Vite, Playwright, ESLint).
*   **Python**: For `vicohome-bridge` (`requirements.txt`) and various scripts (e.g., `analyze-debug-data.py`).
*   **Docker/Podman**: Containerization platforms.
*   **Bash**: For various shell scripts.
*   **Packer**: For VM image creation (`vm-images/`).
*   **Git**: For version control.
