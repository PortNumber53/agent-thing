# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

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
