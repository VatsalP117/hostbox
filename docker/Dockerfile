# --- Stage 1: Build frontend ---
FROM node:20-alpine AS frontend

WORKDIR /build/web
COPY web/package*.json ./
RUN npm ci --prefer-offline
COPY web/ ./
RUN npm run build

# --- Stage 2: Build Go binary ---
FROM golang:1.25-alpine AS backend

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /build/web/dist ./web/dist

ARG VERSION=dev
ARG GIT_COMMIT=unknown

RUN CGO_ENABLED=1 go build \
    -ldflags "-s -w \
        -X github.com/vatsalpatel/hostbox/internal/version.Version=${VERSION} \
        -X github.com/vatsalpatel/hostbox/internal/version.GitCommit=${GIT_COMMIT} \
        -X github.com/vatsalpatel/hostbox/internal/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /hostbox ./cmd/api

RUN CGO_ENABLED=1 go build \
    -ldflags "-s -w \
        -X github.com/vatsalpatel/hostbox/internal/version.Version=${VERSION}" \
    -o /hostbox-cli ./cmd/cli

# --- Stage 3: Production image ---
FROM alpine:3.20

RUN apk add --no-cache \
    ca-certificates \
    sqlite-libs \
    curl \
    git \
    tzdata \
    && addgroup -g 1000 hostbox \
    && adduser -u 1000 -G hostbox -s /bin/sh -D hostbox

COPY --from=backend /hostbox /usr/local/bin/hostbox
COPY --from=backend /hostbox-cli /usr/local/bin/hostbox-cli

# Create data directories
RUN mkdir -p /app/data/backups /app/deployments /app/logs /cache \
    && chown -R hostbox:hostbox /app /cache

WORKDIR /app

USER hostbox

ENV HOST=0.0.0.0 \
    PORT=8080 \
    DATABASE_PATH=/app/data/hostbox.db \
    DEPLOYMENTS_DIR=/app/deployments \
    LOGS_DIR=/app/logs \
    CACHE_DIR=/cache \
    BACKUP_DIR=/app/data/backups \
    LOG_LEVEL=info \
    LOG_FORMAT=json

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/api/v1/health"]

ENTRYPOINT ["hostbox"]
