#!/usr/bin/env bash
set -euo pipefail

REQUIRE_APT_SIGNATURE=0

usage() {
  cat >&2 <<'EOF'
usage: ./scripts/check-release-env.sh [--require-apt-signature]
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --require-apt-signature)
      REQUIRE_APT_SIGNATURE=1
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

errors=()
warnings=()

if ! command -v git >/dev/null 2>&1; then
  errors+=("Git is not installed or not in PATH.")
fi

if ! command -v bash >/dev/null 2>&1; then
  errors+=("bash is not installed or not in PATH.")
fi

if ! command -v gzip >/dev/null 2>&1; then
  errors+=("gzip is not installed or not in PATH.")
fi

if ! command -v tar >/dev/null 2>&1; then
  errors+=("tar is not installed or not in PATH.")
fi

if ! command -v gpg >/dev/null 2>&1; then
  warnings+=("gpg is not installed; apt signing and public key export will be skipped.")
fi

if [[ "$(uname -s 2>/dev/null || echo unknown)" == Linux* ]]; then
  if ! command -v dpkg-deb >/dev/null 2>&1; then
    warnings+=("dpkg-deb is not installed; Debian package and apt repo artifacts will be skipped.")
  fi
else
  warnings+=("Non-Linux environment detected; Debian package and apt repo artifacts will be skipped.")
fi

if [[ "${REQUIRE_APT_SIGNATURE}" -eq 1 ]]; then
  if ! command -v gpg >/dev/null 2>&1; then
    errors+=("gpg is required when --require-apt-signature is used.")
  fi
  if [[ -z "${CLIAI_APT_GPG_KEY_FILE:-}" && -z "${CLIAI_APT_GPG_KEY_ARMORED:-}" ]]; then
    errors+=("Missing CLIAI_APT_GPG_KEY_FILE or CLIAI_APT_GPG_KEY_ARMORED.")
  fi
fi

for warning in "${warnings[@]}"; do
  printf 'Warning: %s\n' "${warning}" >&2
done

if [[ "${#errors[@]}" -gt 0 ]]; then
  for error in "${errors[@]}"; do
    printf 'Error: %s\n' "${error}" >&2
  done
  exit 1
fi

echo "Unix release environment looks good."
