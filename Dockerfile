# Build a static binary with no CGO, then ship a minimal hardened runtime image.
FROM golang:1.22-bookworm AS builder

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o /out/neonsite ./cmd/neonsite

# Distroless static: no shell, no package manager, non-root by default.
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="Neon AI Cloud site" \
      org.opencontainers.image.description="Corporate website and file-backed CMS" \
      org.opencontainers.image.vendor="Neon AI Cloud"

COPY --from=builder --chown=65532:65532 /out/neonsite /neonsite
COPY --chown=65532:65532 content /data/content

ENV HTTP_ADDR=:8080 \
    CONTENT_DIR=/data/content

# Numeric non-root user (distroless nonroot).
USER 65532:65532

EXPOSE 8080
VOLUME ["/data/content"]

# No shell in the image; orchestrators should probe GET /healthz externally.
ENTRYPOINT ["/neonsite"]
