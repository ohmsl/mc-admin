# apps/pb

PocketBase custom Go backend for Minecraft admin actions.

## Implemented routes

- `GET /api/mc/status`
- `POST /api/mc/execute`

Both routes require a valid PocketBase record auth token.

## Execute actions

- `whitelist_add`
- `whitelist_remove`
- `kick`
- `say`
- `save_world`
- `restart_server`
- `raw_command` (disabled by default, owner-only when enabled)

## Auth and roles

- Role is read from auth record field `role` by default (`viewer`, `operator`, `owner`).
- On bootstrap, backend attempts to add the `role` select field to `users` collection if present.
- Unknown/missing roles default to `viewer`.

## Persistence bootstrap

On startup, backend ensures:

- `users.role` field (if `users` collection exists)
- `mc_audit_logs` collection for audit entries

## Required env vars

- `MC_RCON_HOST`
- `MC_RCON_PORT`
- `MC_RCON_PASSWORD`
- `MC_RCON_TIMEOUT_SECONDS` (default `5`)
- `MC_RCON_RETRY_COUNT` (default `1`)
- `MC_ALLOW_RAW_COMMAND` (default `false`)
- `MC_AUDIT_COLLECTION` (default `mc_audit_logs`)
- `MC_ROLE_FIELD` (default `role`)

## Local run

```bash
cd apps/pb
go run . serve --http=0.0.0.0:8090
```
