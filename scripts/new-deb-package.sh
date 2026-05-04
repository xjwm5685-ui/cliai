#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

VERSION=""
ARCH=""
PACKAGE_NAME="cliai"
MAINTAINER="Sanqiu <noreply@users.noreply.github.com>"
DESCRIPTION="Hybrid command prediction CLI with local history, project context and optional PowerShell inline prediction."
BINARY_PATH=""
OUTPUT_DIR=""
STAGE_ONLY=0

usage() {
  cat >&2 <<'EOF'
usage: ./scripts/new-deb-package.sh \
  --version <version> \
  [--arch amd64|arm64] \
  [--binary <path>] \
  [--package-name <name>] \
  [--maintainer "<name <email>>"] \
  [--output <dir>] \
  [--stage-only]
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

detect_arch() {
  local machine
  machine="$(uname -m)"
  case "${machine}" in
    x86_64|amd64)
      echo "amd64"
      ;;
    aarch64|arm64)
      echo "arm64"
      ;;
    *)
      echo "${machine}"
      ;;
  esac
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      require_value "$1" "${2:-}"
      VERSION="$2"
      shift 2
      ;;
    --arch)
      require_value "$1" "${2:-}"
      ARCH="$2"
      shift 2
      ;;
    --binary)
      require_value "$1" "${2:-}"
      BINARY_PATH="$2"
      shift 2
      ;;
    --package-name)
      require_value "$1" "${2:-}"
      PACKAGE_NAME="$2"
      shift 2
      ;;
    --maintainer)
      require_value "$1" "${2:-}"
      MAINTAINER="$2"
      shift 2
      ;;
    --output)
      require_value "$1" "${2:-}"
      OUTPUT_DIR="$2"
      shift 2
      ;;
    --stage-only)
      STAGE_ONLY=1
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

if [[ -z "${VERSION}" ]]; then
  echo "missing required --version" >&2
  usage
  exit 1
fi

if [[ -z "${ARCH}" ]]; then
  ARCH="$(detect_arch)"
fi

case "${ARCH}" in
  amd64|arm64)
    ;;
  *)
    echo "unsupported deb architecture: ${ARCH} (expected amd64 or arm64)" >&2
    exit 1
    ;;
esac

if [[ -z "${BINARY_PATH}" ]]; then
  BINARY_PATH="${ROOT_DIR}/dist/linux-${ARCH}/cliai"
fi

if [[ ! -f "${BINARY_PATH}" ]]; then
  echo "binary not found: ${BINARY_PATH}" >&2
  exit 1
fi

if [[ -z "${OUTPUT_DIR}" ]]; then
  OUTPUT_DIR="${ROOT_DIR}/packaging/deb/${VERSION}/${ARCH}"
fi

PACKAGE_BASENAME="${PACKAGE_NAME}_${VERSION}_${ARCH}"
STAGE_ROOT="${OUTPUT_DIR}/stage/${PACKAGE_BASENAME}"
PACKAGE_OUTPUT="${OUTPUT_DIR}/${PACKAGE_BASENAME}.deb"
BUILD_STAGE_ROOT="${STAGE_ROOT}"
TEMP_BUILD_ROOT=""

rm -rf "${STAGE_ROOT}"
mkdir -p "${STAGE_ROOT}/DEBIAN" "${STAGE_ROOT}/usr/bin" "${STAGE_ROOT}/usr/share/${PACKAGE_NAME}" "${STAGE_ROOT}/usr/share/doc/${PACKAGE_NAME}"
chmod 755 "${STAGE_ROOT}" "${STAGE_ROOT}/DEBIAN" "${STAGE_ROOT}/usr" "${STAGE_ROOT}/usr/bin" "${STAGE_ROOT}/usr/share" "${STAGE_ROOT}/usr/share/${PACKAGE_NAME}" "${STAGE_ROOT}/usr/share/doc/${PACKAGE_NAME}"

install -m 755 "${BINARY_PATH}" "${STAGE_ROOT}/usr/bin/cliai"
install -m 755 "${ROOT_DIR}/scripts/install-unix.sh" "${STAGE_ROOT}/usr/share/${PACKAGE_NAME}/install-unix.sh"

if [[ -f "${ROOT_DIR}/LICENSE" ]]; then
  install -m 644 "${ROOT_DIR}/LICENSE" "${STAGE_ROOT}/usr/share/doc/${PACKAGE_NAME}/copyright"
fi

cat > "${STAGE_ROOT}/DEBIAN/control" <<EOF
Package: ${PACKAGE_NAME}
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: ${ARCH}
Maintainer: ${MAINTAINER}
Description: ${DESCRIPTION}
 cliai provides command prediction with local history, project context,
 feedback learning and optional PowerShell inline prediction support.
EOF

cat > "${STAGE_ROOT}/DEBIAN/postinst" <<EOF
#!/bin/sh
set -e
echo "Installed cliai. Run 'cliai version' to verify the installation."
if command -v pwsh >/dev/null 2>&1; then
  echo "PowerShell 7 detected. To enable inline prediction, run:"
  echo "  cliai shell install powershell"
fi
EOF
chmod 755 "${STAGE_ROOT}/DEBIAN/postinst"
chmod 644 "${STAGE_ROOT}/DEBIAN/control"

echo "deb staging directory prepared at ${STAGE_ROOT}"

if [[ "${STAGE_ONLY}" -eq 1 ]]; then
  exit 0
fi

if ! command -v dpkg-deb >/dev/null 2>&1; then
  echo "dpkg-deb is not available; rerun on Linux or use --stage-only." >&2
  exit 0
fi

if [[ "$(uname -s 2>/dev/null || echo unknown)" == Linux* ]]; then
  TEMP_BUILD_ROOT="$(mktemp -d)"
  trap 'rm -rf "${TEMP_BUILD_ROOT}"' EXIT
  BUILD_STAGE_ROOT="${TEMP_BUILD_ROOT}/${PACKAGE_BASENAME}"
  mkdir -p "$(dirname "${BUILD_STAGE_ROOT}")"
  cp -a "${STAGE_ROOT}" "${BUILD_STAGE_ROOT}"
  chmod 755 "${BUILD_STAGE_ROOT}" "${BUILD_STAGE_ROOT}/DEBIAN"
  chmod 644 "${BUILD_STAGE_ROOT}/DEBIAN/control"
  if [[ -f "${BUILD_STAGE_ROOT}/DEBIAN/postinst" ]]; then
    chmod 755 "${BUILD_STAGE_ROOT}/DEBIAN/postinst"
  fi
fi

mkdir -p "${OUTPUT_DIR}"
rm -f "${PACKAGE_OUTPUT}"
dpkg-deb --build --root-owner-group "${BUILD_STAGE_ROOT}" "${PACKAGE_OUTPUT}"
echo "deb package generated at ${PACKAGE_OUTPUT}"
