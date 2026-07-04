# Build a static binary with no CGO, then ship a minimal runtime image.
FROM golang:1.22-bookworm AS builder

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/neonsite ./cmd/neonsite

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/neonsite /neonsite

ENV HTTP_ADDR=:8080
EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/neonsite"]
