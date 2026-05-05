#!/usr/bin/env bash
set -euo pipefail

REPO="${CLIAI_GITHUB_REPOSITORY:-xjwm5685-ui/cliai}"
VERSION="${CLIAI_VERSION:-}"
DOWNLOAD_BASE="https://github.com/${REPO}/releases/latest/download"

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

detect_os() {
  case "$(uname -s)" in
    Linux) printf 'Linux' ;;
    Darwin) printf 'macOS' ;;
    *) fail "Unsupported operating system: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'x86_64' ;;
    arm64|aarch64) printf 'ARM64' ;;
    *) fail "Unsupported architecture: $(uname -m)" ;;
  esac
}

sha256_file() {
  local path="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$path" | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$path" | awk '{print $1}'
    return
  fi
  fail "Missing checksum tool: install sha256sum or shasum"
}

need_cmd curl
need_cmd tar

if [[ -n "${VERSION}" ]]; then
  VERSION="${VERSION#v}"
  DOWNLOAD_BASE="https://github.com/${REPO}/releases/download/v${VERSION}"
fi

asset_os="$(detect_os)"
asset_arch="$(detect_arch)"
archive_name="cliai_${asset_os}_${asset_arch}.tar.gz"
checksum_name="${archive_name}.sha256"

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

archive_path="${tmp_dir}/${archive_name}"
checksum_path="${tmp_dir}/${checksum_name}"
extract_dir="${tmp_dir}/extract"

log "Downloading ${archive_name} from ${DOWNLOAD_BASE}"
curl -fsSL "${DOWNLOAD_BASE}/${archive_name}" -o "${archive_path}"
curl -fsSL "${DOWNLOAD_BASE}/${checksum_name}" -o "${checksum_path}"

expected_hash="$(awk '{print $1}' "${checksum_path}")"
actual_hash="$(sha256_file "${archive_path}")"
if [[ "${expected_hash}" != "${actual_hash}" ]]; then
  fail "Checksum verification failed for ${archive_name}"
fi

mkdir -p "${extract_dir}"
tar -xzf "${archive_path}" -C "${extract_dir}"

installer_path="${extract_dir}/scripts/install-unix.sh"
[[ -x "${installer_path}" ]] || chmod 755 "${installer_path}"
[[ -f "${installer_path}" ]] || fail "Release archive did not contain scripts/install-unix.sh"

"${installer_path}"
