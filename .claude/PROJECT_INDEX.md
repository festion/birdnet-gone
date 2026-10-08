# Project Index: birdnet-gone

## 1. Core Purpose

BirdNET-Gone is a fork of BirdNET-Go, providing an AI-powered solution for continuous avian monitoring and identification. It's a Go implementation of BirdNET, designed for real-time bird sound identification, primarily for non-serious birders and home users. The project emphasizes local processing, a web UI, and supports various deployment methods.

## 2. Architecture

The codebase follows a clear separation of concerns:

*   **Backend (Go):** Primarily located in the `/internal/` directory for private packages and `/cmd/` for CLI commands using Viper.
*   **Frontend (Svelte 5):** Resides in the `/frontend/` directory, built with Svelte 5 and TypeScript.
*   **API:** New endpoints are added to `internal/api/v2/`. API v1 is not to be expanded.
*   **Deployment:** Utilizes Docker and Podman, with `docker-compose.yml` and `Podman/podman-compose.yml` for orchestration.
*   **Testing:** Uses standard Go testing (`go test`) and `testify`, with `TESTING.md` providing guidelines.

## 3. Key Files

*   **Project Overview & Guidelines:**
    *   `README.md`: Main project overview and installation instructions.
    *   `CLAUDE.md`: General AI development guidelines for the project.
    *   `frontend/CLAUDE.md`: Frontend-specific development guidelines.
    *   `internal/CLAUDE.md`: Backend (Go) specific development guidelines.
    *   `internal/api/v2/CLAUDE.md`: API v2 specific development guidelines.
    *   `ARCHITECTURE.md`: High-level architectural documentation.
    *   `CONTRIBUTING.md`: Contribution instructions.
    *   `CHANGELOG.md`: Project change log.
    *   `TESTING.md`: Testing patterns and usage of `testify`.
*   **Configuration & Build:**
    *   `go.mod`, `go.sum`: Go module dependencies.
    *   `frontend/package.json`, `frontend/package-lock.json`: Frontend dependencies and scripts.
    *   `Taskfile.yml`: Task runner configuration for various builds and operations.
    *   `.golangci.yaml`: Go linter configuration.
    *   `frontend/.ast-grep.yml`, `frontend/doc/AST-GREP-SETUP.md`: `ast-grep` configuration and setup for frontend.
    *   `cliff.toml`: CLI configuration.
    *   `.air.toml`: Configuration for `air` (Go live-reloading).
*   **Source Code Entry Points:**
    *   `main.go`: Main application entry point.
    *   `cmd/root.go`: Main Cobra CLI command definition.
*   **Deployment & Environment:**
    *   `Dockerfile`: Docker image definition.
    *   `docker-compose.yml`, `Docker/docker-compose.yml`, `Podman/podman-compose.yml`: Docker and Podman compose files.
    *   `Docker/ENVIRONMENT_VARIABLES.md`: Documentation for Docker environment variables.
    *   `.devcontainer/devcontainer.json`: Development container setup.
*   **Documentation & Wiki:**
    *   `doc/wiki/*.md`: Various wiki articles (e.g., `installation.md`, `docker_compose_guide.md`).
    *   `doc/*.md`: Other general documentation (e.g., `BUFFER_ALLOCATION_MONITORING.md`).
*   **Data:**
    *   `data/latest.json`: Contains latest data, likely taxonomy or model related.

## 4. Dependencies

*   **Go:** Managed by `go.mod` and `go.sum`. Key internal packages in `/internal/`.
*   **Svelte 5:** Frontend framework, with dependencies managed by `frontend/package.json`.
*   **Viper:** Used for CLI commands in `/cmd/`.
*   **Testify:** Go testing framework used in conjunction with standard `go test`.
*   **Docker/Podman:** For containerized deployment and development.
*   **Task:** A task runner for automating build, linting, and testing processes.
*   **ast-grep:** A structural code search and refactoring tool.

## 5. Common Tasks

*   **Linting:**
    *   Go: `task lint`
    *   Frontend: `npm run check:all` (run from `frontend/` directory)
*   **Testing:**
    *   Go: `go test -race -v ./...` (from root directory)
    *   Frontend: `npm test` (run from `frontend/` directory)
*   **Development Server:**
    *   `task dev_server` (for hot reload)
*   **Building:**
    *   `task` (default build, auto-detects target)
    *   `task frontend-build` (frontend only)
    *   `task linux_amd64` (cross-platform builds)
*   **Cleaning Artifacts:**
    *   `task clean`
*   **Code Search & Refactoring (using ast-grep):**
    *   Search: `ast-grep --pattern "async function $NAME($$$) { $$$ }" src/`
    *   Refactor: `ast-grep --pattern "let $VAR = $VALUE" --rewrite "const $VAR = $VALUE" src/`
*   **Git Workflow:**
    *   Branching: `git pull origin main && git checkout -b feature-name`
    *   Request automated PR review: `gh pr comment <PR_NUMBER> --body "/gemini review"`
*   **Updating Taxonomy:**
    *   `scripts/update_taxonomy.go` (requires Go execution)
*   **Collecting Debug Data:**
    *   `scripts/collect-debug-data.sh`
    *   `scripts/collect-debug-data-docker.sh`
*   **Pushover Notifications:**
    *   `deploy/pushover_notify.sh`
# Project Index: birdnet-gone

## 1. Core Purpose

BirdNET-Gone is a fork of BirdNET-Go, providing an AI-powered solution for continuous avian monitoring and identification. It's a Go implementation of BirdNET, designed for real-time bird sound identification, primarily for non-serious birders and home users. The project emphasizes local processing, a web UI, and supports various deployment methods.

## 2. Architecture

The codebase follows a clear separation of concerns:

*   **Backend (Go):** Primarily located in the `/internal/` directory for private packages and `/cmd/` for CLI commands using Viper.
*   **Frontend (Svelte 5):** Resides in the `/frontend/` directory, built with Svelte 5 and TypeScript.
*   **API:** New endpoints are added to `internal/api/v2/`. API v1 is not to be expanded.
*   **Deployment:** Utilizes Docker and Podman, with `docker-compose.yml` and `Podman/podman-compose.yml` for orchestration.
*   **Testing:** Uses standard Go testing (`go test`) and `testify`, with `TESTING.md` providing guidelines.

## 3. Key Files

*   **Project Overview & Guidelines:**
    *   `README.md`: Main project overview and installation instructions.
    *   `CLAUDE.md`: General AI development guidelines for the project.
    *   `frontend/CLAUDE.md`: Frontend-specific development guidelines.
    *   `internal/CLAUDE.md`: Backend (Go) specific development guidelines.
    *   `internal/api/v2/CLAUDE.md`: API v2 specific development guidelines.
    *   `ARCHITECTURE.md`: High-level architectural documentation.
    *   `CONTRIBUTING.md`: Contribution instructions.
    *   `CHANGELOG.md`: Project change log.
    *   `TESTING.md`: Testing patterns and usage of `testify`.
*   **Configuration & Build:**
    *   `go.mod`, `go.sum`: Go module dependencies.
    *   `frontend/package.json`, `frontend/package-lock.json`: Frontend dependencies and scripts.
    *   `Taskfile.yml`: Task runner configuration for various builds and operations.
    *   `.golangci.yaml`: Go linter configuration.
    *   `frontend/.ast-grep.yml`, `frontend/doc/AST-GREP-SETUP.md`: `ast-grep` configuration and setup for frontend.
    *   `cliff.toml`: CLI configuration.
    *   `.air.toml`: Configuration for `air` (Go live-reloading).
*   **Source Code Entry Points:**
    *   `main.go`: Main application entry point.
    *   `cmd/root.go`: Main Cobra CLI command definition.
*   **Deployment & Environment:**
    *   `Dockerfile`: Docker image definition.
    *   `docker-compose.yml`, `Docker/docker-compose.yml`, `Podman/podman-compose.yml`: Docker and Podman compose files.
    *   `Docker/ENVIRONMENT_VARIABLES.md`: Documentation for Docker environment variables.
    *   `.devcontainer/devcontainer.json`: Development container setup.
*   **Documentation & Wiki:**
    *   `doc/wiki/*.md`: Various wiki articles (e.g., `installation.md`, `docker_compose_guide.md`).
    *   `doc/*.md`: Other general documentation (e.g., `BUFFER_ALLOCATION_MONITORING.md`).
*   **Data:**
    *   `data/latest.json`: Contains latest data, likely taxonomy or model related.

## 4. Dependencies

*   **Go:** Managed by `go.mod` and `go.sum`. Key internal packages in `/internal/`.
*   **Svelte 5:** Frontend framework, with dependencies managed by `frontend/package.json`.
*   **Viper:** Used for CLI commands in `/cmd/`.
*   **Testify:** Go testing framework used in conjunction with standard `go test`.
*   **Docker/Podman:** For containerized deployment and development.
*   **Task:** A task runner for automating build, linting, and testing processes.
*   **ast-grep:** A structural code search and refactoring tool.

## 5. Common Tasks

*   **Linting:**
    *   Go: `task lint`
    *   Frontend: `npm run check:all` (run from `frontend/` directory)
*   **Testing:**
    *   Go: `go test -race -v ./...` (from root directory)
    *   Frontend: `npm test` (run from `frontend/` directory)
*   **Development Server:**
    *   `task dev_server` (for hot reload)
*   **Building:**
    *   `task` (default build, auto-detects target)
    *   `task frontend-build` (frontend only)
    *   `task linux_amd64` (cross-platform builds)
*   **Cleaning Artifacts:**
    *   `task clean`
*   **Code Search & Refactoring (using ast-grep):**
    *   Search: `ast-grep --pattern "async function $$$($$$) { $$$ }" src/`
    *   Refactor: `ast-grep --pattern "let $VAR = $VALUE" --rewrite "const $VAR = $VALUE" src/`
*   **Git Workflow:**
    *   Branching: `git pull origin main && git checkout -b feature-name`
    *   Request automated PR review: `gh pr comment <PR_NUMBER> --body "/gemini review"`
*   **Updating Taxonomy:**
    *   `scripts/update_taxonomy.go` (requires Go execution)
*   **Collecting Debug Data:**
    *   `scripts/collect-debug-data.sh`
    *   `scripts/collect-debug-data-docker.sh`
*   **Pushover Notifications:**
    *   `deploy/pushover_notify.sh`
