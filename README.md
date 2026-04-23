# mc-admin

Lightweight Minecraft admin panel monorepo:

- `apps/pb`: PocketBase custom backend in Go (`/api/mc/status`, `/api/mc/execute`)
- `apps/web`: React + Vite frontend composed with shadcn components
- `infra`: Docker and reverse-proxy deployment scaffolding

## Requirements

- Node.js 22+
- pnpm 10+
- Go 1.23+
- Docker (for containerized deployment)

## Install

```bash
pnpm -w install
```

## Local development

Terminal 1 (backend):

```bash
pnpm dev:pb
```

Terminal 2 (frontend):

```bash
cd apps/web
pnpm dev
```

If backend runs on a different origin than frontend dev server, set:

```bash
VITE_POCKETBASE_URL=http://127.0.0.1:8090
```

## Test and validate

Backend tests:

```bash
cd apps/pb
go test ./...
```

Frontend checks:

```bash
cd apps/web
pnpm typecheck
pnpm lint
pnpm build
```

## Deploy (single image)

```bash
cd infra
docker compose up -d --build
```

Then point reverse proxy hostname (for example `mc-admin.ohmsl.dev`) to `mc-admin:8090` on the Docker network.

If port `8090` is busy on your host, override the published port:

```bash
MC_ADMIN_PORT=18090 docker compose up -d --build
```

## Security notes

- RCON credentials remain server-side only.
- RCON port should stay private/internal.
- Frontend sends only constrained action payloads; backend maps actions to vetted commands.
- Audit records are persisted in PocketBase (`mc_audit_logs`).
