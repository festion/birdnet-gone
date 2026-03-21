# Project Index: birdnet-gone

## 1. Core Purpose

This project, BirdNET-Go, is a comprehensive system for real-time bird sound identification. It captures audio, analyzes it using the BirdNET machine learning model, and provides a web-based frontend for users to view detections. The system is designed for deployment on various platforms, including single-board computers (like Raspberry Pi), servers via Docker/Podman, and custom hardware firmware.

## 2. Architecture

The application follows a client-server architecture. The core logic is written in Go (`main.go`, `internal/`) and acts as the backend server. It handles audio processing, BirdNET analysis, and serves a Svelte-based web interface (`frontend/`). The system is containerized using Docker and Podman for easy deployment. It also includes firmware for ESP32 devices, suggesting capabilities for custom hardware microphone clients. For a detailed breakdown, see [ARCHITECTURE.md](ARCHITECTURE.md).

## 3. Key Files

-   **`main.go`**: The main entry point for the Go backend application.
-   **`go.mod`**: Defines the Go module and its dependencies.
-   **`cmd/`**: Contains the command-line interface logic for different application sub-commands.
-   **`internal/`**: Contains the core application logic, separated into packages for different functionalities like audio processing (`myaudio`), BirdNET analysis (`birdnet`), and API services (`api`).
-   **`frontend/`**: The Svelte-based web frontend, including its source code (`src/`) and build configuration (`vite.config.js`).
-   **`Dockerfile` & `docker-compose.yml`**: Files for building and deploying the application using Docker.
-   **`Podman/`**: Contains configurations for running the application with Podman.
-   **`firmware/`**: Contains firmware for ESP32 devices, likely for dedicated audio capture hardware.
-   **`ARCHITECTURE.md`**: Provides a detailed overview of the project's architecture.

## 4. Dependencies

-   **Backend (Go)**: Dependencies are managed in `go.mod`. Key libraries include frameworks for web servers, audio processing, and database interaction.
-   **Frontend (JavaScript/TypeScript)**: Dependencies are managed in `frontend/package.json`. The frontend is built with Svelte and Vite. It includes various libraries for UI components, API requests, and data visualization.
