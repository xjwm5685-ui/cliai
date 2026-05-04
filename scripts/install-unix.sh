#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_SOURCE="${ROOT_DIR}/cliai"
BIN_DIR="${HOME}/.local/bin"
BIN_TARGET="${BIN_DIR}/cliai"

mkdir -p "${BIN_DIR}"
install -m 755 "${BIN_SOURCE}" "${BIN_TARGET}"

echo "Installed cliai to ${BIN_TARGET}"
echo "Make sure ${BIN_DIR} is in your PATH."

if command -v pwsh >/dev/null 2>&1; then
  echo "PowerShell 7 detected. To enable predictive integration, run:"
  echo "  cliai shell install powershell"
fi
