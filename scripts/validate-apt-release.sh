#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=""
DISTRIBUTION="stable"
PUBLIC_KEY=""
REQUIRE_SIGNATURE=0
DEB_FILES=()

usage() {
  cat >&2 <<'EOF'
usage: ./scripts/validate-apt-release.sh \
  --repo-root <dir> \
  --deb <path/to/package.deb> [--deb <path/to/package.deb> ...] \
  [--distribution stable] \
  [--public-key <path/to/cliai-archive-keyring.asc>] \
  [--require-signature]
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

deb_field() {
  local deb_path="$1"
  local field_name="$2"
  dpkg-deb -f "${deb_path}" "${field_name}" 2>/dev/null
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
    --deb)
      require_value "$1" "${2:-}"
      DEB_FILES+=("$2")
      shift 2
      ;;
    --public-key)
      require_value "$1" "${2:-}"
      PUBLIC_KEY="$2"
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

if [[ -z "${REPO_ROOT}" || "${#DEB_FILES[@]}" -eq 0 ]]; then
  echo "missing required validation arguments" >&2
  usage
  exit 1
fi

if ! command -v dpkg-deb >/dev/null 2>&1; then
  echo "dpkg-deb is required to validate Debian package metadata." >&2
  exit 1
fi

release_dir="${REPO_ROOT}/dists/${DISTRIBUTION}"
release_file="${release_dir}/Release"

if [[ ! -f "${release_file}" ]]; then
  echo "missing Release file: ${release_file}" >&2
  exit 1
fi

for deb_path in "${DEB_FILES[@]}"; do
  if [[ ! -f "${deb_path}" ]]; then
    echo "missing deb package: ${deb_path}" >&2
    exit 1
  fi

  package_name="$(deb_field "${deb_path}" Package)"
  package_arch="$(deb_field "${deb_path}" Architecture)"

  if [[ -z "${package_name}" || -z "${package_arch}" ]]; then
    echo "failed to read package metadata from ${deb_path}" >&2
    exit 1
  fi

  packages_path="${release_dir}/main/binary-${package_arch}/Packages"
  packages_gz_path="${packages_path}.gz"

  if [[ ! -f "${packages_path}" ]]; then
    echo "missing Packages index: ${packages_path}" >&2
    exit 1
  fi

  if [[ ! -f "${packages_gz_path}" ]]; then
    echo "missing compressed Packages index: ${packages_gz_path}" >&2
    exit 1
  fi

  if ! grep -q "^Package: ${package_name}$" "${packages_path}"; then
    echo "package ${package_name} not found in ${packages_path}" >&2
    exit 1
  fi

  if ! gzip -dc "${packages_gz_path}" | grep -q "^Package: ${package_name}$"; then
    echo "package ${package_name} not found in ${packages_gz_path}" >&2
    exit 1
  fi

  deb_filename="$(basename "${deb_path}")"
  if ! find "${REPO_ROOT}/pool" -type f -name "${deb_filename}" | grep -q .; then
    echo "package ${deb_filename} not found in apt pool" >&2
    exit 1
  fi

  if ! grep -q "binary-${package_arch}/Packages" "${release_file}"; then
    echo "Release file does not reference binary-${package_arch}/Packages" >&2
    exit 1
  fi
done

release_gpg="${release_dir}/Release.gpg"
inrelease="${release_dir}/InRelease"
if [[ "${REQUIRE_SIGNATURE}" -eq 1 || -f "${release_gpg}" || -f "${inrelease}" ]]; then
  if [[ ! -f "${release_gpg}" || ! -f "${inrelease}" ]]; then
    echo "signed apt repo is incomplete; both Release.gpg and InRelease are required" >&2
    exit 1
  fi
fi

if [[ -n "${PUBLIC_KEY}" && ! -f "${PUBLIC_KEY}" ]]; then
  echo "missing public key file: ${PUBLIC_KEY}" >&2
  exit 1
fi

echo "apt release validation passed."
