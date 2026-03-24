#!/usr/bin/env bash
set -euo pipefail

# ─── Envo Deploy Script (AWS EC2 — backend only) ───
#
# Frontend is deployed separately (e.g. Vercel).
# This script builds and starts only the Go backend container.
#
# Prerequisites on the EC2 instance:
#   sudo apt install -y docker.io docker-compose-plugin   # Ubuntu
#   sudo systemctl enable --now docker
#   sudo usermod -aG docker $USER   # then re-login
#
# Usage:
#   1. Clone repo on EC2
#   2. cp .env.production.example .env.production
#   3. Edit .env.production with real values
#   4. chmod +x deploy.sh && ./deploy.sh
#

ENV_FILE=".env.production"

if [ ! -f "$ENV_FILE" ]; then
  echo "ERROR: $ENV_FILE not found."
  echo "Copy .env.production.example to .env.production and fill in your values."
  exit 1
fi

echo "==> Building backend..."
docker-compose --env-file "$ENV_FILE" build backend

echo "==> Starting backend..."
docker-compose --env-file "$ENV_FILE" up -d backend

echo "==> Waiting for backend to start..."
sleep 3

echo "==> Running migrations..."
docker-compose --env-file "$ENV_FILE" exec backend envo-server -migrate

echo "==> Seeding tier data..."
docker-compose --env-file "$ENV_FILE" exec backend envo-server -seed

HOST_PORT=$(grep -E '^HOST_PORT=' "$ENV_FILE" | cut -d= -f2 || echo "8080")

echo ""
echo "==> Envo backend is running on port ${HOST_PORT}!"
echo "    Health: curl http://localhost:${HOST_PORT}/health"
echo ""
echo "Next steps:"
echo "  - Ensure host Nginx proxies your subdomain to 127.0.0.1:${HOST_PORT}"
echo "  - Set VITE_API_URL on Vercel to https://your-api-subdomain"
echo "  - Set GOOGLE_REDIRECT_URL in Google Console"
