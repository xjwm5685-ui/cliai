#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${CLIAI_APT_REPO_URL:-https://raw.githubusercontent.com/xjwm5685-ui/cliai/apt-repo}"
KEY_URL="${CLIAI_APT_KEY_URL:-${REPO_URL}/cliai-archive-keyring.asc}"
LIST_PATH="/etc/apt/sources.list.d/cliai.list"
KEYRING_PATH="/etc/apt/keyrings/cliai-archive-keyring.asc"
INSTALL_PACKAGE="${CLIAI_INSTALL_PACKAGE:-0}"
INSTALL_ZSH="${CLIAI_INSTALL_ZSH:-0}"
ENABLE_ZSH="${CLIAI_ENABLE_ZSH:-0}"
ENABLE_BASH="${CLIAI_ENABLE_BASH:-0}"
ENABLE_POWERSHELL="${CLIAI_ENABLE_POWERSHELL:-0}"
ENABLE_POWERSHELL_HELPERS="${CLIAI_ENABLE_POWERSHELL_HELPERS:-0}"

log() {
  printf '%s\n' "$*"
}

fail() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "Missing required command: $1"
}

is_true() {
  case "${1:-}" in
    1|true|TRUE|yes|YES|on|ON) return 0 ;;
    *) return 1 ;;
  esac
}

run_as_root() {
  if [[ "${EUID}" -eq 0 ]]; then
    "$@"
  elif command -v sudo >/dev/null 2>&1; then
    sudo "$@"
  else
    fail "This installer needs root privileges. Re-run as root or install sudo."
  fi
}

run_as_invoking_user() {
  if [[ "${EUID}" -eq 0 && -n "${SUDO_USER:-}" && "${SUDO_USER}" != "root" ]]; then
    sudo -u "${SUDO_USER}" "$@"
  else
    "$@"
  fi
}

need_cmd curl

if ! command -v apt-get >/dev/null 2>&1; then
  fail "This installer only supports Debian/Ubuntu systems with apt-get."
fi

if ! curl -fsSL "${REPO_URL}/dists/stable/Release" >/dev/null; then
  fail "The cliai apt repository is not reachable at ${REPO_URL}."
fi

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

list_content=""
if curl -fsSL "${KEY_URL}" -o "${tmp_dir}/cliai-archive-keyring.asc" 2>/dev/null; then
  if grep -q "BEGIN PGP PUBLIC KEY BLOCK" "${tmp_dir}/cliai-archive-keyring.asc"; then
    run_as_root mkdir -p /etc/apt/keyrings
    run_as_root cp "${tmp_dir}/cliai-archive-keyring.asc" "${KEYRING_PATH}"
    list_content="deb [signed-by=${KEYRING_PATH}] ${REPO_URL} stable main"
    log "Installed apt key to ${KEYRING_PATH}"
  fi
fi

if [[ -z "${list_content}" ]]; then
  list_content="deb [trusted=yes] ${REPO_URL} stable main"
  log "No apt signing key published. Falling back to trusted apt source."
fi

printf '%s\n' "${list_content}" > "${tmp_dir}/cliai.list"
run_as_root mkdir -p "$(dirname "${LIST_PATH}")"
run_as_root cp "${tmp_dir}/cliai.list" "${LIST_PATH}"

log "Installed apt source to ${LIST_PATH}"

packages=()
if is_true "${INSTALL_PACKAGE}"; then
  packages+=(cliai)
fi
if is_true "${INSTALL_ZSH}"; then
  packages+=(zsh)
fi

if [[ "${#packages[@]}" -gt 0 ]]; then
  run_as_root apt-get update
  run_as_root apt-get install -y "${packages[@]}"
  log "Installed packages: ${packages[*]}"
fi

if is_true "${ENABLE_ZSH}"; then
  command -v cliai >/dev/null 2>&1 || fail "cliai is not installed yet; set CLIAI_INSTALL_PACKAGE=1 or install it first."
  run_as_invoking_user cliai shell install zsh
  log "Installed zsh shell integration."
fi

if is_true "${ENABLE_BASH}"; then
  command -v cliai >/dev/null 2>&1 || fail "cliai is not installed yet; set CLIAI_INSTALL_PACKAGE=1 or install it first."
  run_as_invoking_user cliai shell install bash
  log "Installed bash shell integration."
fi

if is_true "${ENABLE_POWERSHELL}"; then
  command -v cliai >/dev/null 2>&1 || fail "cliai is not installed yet; set CLIAI_INSTALL_PACKAGE=1 or install it first."
  run_as_invoking_user cliai shell install powershell
  log "Installed PowerShell predictor integration."
fi

if is_true "${ENABLE_POWERSHELL_HELPERS}"; then
  command -v cliai >/dev/null 2>&1 || fail "cliai is not installed yet; set CLIAI_INSTALL_PACKAGE=1 or install it first."
  run_as_invoking_user cliai shell install powershell-helpers
  log "Installed PowerShell helper aliases."
fi

log ""
if [[ "${#packages[@]}" -eq 0 ]]; then
  log "Next steps:"
  log "  sudo apt update"
  log "  sudo apt install cliai"
  log "  cliai shell install zsh"
  log ""
  log "Or run everything in one step:"
  log "  curl -fsSL https://raw.githubusercontent.com/xjwm5685-ui/cliai/main/install.sh | env CLIAI_INSTALL_PACKAGE=1 CLIAI_INSTALL_ZSH=1 CLIAI_ENABLE_ZSH=1 bash"
else
  log "Installation complete."
fi

if is_true "${ENABLE_ZSH}"; then
  log "Run 'exec zsh' to switch into zsh now."
fi
