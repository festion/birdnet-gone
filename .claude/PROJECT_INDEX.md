# Project Index: birdnet-gone

## 1. Core Purpose
The `birdnet-gone` project appears to be a robust application for bird sound detection and analysis. It integrates a Go backend with a web-based frontend and a Python-driven display system, likely for real-time soundscape monitoring and visualization of bird detections. The project seems to handle audio processing, data storage, and user interface for interacting with BirdNET-related functionalities, potentially including embedded system components (firmware) and containerized deployments (Docker/Podman).

## 2. Architecture
The codebase exhibits a multi-component architecture:
*   **Core Backend (Go):** Written in Go, indicated by `main.go`, `go.mod`, and the `internal/` directory containing various Go packages for analysis, API, data storage, monitoring, MQTT, audio processing, and more. It likely provides the primary logic for BirdNET integration and data management.
*   **Command-Line Interface (CLI):** The `cmd/` directory suggests a CLI application with subcommands for authors, benchmark, directory, file, license, notify, rangefilter, realtime, and support, built around a `root.go`.
*   **Web Frontend:** Located in `frontend/`, this is a Svelte-based application (`svelte.config.js`, `vite.config.js`, `package.json`) responsible for the user interface.
*   **Display System (Python):** The `display/` directory contains Python scripts (`birdnet_display.py`, `location_manager.py`) and service files, likely for driving a dedicated display unit, potentially a Kiosk mode setup.
*   **Containerization:** `Docker/`, `Podman/`, `docker-compose.yml`, and `Dockerfile` indicate strong support for containerized deployment.
*   **Firmware:** The `firmware/` directory suggests integration with embedded systems, specifically ESP32 microcontrollers for RTSP and keepalive functionalities.
*   **VM Images:** The `vm-images/` directory contains scripts and templates for building virtual machine images, indicating support for VM deployments.

## 3. Key Files
*   `./.devcontainer/postCreateCommand.sh`
*   `./.devcontainer/devcontainer.json`
*   `./vm-images/README.md`
*   `./vm-images/templates/meta-data.yml`
*   `./vm-images/templates/user-data.yml`
*   `./vm-images/scripts/configure-services.sh`
*   `./vm-images/scripts/cleanup.sh`
*   `./vm-images/scripts/setup-birdnet-go.sh`
*   `./vm-images/build.sh`
*   `./.claude/learnings.md`
*   `./.claude/PROJECT_INDEX.md`
*   `./PRIVACY.md`
*   `./frontend/I18N_VALIDATION.md`
*   `./frontend/.jscpd.json`
*   `./frontend/node_modules/keyv/README.md`
*   `./frontend/node_modules/keyv/package.json`
*   `./frontend/node_modules/es-object-atoms/README.md`
*   `./frontend/node_modules/es-object-atoms/tsconfig.json`
*   `./frontend/node_modules/es-object-atoms/package.json`
*   `./frontend/node_modules/es-object-atoms/.github/FUNDING.yml`
*   `./frontend/node_modules/es-object-atoms/CHANGELOG.md`
*   `./frontend/node_modules/mimic-function/readme.md`
*   `./frontend/node_modules/mimic-function/package.json`
*   `./frontend/node_modules/normalize-path/README.md`
*   `./frontend/node_modules/normalize-path/package.json`
*   `./frontend/node_modules/parse-ms/readme.md`
*   `./frontend/node_modules/parse-ms/package.json`
*   `./frontend/node_modules/locate-character/README.md`
*   `./frontend/node_modules/locate-character/package.json`
*   `./frontend/node_modules/siginfo/README.md`
*   `./frontend/node_modules/siginfo/.travis.yml`
*   `./frontend/node_modules/siginfo/package.json`
*   `./frontend/node_modules/internmap/README.md`
*   `./frontend/node_modules/internmap/package.json`
*   `./frontend/node_modules/safe-regex/README.md`
*   `./frontend/node_modules/safe-regex/.travis.yml`
*   `./frontend/node_modules/safe-regex/package.json`
*   `./frontend/node_modules/safe-regex/CHANGELOG.md`
*   `./frontend/node_modules/lilconfig/readme.md`
*   `./frontend/node_modules/lilconfig/package.json`
*   `./frontend/node_modules/filing-cabinet/node_modules/commander/Readme.md`
*   `./frontend/node_modules/filing-cabinet/node_modules/commander/package.json`
*   `./frontend/node_modules/filing-cabinet/node_modules/commander/package-support.json`
*   `./frontend/node_modules/filing-cabinet/README.md`
*   `./frontend/node_modules/filing-cabinet/package.json`
*   `./frontend/node_modules/d3-timer/README.md`
*   `./frontend/node_modules/d3-timer/package.json`
*   `./frontend/node_modules/ws/README.md`
*   `./frontend/node_modules/ws/package.json`
*   `./frontend/node_modules/string_decoder/README.md`

## 4. Dependencies
*   **Go:** Dependencies are managed via `go.mod` and `go.sum`.
*   **Frontend (Node.js/npm):** Dependencies are managed via `frontend/package.json` and `frontend/package-lock.json`.
*   **Display (Python):** Python dependencies are specified in `display/requirements.txt`.
*   **VicoHome Bridge (Python):** Python dependencies for the VicoHome bridge are specified in `vicohome-bridge/requirements.txt`.
