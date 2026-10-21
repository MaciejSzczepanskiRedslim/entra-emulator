#!/usr/bin/env bash
# Temporary ACR image publisher. Uses the current `az login` context.
# Usage: ./scripts/push-acr.sh
set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ -f "$ROOT/.env" ]]; then
  source "$ROOT/.env"
fi
: "${ACR_NAME:?Set ACR_NAME in .env or your environment.}"

TAG="dev-$(git -C "$ROOT" rev-parse --short=12 HEAD)"
if [[ -n "$(git -C "$ROOT" status --porcelain)" ]]; then
  TAG+="-dirty"
fi

LOGIN_SERVER="$(az acr show --name "$ACR_NAME" --query loginServer --output tsv)"
IMAGE="$LOGIN_SERVER/entra-emulator"
az acr login --name "$ACR_NAME"

docker build --platform linux/amd64 --build-arg "VERSION=$TAG" \
  --tag "$IMAGE:$TAG" --tag "$IMAGE:dev" "$ROOT"
docker push "$IMAGE:$TAG"
docker push "$IMAGE:dev"

printf '\nPublished:\n  %s:%s\n  %s:dev\n' "$IMAGE" "$TAG" "$IMAGE"
