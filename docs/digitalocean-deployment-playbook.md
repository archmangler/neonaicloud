# DigitalOcean deployment playbook — Neon AI Cloud

Step-by-step guide to host the public website (Go site + CMS + digital twin chat) on a single DigitalOcean Droplet under **low traffic**.

For architecture and environment variables, see [devops-guide.md](devops-guide.md).

---

## Recommended Droplet (low traffic)

| Setting | Recommendation |
| --- | --- |
| **Plan** | Basic (shared vCPU) |
| **Size** | **1 vCPU / 2 GB RAM / 50 GB SSD** |
| **OS** | **Ubuntu 24.04 LTS** |
| **Region** | Closest to your audience (e.g. `sgp1`, `lon1`, `nyc3`) |
| **Authentication** | SSH key (required) |
| **Backups** | Optional — enable weekly Droplet backups (~20% extra) |

**Why 2 GB RAM?**

| Component | Typical RAM |
| --- | --- |
| Ubuntu + Docker daemon | ~400–550 MiB |
| `site` (Go, distroless) | ~64–128 MiB |
| `twin` (Python + persona PDFs in memory) | ~256–512 MiB |
| Caddy reverse proxy | ~30–50 MiB |
| Headroom for deploy/build spikes | ~300+ MiB |

**1 vCPU / 1 GB** can work if you use **OpenAI** (no local Ollama) and accept occasional memory pressure during `docker compose build`. For a stable production minimum, **use 2 GB**.

**Do not** run Ollama on the same 2 GB Droplet for production — use `LLM_PROVIDER=openai` or a separate GPU/larger instance for local models.

Approximate cost (2026): **~$12 USD/month** for the 2 GB Droplet + domain (~$12/year if registered via DO).

---

## Architecture on one Droplet

```text
Internet :443/:80
        │
        ▼
   Caddy (TLS, auto Let's Encrypt)
        │
        ▼
   site :8080  ──internal──►  twin :7861
        │
   Docker volume: site-content (CMS)
```

Only **80** and **443** are public. Port **7861** (twin) stays on the Docker network.

---

## Prerequisites

Before you start, have ready:

- [ ] Domain name (e.g. `www.neoncloud.ai`)
- [ ] DNS managed in DigitalOcean (or elsewhere — A record to Droplet IP)
- [ ] `OPENAI_API_KEY` (if using OpenAI for digital twins)
- [ ] Git deploy access to this repository (or a container registry)
- [ ] Local SSH key added to your DO account

---

## Phase 1 — Create the Droplet

1. DigitalOcean → **Create** → **Droplets**.
2. Choose **Ubuntu 24.04 LTS**.
3. Select **Basic → Regular → 1 vCPU / 2 GB / 50 GB SSD**.
4. Add your SSH key; disable password login if prompted.
5. Hostname: e.g. `neonsite-prod`.
6. Create Droplet and note the **public IPv4 address**.

### DNS

In DigitalOcean → **Networking** → **Domains** (or your DNS provider):

| Type | Host | Value |
| --- | --- | --- |
| A | `@` | `<droplet-ipv4>` |
| A | `www` | `<droplet-ipv4>` |

Wait for DNS to propagate (often 5–30 minutes).

---

## Phase 2 — Secure and prepare the server

SSH in as root:

```bash
ssh root@<droplet-ipv4>
```

### Create deploy user

```bash
adduser deploy
usermod -aG sudo deploy
rsync --archive --chown=deploy:deploy ~/.ssh /home/deploy
```

Reconnect as `deploy`:

```bash
ssh deploy@<droplet-ipv4>
```

### Firewall

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
sudo ufw status
```

### Install Docker

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl git
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker deploy
```

Log out and back in so `docker` group applies:

```bash
exit
ssh deploy@<droplet-ipv4>
docker compose version
```

---

## Phase 3 — Deploy the application

### Clone the repository

```bash
sudo mkdir -p /opt/neonaicloud
sudo chown deploy:deploy /opt/neonaicloud
git clone https://github.com/<your-org>/neonaicloud.git /opt/neonaicloud
cd /opt/neonaicloud
```

(Replace with your fork or private repo URL.)

### Configure environment

```bash
cp .env.example .env
chmod 600 .env
nano .env
```

**Minimum production values:**

```bash
PUBLIC_BASE_URL=https://www.yourdomain.com

# Digital twin (OpenAI — recommended on a 2 GB Droplet)
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-4o-mini

# Optional CMS
ADMIN_USER=admin
ADMIN_PASSWORD=<strong-random-password>
ADMIN_SESSION_SECRET=<long-random-string>
```

`TWIN_SERVICE_URL` is already set inside `docker-compose.yml` (`http://twin:7861`) — do not expose twin publicly.

### Build and start

```bash
docker compose up --build -d
docker compose ps
```

Verify locally on the server:

```bash
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS 'http://127.0.0.1:8080/api/twin/health?persona=cto'
```

---

## Phase 4 — TLS with Caddy

Install Caddy on the host (simplest on a single Droplet):

```bash
sudo apt-get install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt-get update
sudo apt-get install -y caddy
```

Create `/etc/caddy/Caddyfile`:

```caddy
www.yourdomain.com {
    reverse_proxy 127.0.0.1:8080
}

yourdomain.com {
    redir https://www.yourdomain.com{uri} permanent
}
```

Replace `yourdomain.com` with your real domain. Ensure DNS A records point to this Droplet **before** reloading Caddy.

```bash
sudo systemctl reload caddy
sudo systemctl enable caddy
```

Caddy obtains and renews Let's Encrypt certificates automatically.

### Confirm HTTPS

```bash
curl -fsSI https://www.yourdomain.com/healthz
```

Open `https://www.yourdomain.com/contact` — digital twin chips should show **Online**.

---

## Phase 5 — Persistence and updates

### CMS content volume

Compose creates a Docker volume `neonaicloud_site-content`. Inspect:

```bash
docker volume inspect neonaicloud_site-content
```

For easier backups, switch to a bind mount in `docker-compose.override.yml`:

```yaml
services:
  site:
    volumes:
      - /opt/neonaicloud/data/content:/data/content
```

Then:

```bash
mkdir -p /opt/neonaicloud/data/content
docker compose up -d
```

### Deploy updates

```bash
cd /opt/neonaicloud
git pull
docker compose up --build -d
docker compose ps
curl -fsS https://www.yourdomain.com/healthz
```

### Optional: enable Droplet backups

DigitalOcean → Droplet → **Backups** → Enable. Complements (does not replace) exporting the CMS volume periodically.

---

## Phase 6 — Smoke test checklist

Run after every deploy:

| Check | Command / action |
| --- | --- |
| Site health | `curl -fsS https://www.yourdomain.com/healthz` |
| Twin proxy | `curl -fsS 'https://www.yourdomain.com/api/twin/health?persona=cto'` |
| Homepage | Browser → `/` |
| Contact + chat | Browser → `/contact`, send a test message |
| CMS (if enabled) | Browser → `/admin` login |
| Twin not public | `curl --max-time 3 http://<droplet-ip>:7861/health` should **fail** (port closed) |

---

## Monitoring (optional, low effort)

- Enable **DigitalOcean Monitoring** on the Droplet (free metrics: CPU, RAM, disk).
- Set alerts: RAM > 85% sustained, disk > 80%.
- Watch twin logs during first real chat traffic:

  ```bash
  docker compose logs -f twin
  ```

---

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| Caddy certificate error | DNS not pointing to Droplet yet; wait and `sudo systemctl reload caddy` |
| Chat **Standby** | `docker compose logs twin`; verify `OPENAI_API_KEY` in `.env` |
| Out of memory during build | Add swap temporarily or build images on a larger machine / CI and push to DO Container Registry |
| Site 502 from Caddy | `docker compose ps` — site container down; check `docker compose logs site` |
| Stale content after deploy | `docker compose up --build -d` (rebuild, not just restart) |

---

## Droplet startup script (cloud-init) and uptime

DigitalOcean lets you paste a **user-data script** at Droplet creation. It runs **once on first boot** via cloud-init — it is not re-run on ordinary reboots or deploys.

### What it improves

| Benefit | How |
| --- | --- |
| **Faster, repeatable provisioning** | Docker, Caddy, firewall, and app directory installed automatically |
| **Recovery after reboot** | Combined with a **systemd unit** (`neonsite-compose.service`) that runs `docker compose up -d` at boot |
| **Disaster rebuild** | Recreate a Droplet from the same script + restore CMS volume backup |

Your `docker-compose.yml` already sets `restart: unless-stopped` on both services, so Docker will also restart crashed containers without a startup script. The systemd unit adds a belt-and-braces **full stack reconcile** after the host comes back from maintenance or power events.

### What it does *not* improve

- **Zero-downtime deploys** — updating still means `docker compose up --build -d` (brief blip)
- **High availability** — one Droplet is still a single point of failure
- **Application bugs or bad deploys** — startup scripts cannot roll back a broken image
- **Secrets on first boot** — do not put `OPENAI_API_KEY` in user-data; create `.env` over SSH after boot

For higher uptime beyond a single VM, you need load balancing, health-checked replicas, or managed PaaS — not a longer startup script.

### Recommended pattern

1. **First boot (user-data):** install Docker, Caddy, UFW, clone repo, enable systemd unit.  
   Template: [`deploy/digitalocean-cloud-init.sh`](../deploy/digitalocean-cloud-init.sh)

2. **One-time manual step:** SSH in, create `/opt/neonaicloud/.env`, configure `/etc/caddy/Caddyfile`, then:

   ```bash
   sudo systemctl start neonsite-compose
   ```

3. **Every reboot:** `docker.service` → `neonsite-compose.service` → `caddy.service` start in order.

4. **Optional:** DigitalOcean Monitoring alerts on CPU/RAM and external uptime check on `https://www.yourdomain.com/healthz`.

### Using the template on DigitalOcean

When creating the Droplet:

1. **Advanced Options** → **Add Initialization scripts** (user data)
2. Paste the contents of `deploy/digitalocean-cloud-init.sh`
3. Edit `REPO_URL` at the top of the script to your repository
4. Complete Droplet creation; wait ~3–5 minutes, then SSH in and finish `.env` + Caddy as above

Or with `doctl`:

```bash
doctl compute droplet create neonsite-prod \
  --image ubuntu-24-04-x64 \
  --size s-1vcpu-2gb \
  --region sgp1 \
  --ssh-keys <fingerprint> \
  --user-data-file deploy/digitalocean-cloud-init.sh
```

---

## Scaling path (when low traffic grows)

Stay on one Droplet until RAM or CPU is consistently high. Then consider, in order:

1. Resize Droplet to **2 vCPU / 4 GB**.
2. Move twin to a second private Droplet; set `TWIN_SERVICE_URL` to internal IP/VPC URL.
3. Add DO Load Balancer + multiple site Droplets (site is stateless).

---

## Quick reference

| Item | Value |
| --- | --- |
| Recommended Droplet | **1 vCPU, 2 GB RAM, 50 GB SSD, Ubuntu 24.04** |
| Public ports | 22 (SSH), 80, 443 only |
| App stack | `docker compose up -d` |
| TLS | Caddy → `127.0.0.1:8080` |
| Health | `GET /healthz` |
| Docs | [devops-guide.md](devops-guide.md) |
