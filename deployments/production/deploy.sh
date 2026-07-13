#!/bin/sh
set -e

# ============================================================
# ShortURL: Production Deploy Script
# Usage: infisical run --env=prod -- ./deploy.sh <image-tag>
# Note: Should be run INSIDE infisical run (secrets already in env)
# ============================================================

IMAGE_TAG=${1:?Usage: deploy.sh <image-tag>}
COMPOSE_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_SERVICE="app"
HEALTH_URL="http://localhost:7500/healthz"
HEALTH_RETRIES=20
HEALTH_INTERVAL=5

cd "$COMPOSE_DIR"

# ── Validate required secrets are present ──
echo ">>> Validating environment variables..."
if [ -z "$DB_PASS" ] || [ -z "$DB_USER" ] || [ -z "$DB_NAME" ]; then
    echo "!!! ERROR: Required database secrets not found in environment"
    echo "    Make sure this script is run inside: infisical run --env=prod --"
    exit 1
fi

echo "    ✓ Database secrets present"

# ── Save current image for rollback ──
CURRENT_IMAGE=$(docker compose ps -q $APP_SERVICE 2>/dev/null | xargs docker inspect --format='{{.Config.Image}}' 2>/dev/null || echo "")
CURRENT_TAG=$(echo "$CURRENT_IMAGE" | cut -d: -f2)
if [ -z "$CURRENT_TAG" ]; then
    CURRENT_TAG="none"
    echo "No existing deployment found."
else
    echo "Current image: $CURRENT_IMAGE"
fi

echo "Deploying: ghcr.io/gopal-chhetri/url-shortener:${IMAGE_TAG}"

# ── Log in to GHCR ──
echo ""
echo ">>> Authenticating Docker with GitHub Container Registry..."
echo "$REGISTRY_PASSWORD" | docker login ghcr.io -u "$REGISTRY_USERNAME" --password-stdin

# ── Pull new image ──
echo ""
echo ">>> Pulling image..."
export IMAGE_TAG
docker compose pull $APP_SERVICE

# ── Ensure database and cache are running (wait for healthchecks) ──
echo ""
echo ">>> Ensuring database and cache services are running..."
docker compose up -d --wait db redis

# ── Reconcile DB credentials (self-heal on password rotation) ──
# POSTGRES_PASSWORD is only applied on FIRST init of the pgdata volume.
# If the secret was rotated later, the persisted role keeps the old password.
# Local socket connections use trust auth, so we can reset it without one.
echo ""
echo ">>> Reconciling database credentials..."
docker compose exec -T db \
    psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d postgres \
    -v role="$DB_USER" -v pass="$DB_PASS" <<'SQL'
ALTER ROLE :"role" WITH LOGIN PASSWORD :'pass';
SQL
echo "    ✓ Database password synced"

# ── Run migrations ──
echo ""
echo ">>> Running migrations..."
docker compose run --rm migrate

# ── Deploy new version ──
echo ""
echo ">>> Starting new version..."
docker compose up -d --no-deps $APP_SERVICE

# ── Health check ──
echo ""
echo ">>> Waiting for health check..."
RETRIES=$HEALTH_RETRIES
until [ $RETRIES -eq 0 ] || docker compose exec -T $APP_SERVICE wget -qO- "$HEALTH_URL" > /dev/null 2>&1; do
    RETRIES=$((RETRIES - 1))
    echo "    Retries left: $RETRIES"
    sleep $HEALTH_INTERVAL
done

if [ $RETRIES -eq 0 ]; then
    echo ""
    echo "!!! Health check failed!"
    echo ""
    echo ">>> Showing last 50 lines of application logs:"
    docker compose logs --tail=50 $APP_SERVICE
    echo ""
    echo ">>> Rolling back to ${CURRENT_TAG}..."

    if [ "$CURRENT_TAG" = "none" ]; then
        echo ">>> No previous version to roll back to. Stopping."
        docker compose stop $APP_SERVICE
        exit 1
    fi

    # Rollback: use the previous image tag
    IMAGE_TAG="$CURRENT_TAG" docker compose up -d --no-deps $APP_SERVICE

    # Verify rollback
    sleep 5
    if docker compose exec -T $APP_SERVICE wget -qO- "$HEALTH_URL" > /dev/null 2>&1; then
        echo ">>> Rollback successful. Running: ${CURRENT_TAG}"
    else
        echo "!!! Rollback also failed. Manual intervention required."
    fi
    exit 1
fi

echo ""
echo ">>> Deployment successful!"
echo "    Image: ghcr.io/gopal-chhetri/url-shortener:${IMAGE_TAG}"

# ── Prune old images ──
echo ""
echo ">>> Pruning old images..."
docker image prune -f

echo ""
echo ">>> Done!"
