# Reverse Proxy Example (`mc-admin.ohmsl.dev`)

Goal: expose only PocketBase HTTP service over HTTPS at `mc-admin.ohmsl.dev`.

## Requirements

- Public DNS record: `mc-admin.ohmsl.dev` -> reverse proxy host
- TLS termination at reverse proxy (Let's Encrypt or equivalent)
- Upstream target: `mc-admin:8090` on internal Docker network

## Minimal guidance

- Forward `https://mc-admin.ohmsl.dev` to `http://mc-admin:8090`
- Preserve `Host`, `X-Forwarded-For`, `X-Forwarded-Proto`
- Enforce HTTPS redirect from port 80
- Do not expose Minecraft RCON port publicly

## Optional hardening

- Rate limit POST `/api/mc/execute`
- Allowlist source IPs if practical
- Enable HSTS once TLS setup is stable
