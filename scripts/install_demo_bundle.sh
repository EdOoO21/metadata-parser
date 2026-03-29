#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:-EdOoO21/metadata-parser}"
BUNDLE_NAME="${BUNDLE_NAME:-metadata-parser-demo.tar.gz}"
TARGET_DIR="${TARGET_DIR:-$HOME/metadata-parser-demo}"
VERSION="${1:-latest}"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT INT TERM

release_base_url() {
  if [[ "${VERSION}" == "latest" ]]; then
    printf 'https://github.com/%s/releases/latest/download' "${REPO}"
    return 0
  fi

  local tag="${VERSION}"
  if [[ "${tag}" != v* ]]; then
    tag="v${tag}"
  fi

  printf 'https://github.com/%s/releases/download/%s' "${REPO}" "${tag}"
}

download_bundle() {
  local base_url output_path
  base_url="$(release_base_url)"
  output_path="${TMP_DIR}/${BUNDLE_NAME}"

  echo "Downloading ${base_url}/${BUNDLE_NAME}" >&2
  curl -fsSL "${base_url}/${BUNDLE_NAME}" -o "${output_path}"
  printf '%s' "${output_path}"
}

install_bundle() {
  local archive_path="$1"
  rm -rf "${TARGET_DIR}"
  mkdir -p "${TARGET_DIR}"
  tar -xzf "${archive_path}" -C "${TARGET_DIR}"
}

main() {
  local archive_path
  archive_path="$(download_bundle)"
  install_bundle "${archive_path}"

  echo
  echo "Demo bundle installed to ${TARGET_DIR}"
  echo "Next steps:"
  echo "  cd ${TARGET_DIR}"
  echo "  edit .env if you want to change ports, passwords or DSN values"
  echo "  make app-up"
  echo "  catalog run --config ./testcases/files/1.yaml"
}

main "$@"
