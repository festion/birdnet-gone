# Project Index: festion__birdnet-gone
## 1. Core Purpose
This project is a Go-based implementation of the BirdNET AI model for real-time, continuous bird sound identification. It's designed for home users and hobbyists to monitor and analyze avian activity from a soundcard input, running on low-resource hardware like a Raspberry Pi. The system operates locally without needing an internet connection and provides a Svelte-based web UI for data visualization.

## 2. Architecture
The application is a monolith composed of two main parts:
- **Backend**: Written in Go, located in `/internal/`. It handles audio capture, AI-based analysis using an embedded Tensorflow Lite model, data storage (SQLite/MySQL), and serves the API. CLI commands are defined in `/cmd/` using Viper. New API endpoints must be added to `/internal/api/v2/`.
- **Frontend**: A Svelte 5 single-page application located in `/frontend/`. It's written in TypeScript and provides the user interface for visualizing detection data.
- **Deployment**: The primary deployment method is via Docker, with configurations available in `docker-compose.yml` and the `Docker/` directory.

Development guidelines, especially the strict use of `ast-grep` for refactoring and `testify` for testing, are documented in `CLAUDE.md` and `TESTING.md`.

## 3. Key Files
| Path | Description |
|---|---|
| `README.md` | General overview, installation, and project status. |
| `CLAUDE.md` | **Critical:** Development guidelines, project structure, and universal rules. |
| `TESTING.md` | Go testing patterns and `testify` usage conventions. |
| `Taskfile.yml` | Defines common build, development, and linting tasks. |
| `main.go` | Main application entry point for the Go backend. |
| `go.mod` | Defines Go backend modules and dependencies. |
| `internal/` | Contains all private Go packages for the backend logic. |
| `frontend/` | Root directory for the Svelte 5 frontend application. |
| `frontend/package.json` | Defines frontend dependencies and scripts. |
| `docker-compose.yml` | Primary Docker configuration for deployment. |
| `Docker/` | Contains additional Docker-related configurations and scripts. |

## 4. Dependencies
| Type | Dependency | Usage |
|---|---|---|
| **Backend** | Go | Primary language for the backend. |
| | Tensorflow Lite | The AI model for bird sound identification, embedded in the binary. |
| | Testify | The required assertion library for all Go tests. |
| **Frontend**| Svelte 5 | The JavaScript framework for the web UI. |
| | TypeScript | Primary language for the frontend. |
| | Vite | Frontend build tool and development server. |
| **Tooling** | Docker | Used for containerization and simplified deployment. |
| | Task | The main task runner for builds, linting, and other scripts. |
| | ast-grep | **Required** tool for syntax-aware code search and refactoring. |

## 5. Common Tasks
| Task | Command | Environment |
|---|---|---|
| Run default build | `task` | Backend (Go) |
| Run dev server with hot-reload | `task dev_server` | Backend & Frontend |
| Run linter | `task lint` | Backend (Go) |
| Run all checks (lint, format) | `npm run check:all` | Frontend (Svelte) |
| Run tests | `go test -race -v` | Backend (Go) |
| Run tests | `npm test` | Frontend (Svelte) |
| Search code (syntax-aware) | `ast-grep --pattern "..."` | Both |
| Refactor code (syntax-aware) | `ast-grep --pattern "..." --rewrite "..."` | Both |
