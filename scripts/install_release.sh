#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:-EdOoO21/metadata-parser}"
BIN_NAME="${BIN_NAME:-catalog}"
COMMAND_NAME="${COMMAND_NAME:-catalog}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${1:-latest}"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT INT TERM

detect_os() {
  case "$(uname -s)" in
    Linux) printf 'linux' ;;
    Darwin) printf 'darwin' ;;
    *)
      echo "unsupported OS: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64' ;;
    aarch64|arm64) printf 'arm64' ;;
    *)
      echo "unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

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

detect_shell_rc() {
  local shell_name
  shell_name="$(basename "${SHELL:-}")"

  case "${shell_name}" in
    zsh) printf '%s/.zshrc' "${HOME}" ;;
    bash) printf '%s/.bashrc' "${HOME}" ;;
    *)
      if [[ -f "${HOME}/.bashrc" ]]; then
        printf '%s/.bashrc' "${HOME}"
      else
        printf '%s/.profile' "${HOME}"
      fi
      ;;
  esac
}

download_release_archive() {
  local os arch base_url asset_name output_path
  os="$(detect_os)"
  arch="$(detect_arch)"
  base_url="$(release_base_url)"
  asset_name="metadata-parser_${os}_${arch}.tar.gz"
  output_path="${TMP_DIR}/${asset_name}"

  echo "Downloading ${base_url}/${asset_name}" >&2
  curl -fsSL "${base_url}/${asset_name}" -o "${output_path}"
  printf '%s' "${output_path}"
}

extract_binary() {
  local archive_path="$1"
  tar -xzf "${archive_path}" -C "${TMP_DIR}"

  local extracted_binary
  extracted_binary="$(find "${TMP_DIR}" -maxdepth 3 -type f -name "${BIN_NAME}" | head -n 1)"
  if [[ -z "${extracted_binary}" ]]; then
    echo "binary ${BIN_NAME} not found in release archive" >&2
    exit 1
  fi

  printf '%s' "${extracted_binary}"
}

install_binary() {
  local binary_path="$1"
  mkdir -p "${INSTALL_DIR}"
  install -m 0755 "${binary_path}" "${INSTALL_DIR}/${BIN_NAME}"
}

update_shell_rc() {
  local rc_file tmp_rc
  rc_file="$(detect_shell_rc)"
  mkdir -p "$(dirname "${rc_file}")"
  touch "${rc_file}"
  tmp_rc="${TMP_DIR}/shellrc"

  awk '
    BEGIN { skip = 0 }
    /^# >>> metadata-parser >>>$/ { skip = 1; next }
    /^# <<< metadata-parser <<<$/{ skip = 0; next }
    skip == 0 { print }
  ' "${rc_file}" > "${tmp_rc}"

  cat >> "${tmp_rc}" <<EOF
# >>> metadata-parser >>>
export PATH="\$HOME/.local/bin:\$PATH"
alias ${COMMAND_NAME}="\$HOME/.local/bin/${BIN_NAME}"
# <<< metadata-parser <<<
EOF

  mv "${tmp_rc}" "${rc_file}"
  printf '%s' "${rc_file}"
}

main() {
  local archive_path binary_path rc_file

  archive_path="$(download_release_archive)"
  binary_path="$(extract_binary "${archive_path}")"
  install_binary "${binary_path}"
  rc_file="$(update_shell_rc)"

  echo
  echo "Installed ${BIN_NAME} to ${INSTALL_DIR}/${BIN_NAME}"
  echo "Shell configuration updated: ${rc_file}"
  echo "Run one of the following commands in a new terminal session:"
  echo "  ${COMMAND_NAME} --help"
  echo "or reload your shell:"
  echo "  source ${rc_file}"
}

main "$@"
