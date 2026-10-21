#!/usr/bin/env bash
# Temporary ACR image publisher. Uses the current `az login` context.
# Usage: ./scripts/push-acr.sh
# Promote a tested image: ./scripts/push-acr.sh --promote dev-<commit>
set -euo pipefail

if [[ $# -ne 0 && ! ( $# -eq 2 && "$1" == --promote ) ]]; then
  echo "Usage: $0 [--promote <tested-tag>]" >&2
  exit 2
fi

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ -f "$ROOT/.env" ]]; then
  source "$ROOT/.env"
fi
: "${ACR_NAME:?Set ACR_NAME in .env or your environment.}"

LOGIN_SERVER="$(az acr show --name "$ACR_NAME" --query loginServer --output tsv)"
IMAGE="$LOGIN_SERVER/entra-emulator"
az acr login --name "$ACR_NAME"

if [[ $# -eq 2 ]]; then
  docker pull "$IMAGE:$2"
  docker tag "$IMAGE:$2" "$IMAGE:stable"
  docker push "$IMAGE:stable"
  printf '\nPromoted %s:%s to %s:stable\n' "$IMAGE" "$2" "$IMAGE"
  exit 0
fi

TAG="dev-$(git -C "$ROOT" rev-parse --short=12 HEAD)"
if [[ -n "$(git -C "$ROOT" status --porcelain)" ]]; then
  TAG+="-dirty"
fi

docker build --platform linux/amd64 --build-arg "VERSION=$TAG" \
  --tag "$IMAGE:$TAG" --tag "$IMAGE:dev" "$ROOT"
docker push "$IMAGE:$TAG"
docker push "$IMAGE:dev"

printf '\nPublished:\n  %s:%s\n  %s:dev\n' "$IMAGE" "$TAG" "$IMAGE"
