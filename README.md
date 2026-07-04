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
PUBLIC_BASE_URL=https://www.example.com ./scripts/restart
START_FOREGROUND=1 ./scripts/start
```

`PUBLIC_BASE_URL` sets absolute canonical, Open Graph, sitemap, and robots URLs. When unset, the request host is used.

Logs and PID file are written under `bin/` (`bin/neonsite.log`, `bin/neonsite.pid`).

## CMS

Products and media are stored on disk under `CONTENT_DIR` (default `./content`):

```text
content/
  products/*.md
  media/products/*
```

Admin capabilities:

- Login (`ADMIN_USER` / `ADMIN_PASSWORD`, signed session cookie)
- Create / edit / delete products
- Publish flag (`draft` | `published`)
- Media upload and delete
- CSRF protection on admin POST routes

Only `published` products appear on `/products`, home, and capability pages.

## SEO

- Per-page title, description, canonical URL, robots
- Open Graph and Twitter card tags
- `/sitemap.xml` — static pages, capabilities, published products
- `/robots.txt` — allows public routes, disallows `/admin`, points at the sitemap
- Admin pages are `noindex, nofollow`

## Docker

Hardened defaults: distroless static image, numeric non-root user (`65532`), no shell, dropped capabilities, read-only root filesystem (compose), `no-new-privileges`.

```bash
docker build -t neonaicloud-site .
docker run --rm -p 8080:8080 \
  --read-only \
  --tmpfs /tmp:size=16m,mode=1777 \
  --security-opt no-new-privileges \
  --cap-drop ALL \
  --user 65532:65532 \
  -e PUBLIC_BASE_URL=https://www.example.com \
  -e ADMIN_USER=admin \
  -e ADMIN_PASSWORD='choose-a-strong-password' \
  -v neonsite-content:/data/content \
  neonaicloud-site
```

Or:

```bash
PUBLIC_BASE_URL=http://localhost:8080 \
ADMIN_USER=admin ADMIN_PASSWORD='choose-a-strong-password' \
docker compose up --build
```

Probe liveness externally with `GET /healthz` (the image has no shell for in-container healthchecks).

Security response headers are applied on every response (`Content-Security-Policy`, `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy`, and `Strict-Transport-Security` when HTTPS is detected).

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
| `/sitemap.xml` | Sitemap |
| `/robots.txt` | Robots |
| `/healthz` | Liveness |

## Layout

| Path | Role |
| --- | --- |
| `cmd/neonsite` | Process entrypoint |
| `internal/site` | HTTP handlers, CMS, templates, static assets |
| `content/` | Product pages and media (CMS data) |
| `scripts/` | Start / stop / restart control |
| `docker-compose.yml` | Hardened local/prod-style run |
| `bootstrap/` | Visual reference (CapOS aesthetic shipout) |
| `guidelines/` | Design philosophy documents |

## Dependencies

Go standard library only (`net/http`, `html/template`, `embed`, `crypto/...`). No external Go modules.
