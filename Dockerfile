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
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o helm ./cmd/helm

# ── Stage 3: Minimal runtime image ───────────────────────────────────────────
FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=backend /app/helm ./
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["./helm"]
CMD ["/config/config.yml"]
