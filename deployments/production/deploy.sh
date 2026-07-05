#!/bin/sh
set -e

# ============================================================
# ShortURL: Production Deploy Script
# Usage: ./deploy.sh <image-tag>
# Requires: INFISICAL_CLIENT_ID, INFISICAL_CLIENT_SECRET env vars
# ============================================================

IMAGE_TAG=${1:?Usage: deploy.sh <image-tag>}
COMPOSE_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_SERVICE="app"
HEALTH_URL="http://localhost:7500/health"
HEALTH_RETRIES=10
HEALTH_INTERVAL=3

cd "$COMPOSE_DIR"

# ── Authenticate with Infisical ──
echo ">>> Authenticating with Infisical..."

if [ -z "$INFISICAL_CLIENT_ID" ] || [ -z "$INFISICAL_CLIENT_SECRET" ] || [ -z "$INFISICAL_PROJECT_ID" ]; then
    echo "!!! ERROR: Required environment variables not set:"
    echo "    - INFISICAL_CLIENT_ID"
    echo "    - INFISICAL_CLIENT_SECRET"
    echo "    - INFISICAL_PROJECT_ID"
    echo ""
    echo "    Set these before running deploy.sh:"
    echo "      export INFISICAL_CLIENT_ID='<your-client-id>'"
    echo "      export INFISICAL_CLIENT_SECRET='<your-client-secret>'"
    echo "      export INFISICAL_PROJECT_ID='<your-project-id>'"
    exit 1
fi

export INFISICAL_TOKEN=$(infisical login \
    --method=universal-auth \
    --client-id="$INFISICAL_CLIENT_ID" \
    --client-secret="$INFISICAL_CLIENT_SECRET" \
    --silent --plain)

if [ -z "$INFISICAL_TOKEN" ]; then
    echo "!!! ERROR: Failed to authenticate with Infisical"
    exit 1
fi

echo "    Authenticated successfully."

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

# ── Validate required environment variables ──
echo ""
echo ">>> Validating environment variables..."
if [ -z "$INFISICAL_PROJECT_ID" ]; then
    echo "!!! ERROR: INFISICAL_PROJECT_ID not set"
    exit 1
fi

# ── Log in to GHCR ──
echo ""
echo ">>> Authenticating Docker with GitHub Container Registry..."
infisical run \
    --projectId="$INFISICAL_PROJECT_ID" \
    --env="prod" \
    -- sh -c 'echo "$REGISTRY_PASSWORD" | docker login ghcr.io -u "$REGISTRY_USERNAME" --password-stdin'

# ── Pull new image (with secrets injected) ──
echo ""
echo ">>> Pulling image..."
infisical run \
    --projectId="$INFISICAL_PROJECT_ID" \
    --env="prod" \
    -- docker compose pull $APP_SERVICE

# ── Ensure database and cache are running ──
echo ""
echo ">>> Ensuring database and cache services are running..."
infisical run \
    --projectId="$INFISICAL_PROJECT_ID" \
    --env="prod" \
    -- docker compose up -d db redis

# ── Run migrations (with secrets injected) ──
echo ""
echo ">>> Running migrations..."
infisical run \
    --projectId="$INFISICAL_PROJECT_ID" \
    --env="prod" \
    -- docker compose run --rm migrate

# ── Deploy new version (with secrets injected) ──
echo ""
echo ">>> Starting new version..."
infisical run \
    --projectId="$INFISICAL_PROJECT_ID" \
    --env="prod" \
    -- docker compose up -d --no-deps $APP_SERVICE

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
    echo ">>> Rolling back to ${CURRENT_TAG}..."

    if [ "$CURRENT_TAG" = "none" ]; then
        echo ">>> No previous version to roll back to. Stopping."
        docker compose stop $APP_SERVICE
        exit 1
    fi

    # Rollback: use the previous image tag
    IMAGE_TAG="$CURRENT_TAG" infisical run \
        --projectId="$INFISICAL_PROJECT_ID" \
        --env="prod" \
        -- docker compose up -d --no-deps $APP_SERVICE

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
