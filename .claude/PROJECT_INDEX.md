# Project Index: birdnet-gone

## 1. Core Purpose
The `birdnet-gone` project appears to be a system for real-time or near real-time bird sound analysis, identification, and display. It likely processes audio input, identifies bird species using a BirdNET integration, and provides a user interface (frontend) for displaying observations, potentially with location management and weather integration. It supports containerized deployments (Docker/Podman) and custom hardware (firmware for ESP32).

## 2. Architecture
The project follows a client-server architecture:
*   **Backend (Go):** The `main.go` and `internal/` directories indicate a Go application serving as the core backend. It handles audio processing (`myaudio`), BirdNET integration (`birdnet`), data storage (`datastore`), API management (`api`), observation handling (`observation`), and various other services (e.g., `mqtt`, `notification`, `weather`, `birdweather`).
*   **Frontend (Svelte/JS):** The `frontend/` directory contains a web-based user interface, likely built with Svelte (indicated by `svelte.config.js` and `package.json` contents). This UI interacts with the Go backend via an API.
*   **Display (Python):** The `display/` directory suggests a separate Python-based display application, possibly for kiosks or dedicated screens, handling location management and showing bird detections.
*   **Containerization:** Docker (`Dockerfile`, `docker-compose.yml`, `Docker/`) and Podman (`Podman/`) configurations indicate the application is designed for containerized deployment.
*   **Firmware:** The `firmware/` directory suggests custom firmware for ESP32 devices, potentially for audio acquisition or remote monitoring.
*   **VM Images:** `vm-images/` indicates support for creating virtual machine images for deployment.

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
*   **Go Modules:** `go.mod` and `go.sum` define Go dependencies for the backend.
*   **Frontend (Node.js/npm):** `frontend/package.json` and `frontend/package-lock.json` list JavaScript/TypeScript dependencies for the Svelte frontend.
*   **Python:** `display/requirements.txt` and `vicohome-bridge/requirements.txt` specify Python dependencies for the display application and the VicoHome bridge.
*   **Docker/Podman:** `docker-compose.yml`, `Docker/docker-compose.yml`, `Podman/podman-compose.yml` define service dependencies and configurations for containerized deployments.
