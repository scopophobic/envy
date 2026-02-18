#!/usr/bin/env bash
set -euo pipefail

# ─── Envo Deploy Script (AWS EC2) ───
#
# Prerequisites on the EC2 instance:
#   sudo yum install -y docker git        # Amazon Linux 2023
#   sudo systemctl enable --now docker
#   sudo usermod -aG docker $USER         # then re-login
#   sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
#   sudo chmod +x /usr/local/bin/docker-compose
#
# Usage:
#   1. Clone repo on EC2
#   2. cp .env.production.example .env.production
#   3. Edit .env.production with real values
#   4. ./deploy.sh
#

ENV_FILE=".env.production"

if [ ! -f "$ENV_FILE" ]; then
  echo "ERROR: $ENV_FILE not found."
  echo "Copy .env.production.example to .env.production and fill in your values."
  exit 1
fi

echo "==> Building containers..."
docker-compose --env-file "$ENV_FILE" build

echo "==> Starting services..."
docker-compose --env-file "$ENV_FILE" up -d

echo "==> Waiting for database to be ready..."
sleep 5

echo "==> Running migrations..."
docker-compose --env-file "$ENV_FILE" exec backend envo-server -migrate

echo "==> Seeding tier data..."
docker-compose --env-file "$ENV_FILE" exec backend envo-server -seed

echo ""
echo "==> Envo is running!"
echo "    Frontend: http://$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4 2>/dev/null || echo 'your-server-ip')"
echo "    Health:   curl http://localhost/health"
echo ""
echo "Next steps:"
echo "  - Point your subdomain DNS to this server's IP"
echo "  - Set up HTTPS with: sudo certbot --nginx -d your-subdomain.yourdomain.com"
