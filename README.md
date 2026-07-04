# Neon AI Cloud website

Corporate website for Neon AI Cloud — premium AI application development, platform engineering, AI infrastructure design, implementation and validation, embedded systems, and cloud.

The site is a standalone Go server with a file-backed CMS for product pages.

## Requirements

- Go 1.22+

## Run locally

Use the site control scripts so start/stop always clear stale listeners and site processes:

```bash
./scripts/start      # build, clear port, start in background
./scripts/stop       # SIGTERM then SIGKILL; verify port is free
./scripts/restart    # stop + start
./scripts/site status
```

Open [http://localhost:8080](http://localhost:8080).

Enable the CMS admin UI:

```bash
ADMIN_USER=admin ADMIN_PASSWORD='choose-a-strong-password' ./scripts/restart
```

Then open [http://localhost:8080/admin](http://localhost:8080/admin).

Optional:

```bash
HTTP_ADDR=:3000 ./scripts/restart
CONTENT_DIR=/var/lib/neonsite/content ./scripts/restart
START_FOREGROUND=1 ./scripts/start
```

Logs and PID file are written under `bin/` (`bin/neonsite.log`, `bin/neonsite.pid`).

## CMS

Products and media are stored on disk under `CONTENT_DIR` (default `./content`):

```text
content/
  products/*.md
  media/products/*
```

Each product file uses YAML-like front matter:

```markdown
---
title: Edge Inference Platform
slug: edge-inference-platform
status: published
summary: …
capabilities: embedded-systems, ai-infrastructure
updated: 2026-07-04
---

Body in a small Markdown subset (#, ##, lists, **bold**, [links](/url)).
```

Admin capabilities:

- Login (`ADMIN_USER` / `ADMIN_PASSWORD`, signed session cookie)
- Create / edit / delete products
- Publish flag (`draft` | `published`)
- Media upload and delete
- CSRF protection on admin POST routes

Only `published` products appear on `/products`, home, and capability pages.

## Docker

```bash
docker build -t neonaicloud-site .
docker run --rm -p 8080:8080 \
  -e ADMIN_USER=admin \
  -e ADMIN_PASSWORD='choose-a-strong-password' \
  -v neonsite-content:/data/content \
  neonaicloud-site
```

`CONTENT_DIR` defaults to `/data/content` in the image. Mount a volume to persist CMS edits.

Health check: `GET /healthz`

## Site map

| Path | Page |
| --- | --- |
| `/` | Home |
| `/capabilities` | Capability index |
| `/capabilities/:slug` | Capability detail |
| `/products` | Published products |
| `/products/:slug` | Product detail |
| `/approach` | Delivery approach |
| `/about` | About |
| `/contact` | Contact + digital twin chat |
| `/admin` | CMS (auth required) |
| `/media/...` | Uploaded media |
| `/healthz` | Liveness |

## Layout

| Path | Role |
| --- | --- |
| `cmd/neonsite` | Process entrypoint |
| `internal/site` | HTTP handlers, CMS, templates, static assets |
| `content/` | Product pages and media (CMS data) |
| `scripts/` | Start / stop / restart control |
| `bootstrap/` | Visual reference (CapOS aesthetic shipout) |
| `guidelines/` | Design philosophy documents |

## Dependencies

Go standard library only (`net/http`, `html/template`, `embed`, `crypto/...`). No external Go modules.
