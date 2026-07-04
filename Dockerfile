# Build a static binary with no CGO, then ship a minimal runtime image.
FROM golang:1.22-bookworm AS builder

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/neonsite ./cmd/neonsite

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder --chown=nonroot:nonroot /out/neonsite /neonsite
COPY --chown=nonroot:nonroot content /data/content

ENV HTTP_ADDR=:8080
ENV CONTENT_DIR=/data/content
EXPOSE 8080
VOLUME ["/data/content"]

USER nonroot:nonroot
ENTRYPOINT ["/neonsite"]
