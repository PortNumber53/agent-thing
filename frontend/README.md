# Agent Thing Frontend

React + Vite single-page app that connects to the Go backend over WebSocket (`/ws`) and REST endpoints. The app targets Cloudflare Workers/Pages for hosting.

## Running locally

```bash
npm install
npm run dev
```

By default the UI connects to `ws://localhost:8080/ws`. Override the backend by creating a `.env.local` file with:

```
VITE_BACKEND_HOST=agent.dev.portnumber53.com
```

## Build & deploy

```bash
npm run build     # outputs to dist/
npm run deploy    # builds and runs wrangler deploy
npm run cf-dev    # wrangler dev --config wrangler.jsonc
```

The Cloudflare deployment reads `VITE_BACKEND_HOST` from `wrangler.jsonc` vars (defaults to `agent.dev.portnumber53.com`). Update that value per environment so the SPA talks to the correct backend. The Go API serves the built assets from `/public`.
