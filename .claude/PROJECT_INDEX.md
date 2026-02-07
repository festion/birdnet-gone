# birdnet-gone Project Index

Generated: 2026-02-07

## Purpose

The `birdnet-gone` project is a comprehensive application likely focused on bird sound analysis, identification, and display. It integrates a Go backend for core logic and APIs, a Python-based display system for local presentation, and a Svelte-based frontend for a web interface. Its goal is to provide a robust platform for collecting, processing, and visualizing bird observations, potentially for environmental monitoring or research.

## Architecture

The project follows a modular architecture:

*   **Go Backend:** The core application logic resides in Go (`main.go`, `cmd/`, `internal/`). `main.go` serves as the entry point, orchestrating services. `cmd/` contains subcommands for CLI operations. `internal/` holds the main business logic, including modules for BirdNET integration (`internal/birdnet`), API handling (`internal/api`), MQTT communication (`internal/mqtt`), audio processing (`internal/myaudio`), and various other utilities.
*   **Python Display:** The `display/` directory contains Python scripts (`birdnet_display.py`, `location_manager.py`) responsible for local display functionality, potentially managing physical screens, rendering information, and handling location services.
*   **Frontend (Svelte):** The `frontend/` directory houses a Svelte-based web application. This provides a user interface for interacting with the Go backend via HTTP APIs.
*   **Containerization:** `Docker/` and `Podman/` directories contain configurations (`docker-compose.yml`, `podman-compose.yml`, `Dockerfile`, `entrypoint.sh`) for deploying the application using Docker or Podman, facilitating isolated and scalable deployments.
*   **CLI:** Command-line tools defined under `cmd/` provide administrative and operational capabilities.

## Key Files

*   **`.air.toml`**: Configuration for Air, a live-reloading tool for Go applications during development.
*   **`ARCHITECTURE.md`**: Project architecture documentation.
*   **`CHANGELOG.md`**: Records changes made to the project over time.
*   **`.claude/learnings.md`**: AI-specific learning notes or documentation.
*   **`CLAUDE.md`**: General project information or instructions for AI assistants.
*   **`.claude/PROJECT_INDEX.md`**: This file, providing an overview for AI assistants.
*   **`cliff.toml`**: Configuration for `git-cliff`, a tool for generating changelogs.
*   **`CONTRIBUTING.md`**: Guidelines for contributing to the project.
*   **`data/latest.json`**: Possibly stores the latest bird observation data or model information.
*   **`.devcontainer/devcontainer.json`**: Configuration for Visual Studio Code Dev Containers, enabling a consistent development environment.
*   **`.devcontainer/postCreateCommand.sh`**: Script executed after a Dev Container is created.
*   **`display/ap_setup.sh`**: Script for setting up access point functionalities for the display.
*   **`display/birdnet_display_enhanced.py`**: An enhanced Python script for displaying BirdNET results.
*   **`display/birdnet_display.py`**: The main Python script for displaying BirdNET results.
*   **`display/cache_builder.py`**: Python script to build display caches.
*   **`display/DEPLOYMENT_RESULTS.md`**: Documentation on display deployment results.
*   **`display/ENHANCED_SETTINGS_DEPLOYMENT.md`**: Documentation for enhanced display settings deployment.
*   **`display/install.sh`**: Installation script for the display component.
*   **`display/kiosk_launcher.sh`**: Script to launch the display in kiosk mode.
*   **`display/KIOSK_MODE_STATUS.md`**: Documentation on kiosk mode status.
*   **`display/location_manager.py`**: Python script for managing location services for the display.
*   **`display/LOCATION_MANAGER_README.md`**: README for the location manager.
*   **`display/LOCATION_MANAGER_SUMMARY.md`**: Summary of the location manager.
*   **`display/README.md`**: README for the display component.
*   **`display/run.sh`**: Script to run the display component.
*   **`display/utils/config_manager.py`**: Python utility for managing display configurations.
*   **`display/utils/geolocation.py`**: Python utility for geolocation features.
*   **`docker-compose.yml`**: Docker Compose file for defining and running multi-container Docker applications.
*   **`Docker/docker-compose.autotls.yml`**: Docker Compose file with automatic TLS configuration.
*   **`Docker/docker-compose.yml`**: Docker Compose file specific to the Docker deployment directory.
*   **`Docker/entrypoint.sh`**: Entrypoint script for Docker containers.
*   **`Docker/ENVIRONMENT_VARIABLES.md`**: Documentation for environment variables used in Docker.
*   **`Docker/startup-wrapper.sh`**: Wrapper script for application startup within Docker.
*   **`doc/BUFFER_ALLOCATION_MONITORING.md`**: Documentation on buffer allocation monitoring.
*   **`doc/DEBUG-COLLECTION.md`**: Documentation on collecting debug data.
*   **`doc/PROFILING.md`**: Documentation on application profiling.
*   **`docs/plans/2026-02-06-http-audio-streaming.md`**: Planning document for HTTP audio streaming.
*   **`doc/wiki/building.md`**: Wiki page on building the project.
*   **`doc/wiki/cloudflare_tunnel_guide.md`**: Wiki page for Cloudflare Tunnel setup.
*   **`doc/wiki/docker_compose_guide.md`**: Wiki page for Docker Compose guide.
*   **`doc/wiki/esp32-mediamtx-setup.md`**: Wiki page for ESP32 MediaMTX setup.
*   **`doc/wiki/guide.md`**: General project guide.
*   **`doc/wiki/hardware.md`**: Wiki page on hardware considerations.
*   **`doc/wiki/index.md`**: Wiki index page.
*   **`doc/wiki/installation.md`**: Wiki page on installation procedures.
*   **`doc/wiki/rtsp-troubleshooting.md`**: Wiki page for RTSP troubleshooting.
*   **`doc/wiki/security.md`**: Wiki page on security aspects.
*   **`doc/wiki/telemetry.md`**: Wiki page on telemetry.
*   **`go.mod`**: Go module definition file, listing direct and indirect dependencies.
*   **`go.sum`**: Checksum verification for Go modules.
*   **`Makefile`**: Standard build automation script.
*   **`main.go`**: The primary entry point for the Go application.
*   **`frontend/package.json`**: Defines frontend project metadata and JavaScript dependencies.
*   **`frontend/svelte.config.js`**: Configuration for the SvelteKit framework.
*   **`Taskfile.yml`**: Task runner configuration, potentially offering more advanced task automation than Makefile.

## Dependencies

*   **Go:** `go.mod` defines all Go language dependencies.
*   **Python:** `display/requirements.txt` specifies Python dependencies for the display component.
*   **JavaScript/Node.js:** `frontend/package.json` lists frontend (SvelteKit, Vite) and development dependencies (ESLint, Prettier, Playwright, Vitest).
*   **Docker/Podman:** For containerized deployment.
*   **Air:** For live-reloading Go applications during development (`.air.toml`).
*   **Git-cliff:** For changelog generation (`cliff.toml`).
*   **Mockery:** For generating Go mocks (`.mockery.yaml`).
*   **GolangCI-Lint:** For Go static analysis (`.golangci.yaml`).
*   **Bash/Shell utilities:** Used in various scripts (`.sh` files).

## Common Tasks

*   **Build:**
    *   **Go Backend:** `go build ./cmd/birdnet-go` or `make build` (if defined in Makefile)
    *   **Frontend:** Navigate to `frontend/` and run `npm install` then `npm run build`.
*   **Test:**
    *   **Go Backend:** `go test ./...`
    *   **Frontend:** Navigate to `frontend/` and run `npm test` or `npm run test:integration`.
*   **Run/Deploy:**
    *   **Go Backend (Development):** `air` (from the project root, if Air is installed and configured).
    *   **Docker:** `docker-compose up -d` (from `Docker/` directory).
    *   **Podman:** `podman-compose up -d` (from `Podman/` directory).
    *   **Display:** `cd display && ./install.sh && ./run.sh` (or via systemd service if configured).
