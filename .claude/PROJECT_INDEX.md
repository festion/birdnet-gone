# Project Index: birdnet-gone
## 1. Core Purpose
BirdNET-Gone is a Go implementation of BirdNET for real-time AI-powered continuous bird sound identification, primarily aimed at non-serious birders and home users. It offers 24/7 analysis, local processing, a web UI, and supports advanced features like Deep Detection and Live Audio Streaming.

## 2. Architecture
The project follows a modular structure:
*   `/cmd/`: Contains Viper CLI commands for various functionalities.
*   `/internal/`: Houses private Go packages, adhering to Go standards, and includes core logic, API v2 definitions, and internal services.
*   `/frontend/`: Implements the Svelte 5 user interface with TypeScript.
*   `/Docker/` and `docker-compose.yml`: Define the Docker-based deployment environment.

## 3. Key Files
*   **Documentation & Guidelines**:
    *   `ARCHITECTURE.md`: High-level architectural overview.
    *   `CHANGELOG.md`: Project change history.
    *   `CONTRIBUTING.md`: Guidelines for contributors.
    *   `CLAUDE.md`: General development guidelines for AI assistants.
    *   `README.md`: Project overview, installation, and development setup.
    *   `TESTING.md`: Test patterns, `testify` usage, and shared helpers for Go tests.
    *   `doc/wiki/guide.md`: User guide with advanced features like Deep Detection.
    *   `doc/DEBUG-COLLECTION.md`, `doc/PROFILING.md`: Debugging and profiling documentation.
    *   `frontend/CLAUDE.md`: Frontend-specific development guidelines for AI assistants.
*   **Configuration**:
    *   `.air.toml`: Configuration for Air (live-reloading development server for Go).
    *   `cliff.toml`: Configuration for `git-cliff` (changelog generator).
    *   `go.mod`, `go.sum`: Go module dependency definitions.
    *   `Taskfile.yml`: Task runner configurations.
    *   `docker-compose.yml`, `Docker/docker-compose.yml`, `Docker/docker-compose.autotls.yml`: Docker Compose configurations for deployment.
    *   `frontend/package.json`: Frontend (npm/yarn) dependency and script definitions.
    *   `frontend/eslint.config.js`, `frontend/tsconfig.json`: Frontend linting and TypeScript configurations.
*   **Source Code & Entry Points**:
    *   `main.go`: Main entry point for the Go application.
    *   `frontend/embed.go`: Embeds frontend assets into the Go binary.
*   **Scripts**:
    *   `install.sh`: Quick installation script.
    *   `scripts/birdnet-health-check.sh`: Health check script.
    *   `Docker/entrypoint.sh`, `Docker/startup-wrapper.sh`: Docker container entrypoint and startup scripts.
    *   `examples/secure-mqtt-test.sh`: Example MQTT testing script.

## 4. Dependencies
*   **Go**: Managed via `go.mod`.
*   **Frontend**: Svelte 5, TypeScript, and other UI-related libraries managed via `frontend/package.json` (npm/yarn).
*   **Containerization**: Docker and Docker Compose for deployment.
*   **Development Tools**:
    *   `task`: For running various project tasks (build, dev server, clean, lint).
    *   `ast-grep`: For syntax-aware code search and refactoring.
    *   `testify`: For Go testing frameworks.

## 5. Common Tasks
*   **Linting**:
    *   Go: `task lint`
    *   Frontend: `npm run check:all`
*   **Testing**:
    *   Go: `go test -race -v`
    *   Frontend: `npm test`
*   **Building**:
    *   Default build: `task`
    *   Frontend only: `task frontend-build`
    *   Cross-platform (e.g., Linux AMD64): `task linux_amd64`
*   **Development Server**: `task dev_server` (with hot reload).
*   **Cleaning**: `task clean` to remove build artifacts.
*   **Code Search & Refactoring**: Utilize `ast-grep` for syntax-aware operations.
*   **PR Review**: Request automated reviews with `gh pr comment <PR_NUMBER> --body "/gemini review"`.
