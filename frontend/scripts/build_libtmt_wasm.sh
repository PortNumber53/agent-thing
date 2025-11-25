#!/usr/bin/env bash
set -euo pipefail

# Build libtmt.wasm using Dockerized Emscripten.
# No local emcc install required.
#
# Requires: Docker running.
#
# Output: frontend/public/libtmt.wasm

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRONTEND_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
OUT_DIR="${FRONTEND_DIR}/public"

mkdir -p "${OUT_DIR}"

# On arm64 hosts, Emscripten image is amd64; Docker will emulate.
# You can override platform if needed: DOCKER_PLATFORM=linux/amd64 ./scripts/build_libtmt_wasm.sh
DOCKER_PLATFORM="${DOCKER_PLATFORM:-}"

docker run --rm ${DOCKER_PLATFORM:+--platform "$DOCKER_PLATFORM"} \
  -v "${FRONTEND_DIR}:/src" \
  -w /src \
  emscripten/emsdk:3.1.74 \
  emcc src/terminal/libtmt/tmt.c src/terminal/libtmt/tmt_wasm.c \
    -O3 -s STANDALONE_WASM=1 -s NO_ENTRY=1 \
    -s EXPORTED_FUNCTIONS='["_tmtw_open","_tmtw_close","_tmtw_write","_tmtw_resize","_tmtw_dump","_tmtw_get_cursor","_malloc","_free"]' \
    -o public/libtmt.wasm

echo "[build_libtmt_wasm] wrote ${OUT_DIR}/libtmt.wasm"
