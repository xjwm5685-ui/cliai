#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

REPO_ROOT=""
DISTRIBUTION="stable"
KEY_FILE="${CLIAI_APT_GPG_KEY_FILE:-}"
KEY_ARMORED="${CLIAI_APT_GPG_KEY_ARMORED:-}"
KEY_ID="${CLIAI_APT_GPG_KEY_ID:-}"
PASSPHRASE="${CLIAI_APT_GPG_PASSPHRASE:-}"
REQUIRE_SIGNATURE=0

usage() {
  cat >&2 <<'EOF'
usage: ./scripts/sign-apt-repo.sh \
  --repo-root <dir> \
  [--distribution stable] \
  [--key-file <private-key.asc>] \
  [--key-id <fingerprint-or-id>] \
  [--passphrase <passphrase>] \
  [--require-signature]

Environment fallbacks:
  CLIAI_APT_GPG_KEY_FILE
  CLIAI_APT_GPG_KEY_ARMORED
  CLIAI_APT_GPG_KEY_ID
  CLIAI_APT_GPG_PASSPHRASE
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
    --repo-root)
      require_value "$1" "${2:-}"
      REPO_ROOT="$2"
      shift 2
      ;;
    --distribution)
      require_value "$1" "${2:-}"
      DISTRIBUTION="$2"
      shift 2
      ;;
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
    --passphrase)
      require_value "$1" "${2:-}"
      PASSPHRASE="$2"
      shift 2
      ;;
    --require-signature)
      REQUIRE_SIGNATURE=1
      shift
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

if [[ -z "${REPO_ROOT}" ]]; then
  echo "missing required --repo-root" >&2
  usage
  exit 1
fi

release_file="${REPO_ROOT}/dists/${DISTRIBUTION}/Release"
if [[ ! -f "${release_file}" ]]; then
  echo "Release file not found: ${release_file}" >&2
  exit 1
fi

if ! command -v gpg >/dev/null 2>&1; then
  echo "gpg is required to sign apt repository metadata." >&2
  exit 1
fi

if [[ -z "${KEY_FILE}" && -z "${KEY_ARMORED}" ]]; then
  if [[ "${REQUIRE_SIGNATURE}" -eq 1 ]]; then
    echo "No apt signing key provided." >&2
    exit 1
  fi
  echo "No apt signing key provided. Skipping apt repo signing."
  exit 0
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

gpg_base_args=(
  --batch
  --yes
  --pinentry-mode loopback
  --local-user "${KEY_ID}"
)

if [[ -n "${PASSPHRASE}" ]]; then
  gpg_base_args+=(--passphrase "${PASSPHRASE}")
fi

release_gpg="${REPO_ROOT}/dists/${DISTRIBUTION}/Release.gpg"
inrelease="${REPO_ROOT}/dists/${DISTRIBUTION}/InRelease"
rm -f "${release_gpg}" "${inrelease}"

gpg "${gpg_base_args[@]}" \
  --armor \
  --detach-sign \
  --output "${release_gpg}" \
  "${release_file}"

gpg "${gpg_base_args[@]}" \
  --armor \
  --clearsign \
  --output "${inrelease}" \
  "${release_file}"

echo "apt repository signing files generated:"
echo "  ${release_gpg}"
echo "  ${inrelease}"
