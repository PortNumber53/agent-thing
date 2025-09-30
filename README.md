# Agent Thing

An AI-powered developer agent with a Go backend, a React/Vite frontend, and a Docker-based development sandbox. The agent exposes tools (shell, file read/write, Docker lifecycle, SSH keygen, etc.) and chats over WebSocket.

## Architecture

- Backend (Go):
  - Web server on `:8080` with WebSocket endpoint at `/ws`.
  - Serves the frontend during development.
  - Manages a long-lived Docker container to execute tools in a controlled environment.
  - Integrates with Google Gemini via `internal/llm` with per-minute rate limiting.
- Frontend (React + Vite):
  - Dev server proxies `/ws` to the Go server.
  - Chat UI to converse with the agent and execute tools.
- Docker:
  - `Dockerfile` builds the development environment image used by the agent.
  - On Apple Silicon, the base image is pinned to amd64 for compatibility.

## Requirements

- Go 1.21+
- Node.js 20+ and npm
- Docker Desktop

## Configuration

Create a config file at:

`$HOME/.config/agent-thing/config.ini`

Example contents:

```ini
[default]
DB_HOST=localhost
DB_PORT=5432
DB_NAME=agent
DB_USER=agent
DB_PASSWORD=agent
DB_SSLMODE=disable

# Gemini
GEMINI_API_KEY=your_api_key
GEMINI_MODEL=gemini-1.5-pro
GEMINI_RPM=10

# Host directory mounted into the dev container as /home/developer
CHROOT_DIR=/tmp/agent-thing-chroot
```

Notes:

- The application reads this exact path at startup.
- On first run, the agent will `sudo chown -Rv 1000:1000` the `CHROOT_DIR` to match the container user.

## Run the backend

```bash
go run agent.go
```

This will:

- Build the Docker image (first run may take a while)
- Create/start the `dev-environment` container
- Start the WebSocket server on `http://localhost:8080/ws`

## Run the frontend (dev)

```bash
cd frontend
npm install
npm run dev
```

Open the URL printed by Vite (typically `http://localhost:5173`). The Vite dev server proxies `/ws` to `http://localhost:8080/ws`.

## Database migrations

Use the built-in migration subcommands from the Go app:

```bash
# Create timestamped up/down SQL files under db/migrations
go run ./agent.go migrate create <name>

# Apply or rollback migrations using the DSN from config.ini
go run ./agent.go migrate up
go run ./agent.go migrate down
go run ./agent.go migrate status
```

## Tools available (agent side)

Implemented in `internal/tools/` and available through chat or toolbar buttons:

- conversation: plain text responses
- shell: run shell commands in the persistent container session
- file_read, file_write, file_list: file utilities in `/home/developer`
- ssh_key_gen: generate an SSH keypair inside the container
- docker_start, docker_stop, docker_rebuild, docker_status: container lifecycle
- autonomous_execution: multi-step tool orchestration

## Frontend UI

- Top toolbar buttons drive common tools (Docker lifecycle, SSH key generation, file copy/download).
- The message list shows agent and user messages.
- The status bar displays connection and action status.

## Notes on Apple Silicon

- The base image is forced to amd64 for compatibility. Docker will use emulation; expect larger images and slower builds.

## Development tips

- The WebSocket endpoint is `/ws`; the frontend connects to `ws://<host>/ws`.
- The backend serves static files from `./frontend/` for convenience, but during development use Vite (`npm run dev`).

## Deployment

- Refer to `deploy/README.md` for details on the Jenkins pipeline, systemd unit, and nginx routing used in production.

## License

Proprietary – All rights reserved. See `LICENSE` for details. Viewing, cloning, and building are permitted for evaluation and contribution only; any other use requires prior written permission.

## Contributing

Contributions are welcome under our contributor assignment terms. By submitting a pull request, you represent you have the right to contribute and you assign (or where assignment is not permitted, you grant an exclusive license for) your contribution to PortNumber53, as described in `LICENSE`. If you require a separate CLA, contact us at `legal@portnumber53.com`.
