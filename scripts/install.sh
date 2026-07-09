#!/usr/bin/env bash
set -euo pipefail

REPO="kinncj/MAPLE"
RTK_REPO="rtk-ai/rtk"
INSTALL_DIR="${MAPLE_INSTALL_DIR:-$HOME/.tools/maple/bin}"
SKIP_RTK="${SKIP_RTK:-}"
SKIP_HERDR="${SKIP_HERDR:-}"

# ── Parse arguments ────────────────────────────────────────────────────────────
VERSION="${MAPLE_VERSION:-}"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --version|-v)
            VERSION="$2"
            shift 2
            ;;
        --version=*|-v=*)
            VERSION="${1#*=}"
            shift
            ;;
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --skip-rtk)
            SKIP_RTK=1
            shift
            ;;
        --skip-herdr)
            SKIP_HERDR=1
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: install.sh [--version vX.Y.Z] [--install-dir PATH] [--skip-rtk] [--skip-herdr]"
            exit 1
            ;;
    esac
done

# ── Platform detection ─────────────────────────────────────────────────────────
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)        ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac
case "$OS" in
    linux|darwin) ;;
    *) echo "Unsupported OS: $OS — use install.ps1 on Windows"; exit 1 ;;
esac

# ── Resolve maple version ──────────────────────────────────────────────────────
if [ -z "$VERSION" ]; then
    # Fetch up to 100 releases (GitHub API max per page), sort by semver, pick highest.
    # Avoids /releases/latest which sorts by created_at, not version number.
    VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases?per_page=100" \
        | grep '"tag_name"' \
        | cut -d'"' -f4 \
        | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' \
        | sort -V \
        | tail -1)
fi
[ -z "$VERSION" ] && { echo "Could not determine latest version. Set MAPLE_VERSION=vX.Y.Z or pass --version vX.Y.Z"; exit 1; }

# ── Download and install maple ─────────────────────────────────────────────────
ARCHIVE="maple-${OS}-${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

echo "Installing maple ${VERSION} (${OS}/${ARCH})"
echo "  → ${INSTALL_DIR}/maple"
echo ""

mkdir -p "$INSTALL_DIR"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# A freshly-cut release can appear in the API a few seconds before its binaries finish
# uploading. Retry a little so a well-timed install doesn't die on a transient 404.
attempts=0
until curl -fsSL "$URL" -o "$TMP/$ARCHIVE"; do
    attempts=$((attempts + 1))
    if [ "$attempts" -ge 5 ]; then
        echo "Could not download after $attempts tries:"
        echo "  $URL"
        echo "The release may still be publishing its binaries — wait a minute and re-run,"
        echo "or pin a known-good version with --version vX.Y.Z."
        exit 1
    fi
    echo "  binary not ready yet (attempt $attempts/5) — retrying in 5s…"
    sleep 5
done
tar -xzf "$TMP/$ARCHIVE" -C "$TMP"
mv "$TMP/maple" "$INSTALL_DIR/maple"
chmod +x "$INSTALL_DIR/maple"

echo "✓ Installed maple ${VERSION}"
echo ""

# ── Install RTK token optimizer ────────────────────────────────────────────────
# RTK intercepts Bash tool calls and compresses output 60-90% before it hits the
# LLM context window. Single Rust binary, zero runtime dependencies.
# Skip with: --skip-rtk or SKIP_RTK=1
install_rtk() {
    if command -v rtk >/dev/null 2>&1; then
        echo "✓ rtk already installed ($(rtk --version 2>/dev/null || echo unknown))"
        return 0
    fi

    echo "Installing rtk token optimizer…"

    if _rtk_install_from_release; then
        return 0
    fi

    case "$OS" in
        linux|darwin) ;;
        *)
            echo "~ rtk: install failed — install manually: https://github.com/rtk-ai/rtk"
            return 0
            ;;
    esac

    echo "~ falling back to upstream rtk install script…"
    if curl -fsSL "https://raw.githubusercontent.com/rtk-ai/rtk/refs/heads/master/install.sh" | sh; then
        if command -v rtk >/dev/null 2>&1; then
            echo "✓ rtk installed via upstream script"
            return 0
        fi
        echo "~ rtk installed via upstream script but not on PATH — start a new shell, or see https://github.com/rtk-ai/rtk"
        return 0
    fi

    echo "~ rtk: upstream install failed — install manually: https://github.com/rtk-ai/rtk"
    return 0
}

_rtk_install_from_release() {
    local rtk_version
    rtk_version=$(curl -fsSL "https://api.github.com/repos/$RTK_REPO/releases?per_page=100" \
        | grep '"tag_name"' \
        | cut -d'"' -f4 \
        | grep -E '^v?[0-9]+\.[0-9]+' \
        | sort -V \
        | tail -1)
    [ -z "$rtk_version" ] && return 1

    local rtk_triple
    case "${OS}-${ARCH}" in
        darwin-arm64) rtk_triple="aarch64-apple-darwin" ;;
        darwin-amd64) rtk_triple="x86_64-apple-darwin" ;;
        linux-arm64)  rtk_triple="aarch64-unknown-linux-gnu" ;;
        linux-amd64)  rtk_triple="x86_64-unknown-linux-gnu" ;;
        *) return 1 ;;
    esac

    local rtk_url="https://github.com/${RTK_REPO}/releases/download/${rtk_version}/rtk-${rtk_triple}.tar.gz"
    local rtk_tmp="$TMP/rtk-extract"
    mkdir -p "$rtk_tmp"

    curl -fsSL "$rtk_url" -o "$TMP/rtk.tar.gz" 2>/dev/null || return 1
    tar -xzf "$TMP/rtk.tar.gz" -C "$rtk_tmp" 2>/dev/null || return 1

    local rtk_bin
    rtk_bin=$(find "$rtk_tmp" -maxdepth 4 -type f -name "rtk" ! -name "*.tar.gz" 2>/dev/null | head -1)
    if [ -z "$rtk_bin" ]; then
        rtk_bin=$(find "$rtk_tmp" -maxdepth 4 -type f -name "rtk-*" ! -name "*.tar.gz" 2>/dev/null | head -1)
    fi
    [ -z "$rtk_bin" ] && return 1

    cp "$rtk_bin" "$INSTALL_DIR/rtk"
    chmod +x "$INSTALL_DIR/rtk"
    echo "✓ Installed rtk ${rtk_version}"
    return 0
}

if [ -z "$SKIP_RTK" ]; then
    install_rtk
    echo ""
fi

# ── Install herdr (agent-native terminal multiplexer) ──────────────────────────
# maple prefers herdr as its split-pane backend when present (else tmux): harnesses
# open in a herdr side pane and approvals nudge them over herdr's socket API.
# Skip with: --skip-herdr or SKIP_HERDR=1
install_herdr() {
    if command -v herdr >/dev/null 2>&1; then
        echo "✓ herdr already installed ($(herdr --version 2>/dev/null || echo unknown))"
        return 0
    fi

    case "$OS" in
        linux|darwin) ;;
        *)
            echo "~ herdr: unsupported OS — install manually: https://herdr.dev"
            return 0
            ;;
    esac

    echo "Installing herdr terminal multiplexer…"
    if curl -fsSL https://herdr.dev/install.sh | sh; then
        if command -v herdr >/dev/null 2>&1; then
            echo "✓ herdr installed"
            return 0
        fi
        echo "~ herdr installed but not on PATH — start a new shell, or see https://herdr.dev"
        return 0
    fi

    echo "~ herdr: install failed — install manually: curl -fsSL https://herdr.dev/install.sh | sh"
    return 0
}

if [ -z "$SKIP_HERDR" ]; then
    install_herdr
    echo ""
fi

# ── PATH reminder ──────────────────────────────────────────────────────────────
if ! echo ":${PATH}:" | grep -q ":${INSTALL_DIR}:"; then
    echo "Add to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    echo ""
fi

echo "Verify with: maple --version"
