# DevOps deployment guide — Neon AI Cloud

Quick reference for deploying the public website, CMS, and digital twin chat service.

## Architecture

```text
                    ┌─────────────────────────────────────┐
  Browser ─────────►│  neonsite (Go)          :8080        │
  /contact chat     │  • static pages + CMS               │
                    │  • /api/twin/* proxy                │
                    └──────────────┬──────────────────────┘
                                   │ TWIN_SERVICE_URL
                                   ▼
                    ┌─────────────────────────────────────┐
                    │  twin API (Python/FastAPI) :7861  │
                    │  • persona PDFs + OpenAI          │
                    │  • OPENAI_API_KEY server-side     │
                    └─────────────────────────────────────┘
```

| Service | Process | Default port | Public? |
| --- | --- | --- | --- |
| **site** | `neonsite` (Go) | 8080 | Yes — only this should face the internet |
| **twin** | `uvicorn` / FastAPI | 7861 | No — internal only; reached via Go proxy |

The browser never receives `OPENAI_API_KEY`. Chat on `/contact` calls same-origin `/api/twin/chat`; Go forwards to the twin service.

---

## Prerequisites

| Tool | Version | Used for |
| --- | --- | --- |
| Go | 1.22+ | Building `neonsite` (bare-metal) |
| Python | 3.11+ | Twin API (bare-metal) |
| Docker | 24+ | Recommended production path |
| Docker Compose | v2 | Multi-service stack |

Secrets you need before enabling chat:

- **OpenAI** (`LLM_PROVIDER=openai`): `OPENAI_API_KEY`
- **Ollama** (`LLM_PROVIDER=ollama`): local Ollama running; no cloud key required

Optional:

- `ADMIN_USER` / `ADMIN_PASSWORD` — CMS at `/admin`
- `PUSHOVER_TOKEN` / `PUSHOVER_USER` — twin lead/unknown-question alerts

---

## Fastest path — Docker Compose

Best for staging and production-style runs.

```bash
# 1. Configure secrets
cp .env.example .env
# Edit .env — set OPENAI_API_KEY and PUBLIC_BASE_URL at minimum

# 2. Build and start both services
docker compose up --build -d

# 3. Verify
curl -fsS http://localhost:8080/healthz
curl -fsS 'http://localhost:8080/api/twin/health?persona=cto'
```

Open [http://localhost:8080/contact](http://localhost:8080/contact) — chat should show **Online** when twin is healthy.

Stop:

```bash
docker compose down
```

Persist CMS uploads/products:

```bash
docker volume inspect neonaicloud_site-content
# Back up the volume mount path or use a bind mount (see Production checklist).
```

---

## Bare-metal / developer deployment

Use control scripts when running directly on a host without Docker.

### 1. Digital twin API

```bash
cp agentic/.env.example agentic/.env
# Set OPENAI_API_KEY in agentic/.env

./scripts/twin start      # FastAPI on 127.0.0.1:7861
./scripts/twin status
./scripts/twin stop
```

First run creates `agentic/.venv` and installs Python dependencies.

### 2. Website

```bash
export TWIN_SERVICE_URL=http://127.0.0.1:7861

# Optional CMS
export ADMIN_USER=admin
export ADMIN_PASSWORD='choose-a-strong-password'

./scripts/restart         # build + start on :8080
./scripts/site status
```

Logs: `bin/neonsite.log`, `bin/twin.log`

### 3. Smoke checks

```bash
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS 'http://127.0.0.1:8080/api/twin/health?persona=cto'
curl -fsS -X POST http://127.0.0.1:8080/api/twin/chat \
  -H 'Content-Type: application/json' \
  -d '{"persona":"cto","message":"Hello","history":[]}'
```

---

## Environment variables

### Site (`neonsite`)

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `HTTP_ADDR` | No | `:8080` | Listen address |
| `CONTENT_DIR` | No | `content` | CMS products + media root |
| `PUBLIC_BASE_URL` | Prod: yes | request host | Canonical, OG, sitemap URLs |
| `ADMIN_USER` | For CMS | — | Admin username |
| `ADMIN_PASSWORD` | For CMS | — | Admin password |
| `ADMIN_SESSION_SECRET` | No | derived | Session signing secret |
| `TWIN_SERVICE_URL` | For chat | — | Upstream twin base URL, e.g. `http://twin:7861` |
| `TWIN_DEFAULT_PERSONA` | No | `cto` | Default persona when none specified |
| `BLOG_SUBSTACK_URL` | No | neonai.substack.com | Blogs page link |
| `BLOG_MEDIUM_URL` | No | medium.com/@neonaicloud | Blogs page link |

### Twin API (FastAPI)

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `LLM_PROVIDER` | No | `openai` | `openai` or `ollama` |
| `OPENAI_API_KEY` | OpenAI only | — | OpenAI credentials (server-side only) |
| `OPENAI_MODEL` | No | `gpt-4o-mini` | OpenAI chat model |
| `OLLAMA_BASE_URL` | Ollama only | `http://127.0.0.1:11434/v1` | Ollama OpenAI-compatible API base |
| `OLLAMA_MODEL` | Ollama only | `llama3.2` | Local model name (`ollama list`) |
| `OLLAMA_API_KEY` | No | `ollama` | Placeholder key for Ollama API client |
| `OLLAMA_SUPPORTS_TOOLS` | No | `false` | Enable function calling if your model supports it |
| `TWIN_HTTP_HOST` | No | `127.0.0.1` | Bind host (`0.0.0.0` in containers) |
| `TWIN_HTTP_PORT` | No | `7861` | Bind port |
| `PUSHOVER_TOKEN` | No | — | Optional notifications |
| `PUSHOVER_USER` | No | — | Optional notifications |

Persona content is baked into the twin image / `agentic/{ceo,cto,engineering,sales}/` on disk.

### LLM provider selection

**OpenAI (default)**

```bash
# agentic/.env or compose .env
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-4o-mini
```

**Local Ollama (bare-metal)**

```bash
# Ensure Ollama is running and the model is pulled:
# ollama pull llama3.2

LLM_PROVIDER=ollama
OLLAMA_BASE_URL=http://127.0.0.1:11434/v1
OLLAMA_MODEL=llama3.2
```

**Local Ollama (Docker Compose reaching host Ollama)**

```bash
LLM_PROVIDER=ollama
OLLAMA_BASE_URL=http://host.docker.internal:11434/v1
OLLAMA_MODEL=llama3.2
```

Compose already maps `host.docker.internal` via `extra_hosts: host-gateway`.

Tool calling (lead capture / unknown-question recording) is enabled for OpenAI by default. For Ollama, set `OLLAMA_SUPPORTS_TOOLS=true` only if your model supports OpenAI-style function calling; otherwise the twin answers without tools and falls back gracefully.

---

## Health checks

| Endpoint | Service | Expected |
| --- | --- | --- |
| `GET /healthz` | site | `200` body `ok` |
| `GET /api/twin/health?persona=cto` | site → twin | `200` JSON `{"status":"ok",...}` |
| `GET /health` | twin (direct) | `200` JSON `{"status":"ok"}` |

Kubernetes / load balancer: probe **site** `/healthz` for liveness. Optionally probe `/api/twin/health?persona=cto` for chat readiness (returns `503` if twin is down or `TWIN_SERVICE_URL` unset).

The site Docker image is **distroless** (no shell) — use external HTTP probes, not `docker exec`.

---

## Docker images

### Site

```bash
docker build -t neonaicloud-site:latest .
```

- Multi-stage: Go 1.22 builder → distroless static runtime
- Non-root UID `65532`
- Embeds seed `content/`; mount a volume at `/data/content` for CMS persistence

Standalone run (twin disabled):

```bash
docker run --rm -p 8080:8080 \
  --read-only \
  --tmpfs /tmp:size=16m,mode=1777 \
  --security-opt no-new-privileges \
  --cap-drop ALL \
  --user 65532:65532 \
  -e PUBLIC_BASE_URL=https://www.example.com \
  -e ADMIN_USER=admin \
  -e ADMIN_PASSWORD='strong-password' \
  -v neonsite-content:/data/content \
  neonaicloud-site:latest
```

### Twin

```bash
docker build -t neonaicloud-twin:latest ./agentic
```

- Python 3.11 slim, non-root UID `65532`
- Production deps: `agentic/requirements-api.txt` (no Gradio)
- Includes persona PDFs and `summary.txt` files

Do **not** publish port `7861` to the public internet. Only the site container needs reachability.

---

## Production checklist

### TLS and routing

- Terminate TLS at your reverse proxy (nginx, Caddy, ALB, Cloudflare).
- Set `PUBLIC_BASE_URL=https://your-domain.com`.
- Forward `X-Forwarded-Proto: https` so HSTS is applied.

### Secrets

- Inject `OPENAI_API_KEY`, `ADMIN_PASSWORD` via secrets manager / K8s secrets — never commit `.env`.
- Rotate keys independently for staging vs production.

### Persistence

- Mount persistent storage at `CONTENT_DIR` (`/data/content` in Docker) for CMS products and uploaded media.
- Persona PDFs ship with the twin image; rebuild twin to update knowledge base.

### Scaling

- **Site**: horizontally scalable (stateless; session cookie is signed, no server-side session store).
- **Twin**: one instance is sufficient for moderate traffic; scale behind an internal load balancer if needed. Each instance loads persona PDFs into memory on first request.

### Chat without twin

If `TWIN_SERVICE_URL` is unset, the site runs normally; contact chat shows **Unavailable** and the enquiry form still works.

### Resource hints (starting point)

| Service | CPU | Memory |
| --- | --- | --- |
| site | 0.25 vCPU | 64–128 MiB |
| twin | 0.5 vCPU | 256–512 MiB (PDF context + OpenAI client) |

---

## Control scripts reference

| Script | Purpose |
| --- | --- |
| `./scripts/site start` | Build and start Go site (background) |
| `./scripts/site stop` | Stop site, free port |
| `./scripts/site restart` | Stop + start |
| `./scripts/site status` | PID, health, port |
| `./scripts/twin start` | Start FastAPI twin (background) |
| `./scripts/twin stop` | Stop twin |
| `./scripts/twin restart` | Stop + start |
| `./scripts/twin status` | PID, health |

Convenience wrappers: `./scripts/start`, `./scripts/stop`, `./scripts/restart` → delegate to `./scripts/site`.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| Chat shows **Unavailable** | Twin not running or `TWIN_SERVICE_URL` unset | Start twin; set URL in site env |
| `503 digital twin service is not configured` | Missing `TWIN_SERVICE_URL` on site | Set to twin internal URL |
| `502 digital twin service unreachable` | Network / wrong URL | Verify twin health on internal network |
| `OPENAI_API_KEY is not set` | OpenAI provider without key | Set key or switch to `LLM_PROVIDER=ollama` |
| Ollama chat errors / timeouts | Ollama not running or wrong model | `ollama list`; verify `OLLAMA_BASE_URL` and `OLLAMA_MODEL` |
| Site serves stale templates | Old binary still running | `./scripts/site restart` (clears port + rebuilds) |
| CMS login disabled | No admin env | Set `ADMIN_USER` and `ADMIN_PASSWORD` |
| Compose site waits on twin | Twin unhealthy | Check `docker compose logs twin` |

Twin logs (Docker): `docker compose logs -f twin`  
Site logs (Docker): `docker compose logs -f site`  
Bare-metal logs: `bin/twin.log`, `bin/neonsite.log`

---

## Security notes

- **API keys**: twin service only; never expose to browser or embed in templates.
- **Twin port**: keep on private network; public ingress only to site `:8080`.
- **Headers**: site sets CSP, `X-Frame-Options`, HSTS (when HTTPS detected).
- **Admin**: `/admin` is `noindex`; use strong passwords and optional `ADMIN_SESSION_SECRET`.
- **Container hardening**: read-only root, dropped capabilities, non-root user (both images).

---

## Related paths

| Path | Role |
| --- | --- |
| `docs/digitalocean-deployment-playbook.md` | Single-Droplet DO deployment (low traffic) |
| `deploy/digitalocean-cloud-init.sh` | DO first-boot user-data template |
| `cmd/neonsite` | Go entrypoint |
| `internal/site` | HTTP handlers, proxy, templates |
| `agentic/api.py` | Twin FastAPI app |
| `agentic/twin.py` | Persona loader + OpenAI chat |
| `docker-compose.yml` | Full stack |
| `Dockerfile` | Site image |
| `agentic/Dockerfile` | Twin image |
| `.env.example` | Compose environment template |
