# Project Index: birdnet-gone
## 1. Core Purpose
BirdNET-Go is a Go-based application for real-time, continuous bird sound detection and identification using the BirdNET AI model. It is designed for home users and hobbyists to monitor avian activity from audio streams, featuring a Svelte-based web UI for data visualization. The system processes audio locally and does not require internet connectivity for its core functionality.

## 2. Architecture
The project follows a client-server architecture:
-   **Backend**: A Go application responsible for audio processing, AI inference with a Tensorflow Lite model, and serving the API. Key directories are `/cmd` for CLI commands and `/internal` for private Go packages.
-   **Frontend**: A Svelte 5 single-page application located in `/frontend`. It communicates with the Go backend's API for data visualization and control.
-   **Deployment**: The application is containerized using Docker, with configurations defined in `Dockerfile` and `docker-compose.yml`.

Refer to `ARCHITECTURE.md` for a more detailed overview.

## 3. Key Files
-   `main.go`: The main entry point for the Go application.
-   `go.mod`: Defines the Go backend dependencies.
-   `Taskfile.yml`: Contains build, development, and linting tasks managed by the Task runner.
-   `docker-compose.yml`: Defines services for Docker-based deployment.
-   `frontend/package.json`: Defines Node.js dependencies and scripts for the Svelte frontend.
-   `CLAUDE.md`: Top-level development guidelines, including project structure and rules.
-   `frontend/CLAUDE.md`: Frontend-specific development guidelines.
-   `internal/CLAUDE.md`: Backend-specific development guidelines.
-   `CONTRIBUTING.md`: Guidelines for contributors.

## 4. Dependencies
-   **Backend**: Go, Viper, Tensorflow Lite.
-   **Frontend**: Svelte 5, TypeScript, Vite.
-   **Build & Tooling**: Task, Docker, `ast-grep` (for syntax-aware search and refactoring).

## 5. Common Tasks
-   **Run full build**: `task`
-   **Run development server with hot reload**: `task dev_server`
-   **Run backend linters**: `task lint`
-   **Run backend tests**: `go test -race -v ./...`
-   **Run frontend checks (lint, format, type-check)**: `cd frontend && npm run check:all`
-   **Run frontend tests**: `cd frontend && npm test`
-   **Search code (syntax-aware)**: `ast-grep --pattern "<pattern>"`
