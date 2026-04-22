# ── Stage 1: Build frontend ───────────────────────────────────────────────────
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ── Stage 2: Build backend ────────────────────────────────────────────────────
FROM golang:1.26-alpine AS backend
WORKDIR /app
# Download deps before copying source so this layer is cached.
COPY go.mod go.sum ./
RUN go mod download
# Copy source and embed the built frontend.
COPY . .
COPY --from=frontend /app/ui/dist ./ui/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o helm ./cmd/helm

# ── Stage 3: Minimal runtime image ───────────────────────────────────────────
FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata wget \
    && adduser -D -u 1000 -s /sbin/nologin helm \
    && mkdir -p /data \
    && chown helm:helm /data
WORKDIR /app
COPY --from=backend /app/helm ./
RUN chown helm:helm /app/helm
USER helm
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1
ENTRYPOINT ["./helm"]
CMD ["/config/config.yml"]
