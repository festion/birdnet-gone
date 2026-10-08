# Project Index: birdnet-gone
## 1. Core Purpose
BirdNET-Go is an AI-powered solution for continuous avian monitoring and identification. It is a Go implementation of BirdNET for real-time bird sound identification, aimed at non-serious birders and home users. It utilizes the BirdNET AI model for local processing, does not require internet connectivity, and provides a web UI for data visualization.

## 2. Architecture
The project follows a modular structure:
- `/cmd/`: Contains Viper CLI commands for various functionalities.
- `/internal/`: Houses private Go packages, including core logic for birdnet, API, audio processing, and other internal services.
- `/pkg/`: Intended for public Go packages, though currently less utilized.
- `/frontend/`: Contains the Svelte 5 UI, built with TypeScript, providing the web interface.
- `/Docker/` & `docker-compose.yml`: Provides configurations and scripts for Docker-based deployments.

## 3. Key Files
- `README.md`: Project overview, installation instructions, and high-level repository structure.
- `ARCHITECTURE.md`: Details the overall system architecture.
- `CHANGELOG.md`: Records changes and new features across different versions.
- `CONTRIBUTING.md`: Guidelines for contributing to the project, including development setup.
- `CLAUDE.md`: Top-level development guidelines, project overview, and pointers to more specific CLAUDE.md files for frontend, backend, and API v2.
- `frontend/CLAUDE.md`: Specific development guidelines for the Svelte 5 frontend, TypeScript, and UI aspects.
- `internal/CLAUDE.md`: Specific development guidelines for Go backend code, including standards and testing practices.
- `TESTING.md`: Provides patterns, usage of `testify`, and shared helpers for writing tests.
- `go.mod`, `go.sum`: Manage Go module dependencies.
- `frontend/package.json`, `frontend/package-lock.json`: Manage Node.js and frontend dependencies.
- `Taskfile.yml`: Defines various development, build, and linting tasks.
- `Dockerfile`, `docker-compose.yml`: Docker build and orchestration configurations.
- `doc/`: Contains various documentation files, including buffer allocation, debugging, profiling, and a wiki.
- `.devcontainer/devcontainer.json`: Configuration for development containers.

## 4. Dependencies
- **Go**: The primary backend language. Dependencies are managed via Go Modules (`go.mod`, `go.sum`).
- **Svelte 5 / TypeScript**: The core technologies for the frontend UI. Dependencies are managed via npm (`frontend/package.json`).
- **Docker / Docker Compose**: Used for containerized deployments and development environments.
- **Task (taskfile.dev)**: A task runner used to automate build, test, and linting processes.
- **ast-grep**: A syntax-aware tool used for searching and refactoring code.

## 5. Common Tasks
- **Linting**:
    - Go: `task lint`
    - Frontend: `npm run check:all`
- **Testing**:
    - Go: `go test -race -v ./...`
    - Frontend: `npm test`
- **Building**:
    - Default build (auto-detects target): `task`
    - Development server with hot reload: `task dev_server`
    - Frontend only build: `task frontend-build`
    - Clean artifacts: `task clean`
    - Cross-platform builds (e.g., Linux AMD64): `task linux_amd64`
- **Code Search & Refactoring**: Use `ast-grep` for syntax-aware operations over `grep`/`sed`.
- **Branch Management**: Always branch from an updated `main`: `git pull origin main && git checkout -b feature-name`.
- **PR Review**: Request automated reviews by commenting `/gemini review` on a PR.
