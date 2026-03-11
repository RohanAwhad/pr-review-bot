#!/usr/bin/env bash
set -euo pipefail

IMAGE_TAG="${1:-pr-review-bot-stage1:latest}"

podman build \
  -f "docker/stage1/Dockerfile" \
  -t "${IMAGE_TAG}" \
  .

echo "built ${IMAGE_TAG}"
