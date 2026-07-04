# Neon AI Cloud website

Corporate website for Neon AI Cloud — premium AI application development, platform engineering, AI infrastructure design, implementation and validation, embedded systems, and cloud.

Phase 1 delivers a standalone Go server with the Neon visual system, public shell, and static capability content.

## Requirements

- Go 1.22+

## Run locally

Use the site control scripts so start/stop always clear stale listeners and site processes (including prior `go run` and stray binaries):

```bash
./scripts/start      # build, clear port, start in background
./scripts/stop       # SIGTERM then SIGKILL; verify port is free
./scripts/restart    # stop + start
./scripts/site status
```

Open [http://localhost:8080](http://localhost:8080).

Optional:

```bash
HTTP_ADDR=:3000 ./scripts/restart
START_FOREGROUND=1 ./scripts/start
```

Logs and PID file are written under `bin/` (`bin/neonsite.log`, `bin/neonsite.pid`).

## Build

```bash
go build -trimpath -o bin/neonsite ./cmd/neonsite
./bin/neonsite
```

## Docker

```bash
docker build -t neonaicloud-site .
docker run --rm -p 8080:8080 neonaicloud-site
```

Health check: `GET /healthz`

## Site map

| Path | Page |
| --- | --- |
| `/` | Home |
| `/capabilities` | Capability index |
| `/capabilities/:slug` | Capability detail |
| `/approach` | Delivery approach |
| `/about` | About |
| `/contact` | Contact form |
| `/healthz` | Liveness |

## Layout

| Path | Role |
| --- | --- |
| `cmd/neonsite` | Process entrypoint |
| `internal/site` | HTTP handlers, content model, templates, static assets |
| `bootstrap/` | Visual reference (CapOS aesthetic shipout) |
| `guidelines/` | Design philosophy documents |

Visual tokens and component classes are ported from `bootstrap/styles/globals.css` into `internal/site/static/app.css`.

## Dependencies

Phase 1 uses the Go standard library only (`net/http`, `html/template`, `embed`). No external Go modules.
