#!/usr/bin/env bash
set -euo pipefail

KEY_FILE="${CLIAI_APT_GPG_KEY_FILE:-}"
KEY_ARMORED="${CLIAI_APT_GPG_KEY_ARMORED:-}"
KEY_ID="${CLIAI_APT_GPG_KEY_ID:-}"
OUTPUT_PATH="cliai-archive-keyring.asc"

usage() {
  cat >&2 <<'EOF'
usage: ./scripts/export-apt-public-key.sh \
  [--key-file <private-key.asc>] \
  [--key-id <fingerprint-or-id>] \
  [--output <path>]

Environment fallbacks:
  CLIAI_APT_GPG_KEY_FILE
  CLIAI_APT_GPG_KEY_ARMORED
  CLIAI_APT_GPG_KEY_ID
EOF
}

require_value() {
  local flag="$1"
  local value="${2:-}"
  if [[ -z "${value}" ]]; then
    echo "missing value for ${flag}" >&2
    usage
    exit 1
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --key-file)
      require_value "$1" "${2:-}"
      KEY_FILE="$2"
      shift 2
      ;;
    --key-id)
      require_value "$1" "${2:-}"
      KEY_ID="$2"
      shift 2
      ;;
    --output)
      require_value "$1" "${2:-}"
      OUTPUT_PATH="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if ! command -v gpg >/dev/null 2>&1; then
  echo "gpg is required to export the apt public key." >&2
  exit 1
fi

if [[ -z "${KEY_FILE}" && -z "${KEY_ARMORED}" ]]; then
  echo "No apt signing key provided." >&2
  exit 1
fi

temp_dir="$(mktemp -d)"
gnupg_home="${temp_dir}/gnupg"
mkdir -p "${gnupg_home}"
chmod 700 "${gnupg_home}"
export GNUPGHOME="${gnupg_home}"

cleanup() {
  rm -rf "${temp_dir}"
}
trap cleanup EXIT

if [[ -n "${KEY_FILE}" ]]; then
  if [[ ! -f "${KEY_FILE}" ]]; then
    echo "apt signing key file not found: ${KEY_FILE}" >&2
    exit 1
  fi
  gpg --batch --import "${KEY_FILE}" >/dev/null 2>&1
else
  key_path="${temp_dir}/apt-signing-key.asc"
  printf "%s\n" "${KEY_ARMORED}" > "${key_path}"
  gpg --batch --import "${key_path}" >/dev/null 2>&1
fi

if [[ -z "${KEY_ID}" ]]; then
  KEY_ID="$(gpg --batch --list-secret-keys --with-colons | awk -F: '/^sec:/ {print $5; exit}')"
fi

if [[ -z "${KEY_ID}" ]]; then
  echo "Failed to determine apt signing key id." >&2
  exit 1
fi

mkdir -p "$(dirname "${OUTPUT_PATH}")"
gpg --batch --yes --armor --output "${OUTPUT_PATH}" --export "${KEY_ID}"

echo "apt public key exported to ${OUTPUT_PATH}"
