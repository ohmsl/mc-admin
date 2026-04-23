# apps/web

React + Vite frontend for the Minecraft admin panel.

## Commands

```bash
pnpm dev
pnpm typecheck
pnpm lint
pnpm build
```

## Environment

Use `VITE_POCKETBASE_URL` when PocketBase API is not on same origin as the frontend dev server.

Example:

```bash
VITE_POCKETBASE_URL=http://127.0.0.1:8090
```

## Notes

- Uses shadcn components as source (compose-only usage).
- Does not modify shadcn component source files.
- Auth/session is handled via PocketBase JS SDK.
