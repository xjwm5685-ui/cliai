#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'EOF'
usage: ./scripts/release-local.sh <version> [--require-apt-signature]
EOF
}

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

VERSION="$1"
shift

REQUIRE_APT_SIGNATURE=0
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

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

resolve_go_cmd() {
  if [[ -n "${GO_CMD:-}" ]]; then
    printf '%s\n' "${GO_CMD}"
    return 0
  fi
  if command -v go >/dev/null 2>&1; then
    command -v go
    return 0
  fi
  if command -v go.exe >/dev/null 2>&1; then
    command -v go.exe
    return 0
  fi
  for candidate in /d/bin/go.exe "/c/Program Files/Go/bin/go.exe"; do
    if [[ -x "${candidate}" ]]; then
      printf '%s\n' "${candidate}"
      return 0
    fi
  done
  return 1
}

to_native_path() {
  local path="$1"
  local drive rest

  if [[ "${GO_CMD}" != *.exe ]]; then
    printf '%s\n' "${path}"
    return 0
  fi

  if command -v cygpath >/dev/null 2>&1; then
    cygpath -w "${path}"
    return 0
  fi

  if [[ "${path}" =~ ^/mnt/([a-zA-Z])/(.*)$ ]]; then
    drive="${BASH_REMATCH[1]}"
    rest="${BASH_REMATCH[2]//\//\\}"
    printf '%s\n' "${drive^}:\\${rest}"
    return 0
  fi

  if [[ "${path}" =~ ^/([a-zA-Z])/(.*)$ ]]; then
    drive="${BASH_REMATCH[1]}"
    rest="${BASH_REMATCH[2]//\//\\}"
    printf '%s\n' "${drive^}:\\${rest}"
    return 0
  fi

  printf '%s\n' "${path}"
}

if ! GO_CMD="$(resolve_go_cmd)"; then
  echo "go command not found. Set GO_CMD or ensure go/go.exe is available in PATH." >&2
  exit 1
fi

COMMIT="$(git -C "${ROOT_DIR}" rev-parse --short HEAD 2>/dev/null || echo local)"
HOST_GOOS="$("${GO_CMD}" env GOOS)"
HOST_GOARCH="$("${GO_CMD}" env GOARCH)"
HOST_UNAME="$(uname -s 2>/dev/null || echo unknown)"

mkdir -p "${DIST_DIR}"

sha256_value() {
  local path="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${path}" | awk '{print $1}'
  else
    shasum -a 256 "${path}" | awk '{print $1}'
  fi
}

write_sha256_file() {
  local path="$1"
  printf "%s  %s\n" "$(sha256_value "${path}")" "$(basename "${path}")" > "${path}.sha256"
}

build_unix_release() {
  local goos="$1"
  local goarch="$2"
  local artifact="$3"
  local asset_root="${DIST_DIR}/${goos}-${goarch}"
  local artifact_path="${DIST_DIR}/${artifact}"
  local build_output

  rm -rf "${asset_root}"
  mkdir -p "${asset_root}/scripts"
  rm -f "${artifact_path}" "${artifact_path}.sha256"
  build_output="$(to_native_path "${asset_root}/cliai")"

  GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 \
    "${GO_CMD}" build -trimpath \
      -ldflags "-s -w -X github.com/sanqiu/cliai/internal/app.Version=${VERSION} -X github.com/sanqiu/cliai/internal/app.Commit=${COMMIT} -X github.com/sanqiu/cliai/internal/app.BuildDate=${BUILD_DATE}" \
      -o "${build_output}" .

  cp "${ROOT_DIR}/scripts/install-unix.sh" "${asset_root}/scripts/install-unix.sh"
  chmod 755 "${asset_root}/cliai" "${asset_root}/scripts/install-unix.sh"

  tar -C "${asset_root}" -czf "${artifact_path}" .
  write_sha256_file "${artifact_path}"

  if [[ "${goos}" == "${HOST_GOOS}" && "${goarch}" == "${HOST_GOARCH}" ]]; then
    "${asset_root}/cliai" selftest --json >/dev/null
  fi
}

pushd "${ROOT_DIR}" >/dev/null
trap 'popd >/dev/null' EXIT

check_env_args=()
if [[ "${REQUIRE_APT_SIGNATURE}" -eq 1 ]]; then
  check_env_args+=(--require-apt-signature)
fi
./scripts/check-release-env.sh "${check_env_args[@]}"

"${GO_CMD}" test ./...

build_unix_release linux amd64 cliai_Linux_x86_64.tar.gz
build_unix_release linux arm64 cliai_Linux_ARM64.tar.gz
build_unix_release darwin amd64 cliai_macOS_x86_64.tar.gz
build_unix_release darwin arm64 cliai_macOS_ARM64.tar.gz

if [[ "${HOST_UNAME}" == Linux* ]] && command -v dpkg-deb >/dev/null 2>&1; then
  ./scripts/new-deb-package.sh --version "${VERSION}" --arch amd64 --binary "./dist/linux-amd64/cliai" --output "./dist/deb/amd64"
  ./scripts/new-deb-package.sh --version "${VERSION}" --arch arm64 --binary "./dist/linux-arm64/cliai" --output "./dist/deb/arm64"

  ./scripts/new-apt-repo.sh \
    --repo-root "./dist/apt-repo/${VERSION}" \
    --deb "./dist/deb/amd64/cliai_${VERSION}_amd64.deb" \
    --deb "./dist/deb/arm64/cliai_${VERSION}_arm64.deb"

  sign_args=(--repo-root "./dist/apt-repo/${VERSION}")
  if [[ "${REQUIRE_APT_SIGNATURE}" -eq 1 ]]; then
    sign_args+=(--require-signature)
  fi
  ./scripts/sign-apt-repo.sh "${sign_args[@]}"

  if [[ -n "${CLIAI_APT_GPG_KEY_FILE:-}" || -n "${CLIAI_APT_GPG_KEY_ARMORED:-}" ]]; then
    ./scripts/export-apt-public-key.sh --output "./dist/cliai-archive-keyring.asc"
  fi

  repo_archive="./dist/cliai_apt_repo_${VERSION}.tar.gz"
  rm -f "${repo_archive}" "${repo_archive}.sha256"
  tar -C "./dist/apt-repo/${VERSION}" -czf "${repo_archive}" .

  for artifact in \
    "./dist/deb/amd64/cliai_${VERSION}_amd64.deb" \
    "./dist/deb/arm64/cliai_${VERSION}_arm64.deb" \
    "${repo_archive}" \
    "./dist/cliai-archive-keyring.asc"; do
    if [[ -f "${artifact}" ]]; then
      write_sha256_file "${artifact}"
    fi
  done

  validate_args=()
  if [[ -f "./dist/apt-repo/${VERSION}/dists/stable/Release.gpg" || -f "./dist/apt-repo/${VERSION}/dists/stable/InRelease" ]]; then
    validate_args+=(--require-signature)
  fi
  if [[ -f "./dist/cliai-archive-keyring.asc" ]]; then
    validate_args+=(--public-key "./dist/cliai-archive-keyring.asc")
  fi

  ./scripts/validate-apt-release.sh \
    --repo-root "./dist/apt-repo/${VERSION}" \
    --deb "./dist/deb/amd64/cliai_${VERSION}_amd64.deb" \
    --deb "./dist/deb/arm64/cliai_${VERSION}_arm64.deb" \
    "${validate_args[@]}"
else
  echo "Linux dpkg-deb environment not available. Skipping local Debian and apt repository artifacts."
fi

echo "Local Unix release artifacts are ready in ${DIST_DIR}"
