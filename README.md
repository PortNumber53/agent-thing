# agent-thing

Frontend (React/Vite) lives under `frontend/`.

Backend (Go) lives under `backend/` and exposes:
- WebSocket at `/ws` streaming the current system time (RFC3339Nano) once per second.
- Health check at `/health`.
- Docker management API under `/docker/*` (start/stop/rebuild/status).
- Early support for Google OAuth (`/auth/google/*`) and Stripe subscriptions (`/billing/*`).

## Run backend locally

```bash
cd backend
go run .
```

Default port: `18511` (override via `PORT` or `AGENT_THING_LISTEN_ADDR`).

Backend will automatically load variables from `.env` (repo root) and/or `backend/.env` if those files exist.

You can also run:

```bash
./backend/dev.sh
```

## Production config + systemd

- **INI config file**: backend reads an optional INI config at:
  - `/etc/agent-thing/config.ini` (default), override with `CONFIG_INI_PATH`.
  - Environment variables / `.env` override INI values.
- **Sample**: see `deploy/config.ini.sample`.
- **systemd unit**: see `deploy/systemd/agent-thing.service`.
- **Bootstrap**: `deploy/scripts/install_backend.sh` will:
  - create `/opt/agent-thing/bin` and `/etc/agent-thing`,
  - copy the sample to `/etc/agent-thing/config.ini` if missing,
  - install/enable/restart `agent-thing.service`.

## Run frontend locally

```bash
cd frontend
npm install
npm run dev
```

Frontend dev server listens on port `18510`.

## Hot reload (backend)

Install Air (once):

```bash
go install github.com/air-verse/air@latest
```

Run:

```bash
air -c backend/.air.toml
```

## Database & migrations (early support)

We use [`golang-migrate/migrate`](https://github.com/golang-migrate/migrate) for SQL migrations.

Set one of:
- `DATABASE_URL` (production Postgres)
- `XATA_DATABASE_URL` (Xata Postgres endpoint)

Run migrations:

```bash
go run ./backend migrate up
go run ./backend migrate down
go run ./backend migrate status
go run ./backend migrate create <name>
```

Migrations live in `db/migrations/`.

## Required env vars for OAuth & Stripe (early support)

- **Google OAuth**: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL` (optional), `JWT_SECRET`
  - For local dev, Google must be configured with an authorized redirect URI matching the backend callback, e.g. `http://localhost:18511/callback/oauth/google`. If `GOOGLE_REDIRECT_URL` is empty, the backend defaults to `${BACKEND_BASE_URL}/callback/oauth/google`.
- **Stripe**: `STRIPE_SECRET_KEY`, `STRIPE_PUBLISHABLE_KEY`, `STRIPE_WEBHOOK_SECRET`, `STRIPE_PRICE_ID` (default subscription price)
  - Webhook endpoint (register in Stripe dashboard): `${BACKEND_BASE_URL}/webhook/stripe`  
    Matches production path like `https://taskninja.work/webhook/stripe` ([reference](https://taskninja.work/webhook/stripe)).
- **Xata (optional)**: `XATA_DATABASE_URL`, `XATA_API_KEY`
