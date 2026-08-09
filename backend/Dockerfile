# ---- build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Copy sources and build a static binary (no cgo).
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/docuflow-server ./cmd/server

# ---- runtime stage ----
FROM alpine:3.20

# Non-root runtime user and writable storage volume.
RUN addgroup -S app && adduser -S -G app -u 10001 app \
    && mkdir -p /data/storage && chown -R app:app /data

WORKDIR /data
USER app

COPY --from=builder /out/docuflow-server /usr/local/bin/docuflow-server

EXPOSE 8080
VOLUME ["/data/storage"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O - --post-data='{}' \
    --header='Content-Type: application/json' \
    http://127.0.0.1:8080/api/v1/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/docuflow-server"]
