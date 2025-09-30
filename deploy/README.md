# Deployment Guides

This directory contains infrastructure-as-code assets for deploying the Agent Thing backend.

## Jenkins Pipeline

The top-level `Jenkinsfile` performs the following stages:

1. Checkout repository
2. Run `go test ./...`
3. Build the Go backend binary to `build/bin/agent-thing`
4. Build the frontend with `npm ci && npm run build`
5. Assemble release assets under `build/release` with the frontend files in `public/` and include the root `Dockerfile`
6. Rsync the release to `pinky`, rotate `/var/www/vhosts/agent-thing/<timestamp>` directories, update the `current` symlink, and restart `agent-thing.service`
7. Verify the deployment by hitting `https://agent.dev.portnumber53.com/health`

### Jenkins configuration

- Install Go 1.21+, Node.js 20+, `npm`, `rsync`, and `curl` on the Jenkins agent
- Configure an SSH credential (`pinky-ssh-key` by default) granting password-less sudo access on `pinky`
- Adjust the `SSH_CREDENTIALS_ID` value in the `Jenkinsfile` (or override the job parameters) to match your environment
- Ensure `rsync` and `systemctl` are available on `pinky`
- Store the deployment SSH key in Jenkins credentials (`pinky-ssh-key` by default) so the pipeline can wrap `ssh`/`rsync` with `withCredentials`
- Use the Jenkins job parameters (`REMOTE_HOST`, `REMOTE_BASE_DIR`, `REMOTE_OWNER`) to point the pipeline at the correct host and directory without modifying the Jenkinsfile

## Systemd service

`systemd/agent-thing.service` expects the application to run from `/var/www/vhosts/agent-thing/current`. Override the execution user or working directory as needed before copying the file to `/etc/systemd/system/agent-thing.service`.

For convenience, `systemd/agent-thing.env` sets `PORT=32000` so the backend matches the nginx upstream definition. Place this file at `/etc/agent-thing.env` on `pinky` and adjust as needed.
