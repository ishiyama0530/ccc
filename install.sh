#!/usr/bin/env bash

set -euo pipefail

REPO="${CCC_INSTALL_REPO:-ishiyama0530/ccc}"
API_BASE="${CCC_INSTALL_GITHUB_API_BASE:-https://api.github.com/repos/${REPO}/releases}"
DOWNLOAD_BASE="${CCC_INSTALL_GITHUB_DOWNLOAD_BASE:-https://github.com/${REPO}/releases/download}"
INSTALL_DIR="${CCC_INSTALL_DIR:-${HOME}/.local/bin}"
REQUESTED_VERSION="${CCC_INSTALL_VERSION:-}"
REQUESTED_OS="${CCC_INSTALL_OS:-}"
REQUESTED_ARCH="${CCC_INSTALL_ARCH:-}"

main() {
  require_command tar
  require_command mktemp

  local os arch version archive_name checksum_name tmpdir archive_path checksums_path binary_path
  os="$(detect_os)"
  arch="$(detect_arch)"
  version="$(resolve_version)"
  archive_name="ccc_${os}_${arch}.tar.gz"
  checksum_name="checksums.txt"
  tmpdir="$(mktemp -d)"
  archive_path="${tmpdir}/${archive_name}"
  checksums_path="${tmpdir}/${checksum_name}"
  binary_path="${tmpdir}/ccc"

  trap 'rm -rf -- '"'"${tmpdir}"'"'' EXIT

  echo "Installing ccc ${version} for ${os}/${arch}..."
  download_to "${DOWNLOAD_BASE}/${version}/${archive_name}" "${archive_path}"
  download_to "${DOWNLOAD_BASE}/${version}/${checksum_name}" "${checksums_path}"

  verify_checksum "${archive_path}" "${checksums_path}" "${archive_name}"

  tar -xzf "${archive_path}" -C "${tmpdir}" ccc
  mkdir -p "${INSTALL_DIR}"
  install_binary "${binary_path}" "${INSTALL_DIR}/ccc"

  echo "ccc installed to ${INSTALL_DIR}/ccc"
  if ! path_contains "${INSTALL_DIR}"; then
    echo "Add ${INSTALL_DIR} to your PATH to run ccc from a new shell." >&2
  fi
}

detect_os() {
  if [[ -n "${REQUESTED_OS}" ]]; then
    echo "${REQUESTED_OS}"
    return
  fi

  case "$(uname -s)" in
    Darwin) echo "darwin" ;;
    Linux) echo "linux" ;;
    *)
      echo "unsupported operating system: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  if [[ -n "${REQUESTED_ARCH}" ]]; then
    echo "${REQUESTED_ARCH}"
    return
  fi

  case "$(uname -m)" in
    x86_64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

resolve_version() {
  if [[ -n "${REQUESTED_VERSION}" ]]; then
    echo "${REQUESTED_VERSION}"
    return
  fi

  local json version
  json="$(download_text "${API_BASE}/latest")"
  version="$(
    printf '%s' "${json}" \
      | tr -d '\n' \
      | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p'
  )"

  if [[ -z "${version}" ]]; then
    echo "failed to determine the latest ccc release" >&2
    exit 1
  fi

  echo "${version}"
}

verify_checksum() {
  local archive_path checksums_path archive_name expected actual
  archive_path="$1"
  checksums_path="$2"
  archive_name="$3"
  expected="$(awk -v file="${archive_name}" '$2 == file { print $1 }' "${checksums_path}")"

  if [[ -z "${expected}" ]]; then
    echo "checksum for ${archive_name} not found" >&2
    exit 1
  fi

  actual="$(sha256_file "${archive_path}")"
  if [[ "${expected}" != "${actual}" ]]; then
    echo "checksum mismatch for ${archive_name}" >&2
    exit 1
  fi
}

sha256_file() {
  local file
  file="$1"

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${file}" | awk '{ print $1 }'
    return
  fi

  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${file}" | awk '{ print $1 }'
    return
  fi

  echo "missing checksum tool: need sha256sum or shasum" >&2
  exit 1
}

download_to() {
  local url output
  url="$1"
  output="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "${url}" -o "${output}"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -qO "${output}" "${url}"
    return
  fi

  echo "missing downloader: need curl or wget" >&2
  exit 1
}

download_text() {
  local url
  url="$1"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "${url}"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -qO- "${url}"
    return
  fi

  echo "missing downloader: need curl or wget" >&2
  exit 1
}

install_binary() {
  local source target
  source="$1"
  target="$2"

  if command -v install >/dev/null 2>&1; then
    install -m 755 "${source}" "${target}"
    return
  fi

  cp "${source}" "${target}"
  chmod 755 "${target}"
}

path_contains() {
  local target dir
  target="$1"

  IFS=':' read -r -a dirs <<<"${PATH:-}"
  for dir in "${dirs[@]}"; do
    if [[ "${dir}" == "${target}" ]]; then
      return 0
    fi
  done

  return 1
}

require_command() {
  local name
  name="$1"

  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "missing required command: ${name}" >&2
    exit 1
  fi
}

main "$@"
