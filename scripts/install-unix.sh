#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_SOURCE="${ROOT_DIR}/cliai"
BIN_DIR="${CLIAI_INSTALL_BIN_DIR:-${HOME}/.local/bin}"
BIN_TARGET="${BIN_DIR}/cliai"
ENABLE_ZSH="${CLIAI_ENABLE_ZSH:-0}"
ENABLE_BASH="${CLIAI_ENABLE_BASH:-0}"
ENABLE_POWERSHELL="${CLIAI_ENABLE_POWERSHELL:-0}"
ENABLE_POWERSHELL_HELPERS="${CLIAI_ENABLE_POWERSHELL_HELPERS:-0}"

is_true() {
  case "${1:-}" in
    1|true|TRUE|yes|YES|on|ON) return 0 ;;
    *) return 1 ;;
  esac
}

mkdir -p "${BIN_DIR}"
install -m 755 "${BIN_SOURCE}" "${BIN_TARGET}"

echo "Installed cliai to ${BIN_TARGET}"
echo "Make sure ${BIN_DIR} is in your PATH."

if is_true "${ENABLE_ZSH}"; then
  "${BIN_TARGET}" shell install zsh
  echo "Installed zsh shell integration."
fi

if is_true "${ENABLE_BASH}"; then
  "${BIN_TARGET}" shell install bash
  echo "Installed bash shell integration."
fi

if is_true "${ENABLE_POWERSHELL}"; then
  "${BIN_TARGET}" shell install powershell
  echo "Installed PowerShell predictor integration."
fi

if is_true "${ENABLE_POWERSHELL_HELPERS}"; then
  "${BIN_TARGET}" shell install powershell-helpers
  echo "Installed PowerShell helper aliases."
fi

current_shell="$(basename "${SHELL:-}")"
if [[ "${current_shell}" == "bash" || "${current_shell}" == "zsh" ]]; then
  echo "To enable cliai suggestions in your current shell, run:"
  echo "  cliai shell install ${current_shell}"
fi

if command -v zsh >/dev/null 2>&1 && [[ "${current_shell}" != "zsh" ]]; then
  echo "For native grey inline suggestions on Linux/macOS or WSL2, zsh is recommended:"
  echo "  cliai shell install zsh"
fi

if command -v pwsh >/dev/null 2>&1; then
  echo "PowerShell 7 detected. To enable predictive integration, run:"
  echo "  cliai shell install powershell"
  echo "Or helper aliases only:"
  echo "  cliai shell install powershell-helpers"
fi

if is_true "${ENABLE_ZSH}"; then
  echo "Run 'exec zsh' to switch into zsh now."
fi
