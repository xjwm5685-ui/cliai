#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${CLIAI_APT_REPO_URL:-https://raw.githubusercontent.com/xjwm5685-ui/cliai/apt-repo}"
KEY_URL="${CLIAI_APT_KEY_URL:-${REPO_URL}/cliai-archive-keyring.asc}"
LIST_PATH="/etc/apt/sources.list.d/cliai.list"
KEYRING_PATH="/etc/apt/keyrings/cliai-archive-keyring.asc"

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

run_as_root() {
  if [[ "${EUID}" -eq 0 ]]; then
    "$@"
  elif command -v sudo >/dev/null 2>&1; then
    sudo "$@"
  else
    fail "This installer needs root privileges. Re-run as root or install sudo."
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
log ""
log "Next steps:"
log "  sudo apt update"
log "  sudo apt install cliai"
