#!/bin/sh
set -e

REPO="zeropsio/rendspec"
INSTALL_DIR="/usr/local/bin"

# Allow overriding install dir
if [ -n "$RENDSPEC_INSTALL_DIR" ]; then
  INSTALL_DIR="$RENDSPEC_INSTALL_DIR"
elif [ ! -w "$INSTALL_DIR" ] && [ "$(id -u)" != "0" ]; then
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi

detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *) echo "unsupported" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64)  echo "arm64" ;;
    *) echo "unsupported" ;;
  esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"

if [ "$OS" = "unsupported" ] || [ "$ARCH" = "unsupported" ]; then
  echo "Error: unsupported platform $(uname -s)/$(uname -m)" >&2
  exit 1
fi

# Determine version
if [ -n "$1" ]; then
  VERSION="$1"
else
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')"
fi

if [ -z "$VERSION" ]; then
  echo "Error: could not determine latest version" >&2
  exit 1
fi

echo "Installing rendspec ${VERSION} (${OS}/${ARCH})..."

if [ "$OS" = "windows" ]; then
  URL="https://github.com/${REPO}/releases/download/${VERSION}/rendspec-${VERSION}-${OS}-${ARCH}.zip"
  TMP="$(mktemp -d)"
  curl -fsSL "$URL" -o "${TMP}/rendspec.zip"
  unzip -qo "${TMP}/rendspec.zip" -d "${TMP}"
  mv "${TMP}/rendspec.exe" "${INSTALL_DIR}/rendspec.exe"
  mv "${TMP}/rendspec-mcp.exe" "${INSTALL_DIR}/rendspec-mcp.exe"
  rm -rf "$TMP"
else
  URL="https://github.com/${REPO}/releases/download/${VERSION}/rendspec-${VERSION}-${OS}-${ARCH}.tar.gz"
  TMP="$(mktemp -d)"
  curl -fsSL "$URL" | tar -xz -C "$TMP"
  mv "${TMP}/rendspec" "${INSTALL_DIR}/rendspec"
  mv "${TMP}/rendspec-mcp" "${INSTALL_DIR}/rendspec-mcp"
  chmod +x "${INSTALL_DIR}/rendspec" "${INSTALL_DIR}/rendspec-mcp"
  rm -rf "$TMP"
fi

echo "Installed to ${INSTALL_DIR}:"
echo "  rendspec"
echo "  rendspec-mcp"

# Check PATH
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo ""
    echo "Note: ${INSTALL_DIR} is not in your PATH."
    echo "Add it with:  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac
