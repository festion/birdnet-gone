# Project Index: birdnet-gone

## 1. Core Purpose
The `birdnet-gone` project appears to be a comprehensive system for real-time bird sound detection, analysis, and visualization. It integrates a Go-based backend with a web frontend, a Python-based local display/kiosk application, and potentially embedded firmware components, aiming to provide an accessible platform for environmental sound monitoring and bird identification.

## 2. Architecture
The codebase follows a multi-component architecture:
-   **Backend (Go):** The core application logic resides in `main.go`, `cmd/`, and `internal/`. It handles audio analysis, data storage, API services, MQTT communication, and notifications.
-   **Frontend (Svelte/Vite):** Located in `frontend/`, this is a modern web application built with Svelte and managed by Vite, providing a user interface for interaction and data display.
-   **Display Application (Python):** The `display/` directory contains Python scripts and related files for a dedicated local display or kiosk mode, potentially for physical installations like Raspberry Pi devices.
-   **Containerization:** `Docker/` and `Podman/` directories, along with `Dockerfile` and `docker-compose.yml`, indicate robust support for containerized deployments.
-   **Embedded Systems:** `firmware/` suggests the project interacts with or provides firmware for embedded devices (e.g., ESP32) for audio capture or other functionalities.
-   **Virtual Machines:** `vm-images/` provides infrastructure for building and deploying virtual machine images, offering another deployment method.

## 3. Key Files
-   `ARCHITECTURE.md`: High-level architectural overview of the project.
-   `main.go`: Entry point for the main Go application.
-   `go.mod`, `go.sum`: Go module definition and dependency lock file.
-   `frontend/package.json`: Frontend project dependencies and scripts.
-   `frontend/src/`: Source code for the Svelte frontend application.
-   `display/birdnet_display.py`: Main script for the Python display application.
-   `display/requirements.txt`: Python dependencies for the display application.
-   `Dockerfile`, `docker-compose.yml`: Docker configurations for building and orchestrating services.
-   `Podman/podman-compose.yml`: Podman configurations for container orchestration.
-   `internal/`: Contains core Go packages for various functionalities (e.g., `analysis`, `api`, `birdnet`, `datastore`, `mqtt`).
-   `cmd/`: Defines command-line interface subcommands.
-   `install.sh`, `podman-install.sh`: Installation scripts for the project.
-   `vm-images/`: Contains Packer templates and scripts for building VM images.
-   `LICENSES.md`: Project licensing information.
-   `CONTRIBUTING.md`: Guidelines for contributing to the project.
-   `README.md`: General project overview and setup instructions.

## 4. Dependencies
-   **Go:** Managed via `go.mod` and `go.sum`.
-   **Node.js/JavaScript:** For the frontend, managed by `frontend/package.json` and `frontend/package-lock.json` (Svelte, Vite, various npm packages).
-   **Python:** For the `display/` and `vicohome-bridge/` components, dependencies are listed in `display/requirements.txt` and `vicohome-bridge/requirements.txt` respectively.
-   **Docker/Podman:** For containerization, relying on standard Docker/Podman tools and images.
-   **Packer:** For `vm-images/` to build virtual machine images.
