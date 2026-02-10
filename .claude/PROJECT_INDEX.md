I have created the `PROJECT_INDEX.md` file as requested.
urpose

The `birdnet-gone` project is a comprehensive solution for real-time bird sound detection and analysis. It integrates the BirdNET-Lite deep learning model to identify bird species from audio streams, provides various interfaces for displaying observations (a local Python-based kiosk display and a modern web frontend), and manages collected data. The system is designed for robust operation, often deployed on edge devices, to continuously monitor and report avian activity.

## Architecture

The project employs a modular architecture, primarily consisting of a Go backend, a Python-based display, and a Svelte/Vite web frontend.

*   **Go Backend (`main.go`, `cmd/`, `internal/`):** This is the core application responsible for:
    *   **Audio Ingestion & Processing (`internal/myaudio`):** Handles receiving and processing audio data.
    *   **BirdNET Analysis (`internal/birdnet`):** Integrates and runs the BirdNET-Lite model for species identification.
    *   **Observation Management (`internal/observation`):** Stores and manages detection events.
    *   **API (`internal/api`):** Exposes RESTful endpoints for the frontend and other services to retrieve data and control the application.
    *   **MQTT (`internal/mqtt`):** Facilitates real-time communication and notifications.
    *   **Data Storage (`internal/datastore`):** Persists application data.
    *   **CLI (`cmd/`):** Provides command-line utilities for various operations.
*   **Python Display (`display/`):** A lightweight, often Raspberry Pi-targeted, Python application that visualizes bird detections, potentially in a kiosk mode. It communicates with the Go backend to fetch observations.
*   **Web Frontend (`frontend/`):** A modern Single Page Application (SPA) built with Svelte and Vite. It offers a rich user interface for monitoring detections, configuring the system, and viewing historical data, interacting with the Go backend via its API.
*   **Containerization (`Docker/`, `Podman/`):** The entire application stack can be deployed using Docker or Podman, with `docker-compose.yml` and `podman-compose.yml` defining the multi-service deployment.

## Key Files

*   `./.air.toml`: Configuration for