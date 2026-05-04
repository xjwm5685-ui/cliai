#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

VERSION=""
FORMULA_NAME="cliai"
CLASS_NAME="Cliai"
OUTPUT_PATH=""
DESC_TEXT="Hybrid command prediction CLI with PowerShell-first inline prediction support"
HOMEPAGE_URL="https://github.com/xjwm5685-ui/cliai"
LICENSE_NAME="MIT"

DARWIN_AMD64_URL=""
DARWIN_AMD64_SHA256=""
DARWIN_ARM64_URL=""
DARWIN_ARM64_SHA256=""
LINUX_AMD64_URL=""
LINUX_AMD64_SHA256=""
LINUX_ARM64_URL=""
LINUX_ARM64_SHA256=""

usage() {
  cat >&2 <<'EOF'
usage: ./scripts/new-homebrew-formula.sh \
  --version <version> \
  --darwin-amd64-url <url> \
  --darwin-amd64-sha256 <sha256> \
  --darwin-arm64-url <url> \
  --darwin-arm64-sha256 <sha256> \
  [--linux-amd64-url <url>] \
  [--linux-amd64-sha256 <sha256>] \
  [--linux-arm64-url <url>] \
  [--linux-arm64-sha256 <sha256>] \
  [--formula-name <name>] \
  [--class-name <ruby class>] \
  [--output <path>]
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
    --version)
      require_value "$1" "${2:-}"
      VERSION="$2"
      shift 2
      ;;
    --formula-name)
      require_value "$1" "${2:-}"
      FORMULA_NAME="$2"
      shift 2
      ;;
    --class-name)
      require_value "$1" "${2:-}"
      CLASS_NAME="$2"
      shift 2
      ;;
    --output)
      require_value "$1" "${2:-}"
      OUTPUT_PATH="$2"
      shift 2
      ;;
    --darwin-amd64-url)
      require_value "$1" "${2:-}"
      DARWIN_AMD64_URL="$2"
      shift 2
      ;;
    --darwin-amd64-sha256)
      require_value "$1" "${2:-}"
      DARWIN_AMD64_SHA256="$2"
      shift 2
      ;;
    --darwin-arm64-url)
      require_value "$1" "${2:-}"
      DARWIN_ARM64_URL="$2"
      shift 2
      ;;
    --darwin-arm64-sha256)
      require_value "$1" "${2:-}"
      DARWIN_ARM64_SHA256="$2"
      shift 2
      ;;
    --linux-amd64-url)
      require_value "$1" "${2:-}"
      LINUX_AMD64_URL="$2"
      shift 2
      ;;
    --linux-amd64-sha256)
      require_value "$1" "${2:-}"
      LINUX_AMD64_SHA256="$2"
      shift 2
      ;;
    --linux-arm64-url)
      require_value "$1" "${2:-}"
      LINUX_ARM64_URL="$2"
      shift 2
      ;;
    --linux-arm64-sha256)
      require_value "$1" "${2:-}"
      LINUX_ARM64_SHA256="$2"
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

if [[ -z "${VERSION}" || -z "${DARWIN_AMD64_URL}" || -z "${DARWIN_AMD64_SHA256}" || -z "${DARWIN_ARM64_URL}" || -z "${DARWIN_ARM64_SHA256}" ]]; then
  echo "missing required macOS release metadata" >&2
  usage
  exit 1
fi

if [[ -z "${OUTPUT_PATH}" ]]; then
  OUTPUT_PATH="${ROOT_DIR}/packaging/homebrew/${VERSION}/${FORMULA_NAME}.rb"
fi

mkdir -p "$(dirname "${OUTPUT_PATH}")"

cat > "${OUTPUT_PATH}" <<EOF
class ${CLASS_NAME} < Formula
  desc "${DESC_TEXT}"
  homepage "${HOMEPAGE_URL}"
  version "${VERSION}"
  license "${LICENSE_NAME}"
EOF

cat >> "${OUTPUT_PATH}" <<EOF

  on_macos do
    on_intel do
      url "${DARWIN_AMD64_URL}"
      sha256 "${DARWIN_AMD64_SHA256}"
    end
    on_arm do
      url "${DARWIN_ARM64_URL}"
      sha256 "${DARWIN_ARM64_SHA256}"
    end
  end
EOF

if [[ -n "${LINUX_AMD64_URL}" && -n "${LINUX_AMD64_SHA256}" && -n "${LINUX_ARM64_URL}" && -n "${LINUX_ARM64_SHA256}" ]]; then
  cat >> "${OUTPUT_PATH}" <<EOF

  on_linux do
    on_intel do
      url "${LINUX_AMD64_URL}"
      sha256 "${LINUX_AMD64_SHA256}"
    end
    on_arm do
      url "${LINUX_ARM64_URL}"
      sha256 "${LINUX_ARM64_SHA256}"
    end
  end
EOF
fi

cat >> "${OUTPUT_PATH}" <<'EOF'

  def install
    bin.install "cliai"
    pkgshare.install "scripts/install-unix.sh"
  end

  def caveats
    <<~EOS
      The core CLI is installed as:
        cliai

      Optional PowerShell inline prediction:
        cliai shell install powershell

      The bundled Unix install helper is stored at:
        #{pkgshare}/install-unix.sh
    EOS
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/cliai version")
  end
end
EOF

echo "Homebrew formula generated at ${OUTPUT_PATH}"
