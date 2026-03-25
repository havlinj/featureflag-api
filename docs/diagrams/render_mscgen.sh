#!/usr/bin/env bash
# Render all *_mscgen.msc files in this directory to PNG and SVG.
#
# Usage:
#   ./docs/diagrams/render_mscgen.sh
#
# Uses a local `mscgen` if installed (e.g. apt install mscgen). Otherwise builds
# and runs Docker image from Dockerfile.mscgen (override name: MSC_IMAGE=…).

set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE="${MSC_IMAGE:-featureflag-mscgen}"

shopt -s nullglob
maps=( "$DIR"/*_mscgen.msc )
if ((${#maps[@]} == 0)); then
  echo "No *_mscgen.msc files in $DIR" >&2
  exit 1
fi

render_with_docker() {
  if ! command -v docker &>/dev/null; then
    echo "Neither mscgen nor docker found. Install one of: mscgen, Docker." >&2
    exit 1
  fi
  if ! docker image inspect "$IMAGE" &>/dev/null; then
    echo "Building $IMAGE (one-time)…" >&2
    docker build -f "$DIR/Dockerfile.mscgen" -t "$IMAGE" "$DIR"
  fi
  for msc in "${maps[@]}"; do
    base="$(basename "$msc" .msc)"
    name="$(basename "$msc")"
    echo "Rendering $base → PNG, SVG (docker)" >&2
    docker run --rm -v "$DIR:/work" -w /work "$IMAGE" -T png -i "/work/$name" -o "/work/${base}.png"
    docker run --rm -v "$DIR:/work" -w /work "$IMAGE" -T svg -i "/work/$name" -o "/work/${base}.svg"
  done
}

if command -v mscgen &>/dev/null; then
  for msc in "${maps[@]}"; do
    base="${msc%.msc}"
    echo "Rendering $(basename "$base") → PNG, SVG (local mscgen)" >&2
    mscgen -T png -i "$msc" -o "${base}.png"
    mscgen -T svg -i "$msc" -o "${base}.svg"
  done
else
  render_with_docker
fi

echo "Done." >&2
