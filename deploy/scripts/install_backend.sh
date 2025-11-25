#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME="agent-thing"
INSTALL_ROOT="/opt/agent-thing"
BIN_DIR="${INSTALL_ROOT}/bin"
ETC_DIR="/etc/agent-thing"
CONFIG_PATH="${ETC_DIR}/config.ini"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# When invoked from Jenkins deploy we only scp this script (and optionally sample/unit)
# into /tmp on the target; the full repo may not be present.
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." 2>/dev/null && pwd || true)"

mkdir -p "${BIN_DIR}" "${ETC_DIR}"

if [[ ! -f "${CONFIG_PATH}" ]]; then
  echo "[install] ${CONFIG_PATH} not found; installing sample"
  SAMPLE_SRC=""
  if [[ -f "${SCRIPT_DIR}/config.ini.sample" ]]; then
    SAMPLE_SRC="${SCRIPT_DIR}/config.ini.sample"
  elif [[ -n "${REPO_ROOT}" && -f "${REPO_ROOT}/deploy/config.ini.sample" ]]; then
    SAMPLE_SRC="${REPO_ROOT}/deploy/config.ini.sample"
  fi

  if [[ -n "${SAMPLE_SRC}" ]]; then
    cp "${SAMPLE_SRC}" "${CONFIG_PATH}"
  else
    echo "[install] sample config not found on target; creating empty ${CONFIG_PATH}"
    cat > "${CONFIG_PATH}" <<'EOF'
[app]
# Fill these values or set env vars; env overrides ini.
APP_BASE_URL=
BACKEND_BASE_URL=

[database]
DATABASE_URL=
XATA_DATABASE_URL=
XATA_API_KEY=

[google]
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GOOGLE_REDIRECT_URL=
JWT_SECRET=

[stripe]
STRIPE_SECRET_KEY=
STRIPE_PUBLISHABLE_KEY=
STRIPE_WEBHOOK_SECRET=
STRIPE_PRICE_ID=
EOF
  fi
fi

echo "[install] installing systemd unit"
UNIT_SRC=""
if [[ -f "${SCRIPT_DIR}/agent-thing.service" ]]; then
  UNIT_SRC="${SCRIPT_DIR}/agent-thing.service"
elif [[ -n "${REPO_ROOT}" && -f "${REPO_ROOT}/deploy/systemd/agent-thing.service" ]]; then
  UNIT_SRC="${REPO_ROOT}/deploy/systemd/agent-thing.service"
fi

if [[ -n "${UNIT_SRC}" ]]; then
  cp "${UNIT_SRC}" "/etc/systemd/system/${SERVICE_NAME}.service"
else
  echo "[install] systemd unit not found on target; skipping unit install"
  exit 1
fi

echo "[install] enabling + restarting service"
systemctl daemon-reload
systemctl enable "${SERVICE_NAME}.service"
systemctl restart "${SERVICE_NAME}.service"

systemctl status "${SERVICE_NAME}.service" --no-pager || true
