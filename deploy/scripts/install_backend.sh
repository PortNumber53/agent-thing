#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME="agent-thing"
INSTALL_ROOT="/opt/agent-thing"
BIN_DIR="${INSTALL_ROOT}/bin"
ETC_DIR="/etc/agent-thing"
CONFIG_PATH="${ETC_DIR}/config.ini"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

mkdir -p "${BIN_DIR}" "${ETC_DIR}"

if [[ ! -f "${CONFIG_PATH}" ]]; then
  echo "[install] ${CONFIG_PATH} not found; installing sample"
  cp "${REPO_ROOT}/deploy/config.ini.sample" "${CONFIG_PATH}"
fi

echo "[install] installing systemd unit"
cp "${REPO_ROOT}/deploy/systemd/agent-thing.service" "/etc/systemd/system/${SERVICE_NAME}.service"

echo "[install] enabling + restarting service"
systemctl daemon-reload
systemctl enable "${SERVICE_NAME}.service"
systemctl restart "${SERVICE_NAME}.service"

systemctl status "${SERVICE_NAME}.service" --no-pager || true
