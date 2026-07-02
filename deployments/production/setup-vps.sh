#!/bin/sh
set -e

# ============================================================
# ShortURL — VPS Setup Script
# Usage: ./setup-vps.sh <vps-ip> <ssh-user>
# One-time VPS setup — no source code exposed
# ============================================================

VPS=${1:?Usage: setup-vps.sh <vps-ip> <ssh-user>}
USER=${2:?Usage: setup-vps.sh <vps-ip> <ssh-user>}
REMOTE_DIR="/opt/url-shortener/deployments/production"
DEPLOY_DIR="$(cd "$(dirname "$0")" && pwd)"

echo ">>> Setting up VPS: $USER@$VPS"

# ── Install Docker (skip if already installed) ──
echo ""
echo ">>> Checking Docker..."
if ssh "$USER@$VPS" "command -v docker > /dev/null 2>&1"; then
    echo "    Docker already installed, skipping."
else
    echo ">>> Installing Docker..."
    ssh "$USER@$VPS" "sudo apt-get update && sudo apt-get install -y docker.io docker-compose-plugin && sudo usermod -aG docker $USER"
fi

# ── Create deployment directory ──
echo ""
echo ">>> Creating deployment directory..."
ssh "$USER@$VPS" "sudo mkdir -p $REMOTE_DIR && sudo chown $USER:$USER $REMOTE_DIR"

# ── Copy deployment files ──
echo ""
echo ">>> Copying deployment files..."
scp "$DEPLOY_DIR/compose.yml" "$DEPLOY_DIR/deploy.sh" "$USER@$VPS:$REMOTE_DIR/"
scp -r "$DEPLOY_DIR/migrations" "$USER@$VPS:$REMOTE_DIR/"

# ── Make deploy.sh executable ──
echo ""
echo ">>> Making deploy.sh executable..."
ssh "$USER@$VPS" "chmod +x $REMOTE_DIR/deploy.sh"

echo ""
echo "============================================"
echo "  VPS setup complete!"
echo "============================================"
echo ""
echo "Next steps on VPS (SSH in):"
echo ""
echo "  1. Log out and log back in (for Docker group)"
echo ""
echo "  2. Set Infisical credentials:"
echo "     export INFISICAL_CLIENT_ID='<vps-deploy-client-id>'"
echo "     export INFISICAL_CLIENT_SECRET='<vps-deploy-client-secret>'"
echo "     export INFISICAL_PROJECT_ID='<project-uuid>'"
echo ""
echo "  3. Deploy:"
echo "     cd $REMOTE_DIR"
echo "     ./deploy.sh latest"
echo ""
