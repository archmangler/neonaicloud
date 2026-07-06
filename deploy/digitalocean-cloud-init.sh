#!/bin/bash
# DigitalOcean Droplet user-data (cloud-init) — first-boot bootstrap.
#
# Paste into: Create Droplet → Advanced Options → User data
# Or: doctl compute droplet create ... --user-data-file deploy/digitalocean-cloud-init.sh
#
# BEFORE USE:
#   1. Set REPO_URL to your git remote (or remove git clone and deploy images another way).
#   2. After first boot, SSH in and create /opt/neonaicloud/.env with secrets
#      (OPENAI_API_KEY, PUBLIC_BASE_URL, etc.) — never bake secrets into user-data.
#   3. Run: sudo neonsite-compose up -d --build
#
# Uptime role: installs Docker + Caddy + a systemd unit that runs
# `docker compose up -d` on every boot so the stack recovers after reboots.

set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

REPO_URL="${REPO_URL:-https://github.com/your-org/neonaicloud.git}"
APP_DIR="/opt/neonaicloud"
DEPLOY_USER="deploy"

log() { echo "[neonsite-cloud-init] $*"; }

log "Updating packages"
apt-get update -qq
apt-get upgrade -y -qq

log "Installing base packages"
apt-get install -y -qq ca-certificates curl git ufw

log "Creating deploy user"
if ! id "$DEPLOY_USER" &>/dev/null; then
  adduser --disabled-password --gecos "" "$DEPLOY_USER"
  usermod -aG sudo "$DEPLOY_USER"
fi

log "Configuring firewall"
ufw default deny incoming
ufw default allow outgoing
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

log "Installing Docker"
curl -fsSL https://get.docker.com | sh
usermod -aG docker "$DEPLOY_USER"

log "Installing Caddy"
apt-get install -y -qq debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' \
  | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' \
  | tee /etc/apt/sources.list.d/caddy-stable.list
apt-get update -qq
apt-get install -y -qq caddy

log "Cloning application (customise REPO_URL before use)"
mkdir -p "$APP_DIR"
if [[ ! -d "$APP_DIR/.git" ]]; then
  git clone "$REPO_URL" "$APP_DIR"
fi
chown -R "$DEPLOY_USER:$DEPLOY_USER" "$APP_DIR"

log "Installing systemd unit for compose on boot"
cat >/etc/systemd/system/neonsite-compose.service <<EOF
[Unit]
Description=Neon AI Cloud Docker Compose stack
After=docker.service network-online.target
Wants=network-online.target
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=$APP_DIR
# Requires /opt/neonaicloud/.env to exist before the stack will be healthy.
ExecStart=/usr/bin/docker compose up -d
ExecStop=/usr/bin/docker compose down
TimeoutStartSec=300
User=$DEPLOY_USER
Group=$DEPLOY_USER

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable neonsite-compose.service
systemctl enable caddy.service
systemctl enable docker.service

log "Bootstrap complete"
log "NEXT: create $APP_DIR/.env, configure /etc/caddy/Caddyfile, then:"
log "  sudo systemctl start neonsite-compose"
