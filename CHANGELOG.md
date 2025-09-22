# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Changed

- **Development Environment**: Added `vim`, `nano`, `make`, `cmake`, `python`, `nodejs-lts-iron`, `npm`, `gdb`, `curl`, and `wget` to the `Dockerfile` to provide a more complete development environment for the agent.

## [0.2.0] - 2025-09-21

### Added

- **React+Vite Frontend**: Migrated the entire frontend from vanilla JS to a modern React+Vite stack, scaffolded with the official Cloudflare CLI for future deployment to Cloudflare Workers.
- **SSH Key Management UI**: Added a new toolbar to the frontend for generating, downloading, and copying ed25519 SSH keys.
- **SSH Key Generation Tool**: Implemented a new `ssh_key_gen` tool on the backend to support the new UI feature.
- **Direct Tool Execution**: Implemented a new JSON-based protocol and a backend command dispatcher that allows the frontend to execute deterministic tools directly, bypassing the LLM for significantly improved performance and responsiveness.

### Changed

- **Docker SDK Integration**: Replaced all `docker exec` shell commands with the official Docker Go SDK, making the agent's interaction with the container more robust and efficient.
- **Summarization-Based Memory**: Replaced the naive, full-history conversation memory with a more sophisticated and token-efficient summarization mechanism.
- **UI/UX Improvements**: Moved all status messages to a fixed bottom status bar, and implemented a robust WebSocket auto-reconnection mechanism.

### Fixed

- **Container Environment**: Added `coreutils` to the `Dockerfile` to ensure essential shell commands are available.
- **Vite Development Server**: Configured the Vite proxy to correctly forward WebSocket requests to the Go backend during local development.
- Corrected numerous CSS layout bugs related to overlapping elements.

## [0.1.0] - 2025-09-21

### Added

- Initial project structure with Go modules.
- `agent.go` as the main application entry point.
- Hot-reloading support using `Air`.
- `Dockerfile` for containerized development.
- Database migration setup with `golang-migrate/migrate` and a `Makefile`.
- Initial implementation of AI agent persona management.
- Project tracking files: `.windsurf_plan.md` and `CHANGELOG.md`.

### Changed

- Refactored the project to use an externally provided Postgres database, removing the `db` service from `docker-compose.yml`.
- Moved the main application file from `cmd/agent/agent.go` to `agent.go` in the project root.
- Updated `Makefile`, `.air.toml`, and `Dockerfile` to reflect the new project structure.
- Pivoted the Docker environment's purpose from running the Go agent to providing a general-purpose Arch Linux development environment. The `Dockerfile` now sets up a container with `base-devel`, `git`, `openssh`, and a non-root `developer` user.
- Removed the `docker-compose` dependency. The agent now manages the container lifecycle using direct `docker` CLI commands.
