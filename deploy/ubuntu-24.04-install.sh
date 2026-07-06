#!/usr/bin/env bash
# Neon AI Cloud — Ubuntu 24.04 install, build, and deploy
#
# Run on a fresh or prepared Droplet after SSH login:
#   curl -fsSL https://raw.githubusercontent.com/<org>/neonaicloud/main/deploy/ubuntu-24.04-install.sh -o install.sh
#   chmod +x install.sh
#   sudo ./install.sh
#
# Or from a cloned repo:
#   sudo ./deploy/ubuntu-24.04-install.sh
#
# Non-interactive (set all required vars):
#   sudo REPO_URL=https://github.com/you/neonaicloud.git \
#        DOMAIN=neoncloud.ai \
#        PUBLIC_BASE_URL=https://www.neoncloud.ai \
#        OPENAI_API_KEY=sk-... \
#        ADMIN_PASSWORD='...' \
#        ./deploy/ubuntu-24.04-install.sh
#
# Optional env:
#   APP_DIR=/opt/neonaicloud
#   DEPLOY_USER=deploy
#   SKIP_FIREWALL=1
#   SKIP_CADDY=1          # site only on :8080 (no TLS)
#   SKIP_SYSTEMD=1
#   SKIP_APT_UPGRADE=1
#   GIT_BRANCH=main
#   LLM_PROVIDER=openai
#   OPENAI_MODEL=gpt-4o-mini
#   ADMIN_USER=admin
#   ADMIN_SESSION_SECRET=...

set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

# --- Config (override via environment) ---
APP_DIR="${APP_DIR:-/opt/neonaicloud}"
DEPLOY_USER="${DEPLOY_USER:-deploy}"
REPO_URL="${REPO_URL:-}"
DOMAIN="${DOMAIN:-}"
PUBLIC_BASE_URL="${PUBLIC_BASE_URL:-}"
OPENAI_API_KEY="${OPENAI_API_KEY:-}"
LLM_PROVIDER="${LLM_PROVIDER:-openai}"
OPENAI_MODEL="${OPENAI_MODEL:-gpt-4o-mini}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
ADMIN_SESSION_SECRET="${ADMIN_SESSION_SECRET:-}"
GIT_BRANCH="${GIT_BRANCH:-main}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

log() { printf '[neonsite-install] %s\n' "$*"; }
die() { log "ERROR: $*"; exit 1; }
need_cmd() { command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"; }

require_root() {
  if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
    die "run as root: sudo $0"
  fi
}

detect_ubuntu() {
  [[ -r /etc/os-release ]] || die "/etc/os-release not found"
  # shellcheck source=/dev/null
  source /etc/os-release
  [[ "${ID:-}" == "ubuntu" ]] || die "this script targets Ubuntu (found: ${ID:-unknown})"
  [[ "${VERSION_ID:-}" == "24.04" ]] || log "warning: tested on Ubuntu 24.04 LTS (found ${PRETTY_NAME:-unknown})"
}

ensure_deploy_user() {
  if ! id "$DEPLOY_USER" &>/dev/null; then
    log "creating user $DEPLOY_USER"
    adduser --disabled-password --gecos "" "$DEPLOY_USER"
    usermod -aG sudo "$DEPLOY_USER"
  fi
  if [[ -d /root/.ssh ]] && [[ ! -d "/home/$DEPLOY_USER/.ssh" ]]; then
    log "copying root SSH keys to $DEPLOY_USER"
    install -d -m 700 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "/home/$DEPLOY_USER/.ssh"
    if [[ -f /root/.ssh/authorized_keys ]]; then
      install -m 600 -o "$DEPLOY_USER" -g "$DEPLOY_USER" \
        /root/.ssh/authorized_keys "/home/$DEPLOY_USER/.ssh/authorized_keys"
    fi
  fi
}

apt_packages() {
  log "installing base packages"
  apt-get update -qq
  if [[ "${SKIP_APT_UPGRADE:-0}" != "1" ]]; then
    apt-get upgrade -y -qq
  fi
  apt-get install -y -qq ca-certificates curl git ufw
}

install_docker() {
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    log "docker already installed: $(docker --version), $(docker compose version)"
    return
  fi
  log "installing Docker"
  curl -fsSL https://get.docker.com | sh
  systemctl enable docker.service
  systemctl start docker.service
}

install_caddy() {
  if [[ "${SKIP_CADDY:-0}" == "1" ]]; then
    log "skipping Caddy (SKIP_CADDY=1)"
    return
  fi
  if command -v caddy >/dev/null 2>&1; then
    log "caddy already installed: $(caddy version 2>/dev/null | head -1 || true)"
    return
  fi
  log "installing Caddy"
  apt-get install -y -qq debian-keyring debian-archive-keyring apt-transport-https
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' \
    | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' \
    | tee /etc/apt/sources.list.d/caddy-stable.list >/dev/null
  apt-get update -qq
  apt-get install -y -qq caddy
  systemctl enable caddy.service
}

configure_firewall() {
  if [[ "${SKIP_FIREWALL:-0}" == "1" ]]; then
    log "skipping firewall (SKIP_FIREWALL=1)"
    return
  fi
  log "configuring UFW"
  ufw default deny incoming
  ufw default allow outgoing
  ufw allow OpenSSH
  ufw allow 80/tcp
  ufw allow 443/tcp
  ufw --force enable
  ufw status verbose || true
}

prompt_if_empty() {
  local var_name="$1"
  local prompt_text="$2"
  local secret="${3:-0}"
  local current="${!var_name:-}"
  if [[ -n "$current" ]]; then
    return
  fi
  if [[ ! -t 0 ]]; then
    die "set $var_name (non-interactive shell)"
  fi
  if [[ "$secret" == "1" ]]; then
    read -r -s -p "$prompt_text: " current
    echo
  else
    read -r -p "$prompt_text: " current
  fi
  printf -v "$var_name" '%s' "$current"
}

resolve_repo_url() {
  if [[ -n "$REPO_URL" ]]; then
    return
  fi
  if [[ -d "$REPO_ROOT/.git" ]]; then
    REPO_URL="$(git -C "$REPO_ROOT" remote get-url origin 2>/dev/null || true)"
    if [[ -n "$REPO_URL" ]]; then
      log "using git remote from local checkout: $REPO_URL"
      return
    fi
  fi
  prompt_if_empty REPO_URL "Git repository URL (https://github.com/you/neonaicloud.git)"
}

resolve_domain() {
  if [[ -n "$DOMAIN" ]]; then
    return
  fi
  if [[ -n "$PUBLIC_BASE_URL" ]]; then
    DOMAIN="${PUBLIC_BASE_URL#https://}"
    DOMAIN="${DOMAIN#http://}"
    DOMAIN="${DOMAIN%%/*}"
    DOMAIN="${DOMAIN#www.}"
    log "derived DOMAIN=$DOMAIN from PUBLIC_BASE_URL"
    return
  fi
  if [[ "${SKIP_CADDY:-0}" == "1" ]]; then
    return
  fi
  prompt_if_empty DOMAIN "Primary domain (apex, e.g. neoncloud.ai)"
}

resolve_public_base_url() {
  if [[ -n "$PUBLIC_BASE_URL" ]]; then
    return
  fi
  if [[ -n "$DOMAIN" ]]; then
    PUBLIC_BASE_URL="https://www.${DOMAIN}"
    log "using PUBLIC_BASE_URL=$PUBLIC_BASE_URL"
    return
  fi
  prompt_if_empty PUBLIC_BASE_URL "Public site URL (e.g. https://www.neoncloud.ai)"
}

prompt_secrets() {
  if [[ "$LLM_PROVIDER" == "openai" ]] && [[ -z "$OPENAI_API_KEY" ]]; then
    prompt_if_empty OPENAI_API_KEY "OpenAI API key (sk-...)" 1
  fi
  if [[ -z "$ADMIN_PASSWORD" ]]; then
    prompt_if_empty ADMIN_PASSWORD "CMS admin password (leave blank to disable /admin)" 1
  fi
}

sync_application() {
  log "syncing application to $APP_DIR"
  install -d -m 755 "$APP_DIR"

  if [[ -d "$APP_DIR/.git" ]]; then
    log "updating existing clone"
    sudo -u "$DEPLOY_USER" git -C "$APP_DIR" fetch --all --prune
    sudo -u "$DEPLOY_USER" git -C "$APP_DIR" checkout "$GIT_BRANCH"
    sudo -u "$DEPLOY_USER" git -C "$APP_DIR" pull --ff-only origin "$GIT_BRANCH"
  elif [[ -d "$REPO_ROOT/.git" ]] && [[ "$REPO_ROOT" != "$APP_DIR" ]]; then
    log "copying local checkout to $APP_DIR"
    rsync -a --delete \
      --exclude '.git' \
      --exclude 'bin/' \
      --exclude 'agentic/.venv' \
      --exclude '.env' \
      "$REPO_ROOT/" "$APP_DIR/"
    chown -R "$DEPLOY_USER:$DEPLOY_USER" "$APP_DIR"
  else
    log "cloning $REPO_URL (branch $GIT_BRANCH)"
    if [[ -d "$APP_DIR" ]] && [[ -n "$(ls -A "$APP_DIR" 2>/dev/null || true)" ]]; then
      die "$APP_DIR exists and is not a git repo — move it aside or set APP_DIR"
    fi
    sudo -u "$DEPLOY_USER" git clone --branch "$GIT_BRANCH" --depth 1 "$REPO_URL" "$APP_DIR"
  fi

  chown -R "$DEPLOY_USER:$DEPLOY_USER" "$APP_DIR"
}

write_env_file() {
  local env_file="$APP_DIR/.env"
  if [[ -f "$env_file" ]] && [[ "${FORCE_ENV:-0}" != "1" ]]; then
    log "keeping existing $env_file (set FORCE_ENV=1 to overwrite)"
    return
  fi

  log "writing $env_file"
  if [[ -z "$ADMIN_SESSION_SECRET" ]] && [[ -n "$ADMIN_PASSWORD" ]]; then
    ADMIN_SESSION_SECRET="$(openssl rand -hex 32)"
  fi

  cat >"$env_file" <<EOF
# Generated by deploy/ubuntu-24.04-install.sh on $(date -u +%Y-%m-%dT%H:%MZ)
PUBLIC_BASE_URL=${PUBLIC_BASE_URL}
LLM_PROVIDER=${LLM_PROVIDER}
OPENAI_API_KEY=${OPENAI_API_KEY}
OPENAI_MODEL=${OPENAI_MODEL}
EOF

  if [[ -n "$ADMIN_PASSWORD" ]]; then
    cat >>"$env_file" <<EOF
ADMIN_USER=${ADMIN_USER}
ADMIN_PASSWORD=${ADMIN_PASSWORD}
ADMIN_SESSION_SECRET=${ADMIN_SESSION_SECRET}
EOF
  fi

  chown "$DEPLOY_USER:$DEPLOY_USER" "$env_file"
  chmod 600 "$env_file"
}

write_caddyfile() {
  if [[ "${SKIP_CADDY:-0}" == "1" ]]; then
    return
  fi
  [[ -n "$DOMAIN" ]] || die "DOMAIN required when Caddy is enabled"

  local caddyfile="/etc/caddy/Caddyfile"
  log "writing $caddyfile"
  cat >"$caddyfile" <<EOF
www.${DOMAIN} {
    reverse_proxy 127.0.0.1:8080
}

${DOMAIN} {
    redir https://www.${DOMAIN}{uri} permanent
}
EOF

  if caddy validate --config "$caddyfile" >/dev/null 2>&1; then
    log "Caddyfile validated"
  else
    log "warning: caddy validate failed — check DNS before reload"
  fi
}

install_systemd_unit() {
  if [[ "${SKIP_SYSTEMD:-0}" == "1" ]]; then
    log "skipping systemd unit (SKIP_SYSTEMD=1)"
    return
  fi

  log "installing neonsite-compose.service"
  cat >/etc/systemd/system/neonsite-compose.service <<EOF
[Unit]
Description=Neon AI Cloud Docker Compose stack
After=docker.service network-online.target
Wants=network-online.target
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=${APP_DIR}
ExecStart=/usr/bin/docker compose up -d
ExecStop=/usr/bin/docker compose down
TimeoutStartSec=600
User=${DEPLOY_USER}
Group=${DEPLOY_USER}

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable neonsite-compose.service
}

add_deploy_to_docker_group() {
  usermod -aG docker "$DEPLOY_USER" || true
}

build_and_deploy() {
  log "building and starting containers (this may take several minutes on 1 vCPU)"
  # shellcheck disable=SC1091
  sudo -u "$DEPLOY_USER" bash -lc "
    set -euo pipefail
    cd '$APP_DIR'
    docker compose build
    docker compose up -d
    docker compose ps
  "
}

reload_caddy() {
  if [[ "${SKIP_CADDY:-0}" == "1" ]]; then
    return
  fi
  log "reloading Caddy"
  systemctl reload caddy || systemctl restart caddy
}

wait_for_health() {
  local url="$1"
  local label="$2"
  local attempts="${3:-30}"
  local i

  log "waiting for $label ($url)"
  for ((i = 1; i <= attempts; i++)); do
    if curl -fsS --max-time 5 "$url" >/dev/null 2>&1; then
      log "$label is healthy"
      return 0
    fi
    sleep 5
  done
  die "$label did not become healthy in time — check: docker compose -f $APP_DIR/docker-compose.yml logs"
}

smoke_tests() {
  wait_for_health "http://127.0.0.1:8080/healthz" "site /healthz"
  wait_for_health "http://127.0.0.1:8080/api/twin/health?persona=cto" "twin proxy"

  if [[ "${SKIP_CADDY:-0}" != "1" ]] && [[ -n "$PUBLIC_BASE_URL" ]]; then
    if curl -fsSI --max-time 15 "${PUBLIC_BASE_URL}/healthz" >/dev/null 2>&1; then
      log "HTTPS health check passed: ${PUBLIC_BASE_URL}/healthz"
    else
      log "warning: HTTPS check failed — ensure DNS A records point to this host, then: sudo systemctl reload caddy"
    fi
  fi
}

print_summary() {
  cat <<EOF

================================================================================
 Neon AI Cloud deployment complete
================================================================================
 App directory : $APP_DIR
 Public URL    : ${PUBLIC_BASE_URL:-http://<host>:8080}
 Compose       : cd $APP_DIR && docker compose ps
 Logs          : cd $APP_DIR && docker compose logs -f site twin
 Update        : cd $APP_DIR && git pull && docker compose up --build -d
 Health        : curl -fsS ${PUBLIC_BASE_URL:-http://127.0.0.1:8080}/healthz
 Twin health   : curl -fsS '${PUBLIC_BASE_URL:-http://127.0.0.1:8080}/api/twin/health?persona=cto'
================================================================================
EOF
}

main() {
  require_root
  detect_ubuntu
  need_cmd curl
  need_cmd openssl

  resolve_repo_url
  resolve_domain
  resolve_public_base_url
  prompt_secrets

  [[ -n "$PUBLIC_BASE_URL" ]] || die "PUBLIC_BASE_URL is required"
  if [[ "$LLM_PROVIDER" == "openai" ]] && [[ -z "$OPENAI_API_KEY" ]]; then
    die "OPENAI_API_KEY is required when LLM_PROVIDER=openai"
  fi

  ensure_deploy_user
  apt_packages
  install_docker
  add_deploy_to_docker_group
  install_caddy
  configure_firewall
  sync_application
  write_env_file
  write_caddyfile
  install_systemd_unit
  build_and_deploy
  reload_caddy
  smoke_tests
  print_summary
}

main "$@"
