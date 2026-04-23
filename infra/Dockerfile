# syntax=docker/dockerfile:1.7

FROM node:22-alpine AS web-builder
WORKDIR /workspace

RUN corepack enable

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json apps/web/package.json
RUN pnpm install --frozen-lockfile

COPY apps/web apps/web
RUN pnpm --filter web build

FROM golang:1.23-alpine AS pb-builder
WORKDIR /workspace/apps/pb

COPY apps/pb/go.mod apps/pb/go.sum ./
RUN go mod download

COPY apps/pb .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/pb .

FROM alpine:3.20 AS runtime
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 pocketbase

COPY --from=pb-builder /out/pb /app/pb
COPY --from=web-builder /workspace/apps/web/dist /app/pb_public

RUN mkdir -p /app/pb_data && chown -R pocketbase:pocketbase /app

USER pocketbase
EXPOSE 8090

ENTRYPOINT ["/app/pb"]
CMD ["serve", "--http=0.0.0.0:8090"]
