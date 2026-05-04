#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

REPO_ROOT=""
DISTRIBUTION="stable"
COMPONENT="main"
ORIGIN="Sanqiu"
LABEL="cliai"
DESCRIPTION="cliai apt repository"
DEB_FILES=()

usage() {
  cat >&2 <<'EOF'
usage: ./scripts/new-apt-repo.sh \
  --repo-root <dir> \
  [--distribution stable] \
  [--component main] \
  [--origin Sanqiu] \
  [--label cliai] \
  [--description "cliai apt repository"] \
  --deb <path/to/package.deb> [--deb <path/to/package.deb> ...]
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

checksum_value() {
  local algorithm="$1"
  local path="$2"
  case "${algorithm}" in
    md5)
      md5sum "${path}" | awk '{print $1}'
      ;;
    sha256)
      sha256sum "${path}" | awk '{print $1}'
      ;;
    *)
      echo "unsupported checksum algorithm: ${algorithm}" >&2
      exit 1
      ;;
  esac
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
    --component)
      require_value "$1" "${2:-}"
      COMPONENT="$2"
      shift 2
      ;;
    --origin)
      require_value "$1" "${2:-}"
      ORIGIN="$2"
      shift 2
      ;;
    --label)
      require_value "$1" "${2:-}"
      LABEL="$2"
      shift 2
      ;;
    --description)
      require_value "$1" "${2:-}"
      DESCRIPTION="$2"
      shift 2
      ;;
    --deb)
      require_value "$1" "${2:-}"
      DEB_FILES+=("$2")
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

if [[ -z "${REPO_ROOT}" || "${#DEB_FILES[@]}" -eq 0 ]]; then
  echo "missing required repo arguments" >&2
  usage
  exit 1
fi

if ! command -v dpkg-deb >/dev/null 2>&1; then
  echo "dpkg-deb is required to inspect .deb metadata." >&2
  exit 1
fi

if ! command -v sha256sum >/dev/null 2>&1 || ! command -v md5sum >/dev/null 2>&1 || ! command -v gzip >/dev/null 2>&1; then
  echo "md5sum, sha256sum and gzip are required." >&2
  exit 1
fi

mkdir -p "${REPO_ROOT}"

declare -A PACKAGE_ENTRIES

for deb_path in "${DEB_FILES[@]}"; do
  if [[ ! -f "${deb_path}" ]]; then
    echo "deb file not found: ${deb_path}" >&2
    exit 1
  fi

  package_name="$(deb_field "${deb_path}" Package)"
  package_version="$(deb_field "${deb_path}" Version)"
  package_arch="$(deb_field "${deb_path}" Architecture)"
  package_section="$(deb_field "${deb_path}" Section)"
  package_priority="$(deb_field "${deb_path}" Priority)"
  package_maintainer="$(deb_field "${deb_path}" Maintainer)"
  package_description="$(deb_field "${deb_path}" Description)"

  if [[ -z "${package_name}" || -z "${package_version}" || -z "${package_arch}" ]]; then
    echo "failed to read package metadata from ${deb_path}" >&2
    exit 1
  fi

  package_prefix="${package_name:0:1}"
  pool_dir="${REPO_ROOT}/pool/${COMPONENT}/${package_prefix}/${package_name}"
  mkdir -p "${pool_dir}"
  deb_filename="$(basename "${deb_path}")"
  pool_target="${pool_dir}/${deb_filename}"
  cp "${deb_path}" "${pool_target}"

  relative_filename="${pool_target#${REPO_ROOT}/}"
  package_size="$(wc -c < "${pool_target}" | tr -d ' ')"
  package_md5="$(checksum_value md5 "${pool_target}")"
  package_sha256="$(checksum_value sha256 "${pool_target}")"

  entry=$(cat <<EOF
Package: ${package_name}
Version: ${package_version}
Architecture: ${package_arch}
Maintainer: ${package_maintainer}
Section: ${package_section:-utils}
Priority: ${package_priority:-optional}
Filename: ${relative_filename}
Size: ${package_size}
MD5sum: ${package_md5}
SHA256: ${package_sha256}
Description: ${package_description}

EOF
)

  PACKAGE_ENTRIES["${package_arch}"]+="${entry}"
done

release_temp="$(mktemp)"
trap 'rm -f "${release_temp}"' EXIT

for arch in "${!PACKAGE_ENTRIES[@]}"; do
  binary_dir="${REPO_ROOT}/dists/${DISTRIBUTION}/${COMPONENT}/binary-${arch}"
  mkdir -p "${binary_dir}"

  printf "%s" "${PACKAGE_ENTRIES[${arch}]}" > "${binary_dir}/Packages"
  gzip -n -9 -c "${binary_dir}/Packages" > "${binary_dir}/Packages.gz"
done

release_file="${REPO_ROOT}/dists/${DISTRIBUTION}/Release"
mkdir -p "$(dirname "${release_file}")"

{
  echo "Origin: ${ORIGIN}"
  echo "Label: ${LABEL}"
  echo "Suite: ${DISTRIBUTION}"
  echo "Codename: ${DISTRIBUTION}"
  echo "Architectures: $(printf '%s\n' "${!PACKAGE_ENTRIES[@]}" | sort | paste -sd ' ' -)"
  echo "Components: ${COMPONENT}"
  echo "Description: ${DESCRIPTION}"
  echo "Date: $(LC_ALL=C date -Ru)"
  echo "MD5Sum:"
} > "${release_temp}"

while IFS= read -r -d '' file_path; do
  relative_path="${file_path#${REPO_ROOT}/dists/${DISTRIBUTION}/}"
  file_size="$(wc -c < "${file_path}" | tr -d ' ')"
  file_md5="$(checksum_value md5 "${file_path}")"
  printf " %s %16s %s\n" "${file_md5}" "${file_size}" "${relative_path}" >> "${release_temp}"
done < <(find "${REPO_ROOT}/dists/${DISTRIBUTION}" -type f \( -name 'Packages' -o -name 'Packages.gz' \) -print0 | sort -z)

echo "SHA256:" >> "${release_temp}"
while IFS= read -r -d '' file_path; do
  relative_path="${file_path#${REPO_ROOT}/dists/${DISTRIBUTION}/}"
  file_size="$(wc -c < "${file_path}" | tr -d ' ')"
  file_sha256="$(checksum_value sha256 "${file_path}")"
  printf " %s %16s %s\n" "${file_sha256}" "${file_size}" "${relative_path}" >> "${release_temp}"
done < <(find "${REPO_ROOT}/dists/${DISTRIBUTION}" -type f \( -name 'Packages' -o -name 'Packages.gz' \) -print0 | sort -z)

mv "${release_temp}" "${release_file}"
trap - EXIT

echo "apt repository metadata generated at ${REPO_ROOT}"
