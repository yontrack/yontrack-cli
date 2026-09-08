#!/bin/sh
#
# Installs the Yontrack CLI.
#
#   curl -fsSL https://raw.githubusercontent.com/yontrack/yontrack-cli/main/install.sh | sh
#
# Environment:
#
#   INSTALL_DIR  where to put the binary. Default /usr/local/bin.
#   VERSION      release to install, e.g. 5.4.0. Default: the latest release.
#   BASE_URL     where to fetch releases from. Default: GitHub. For mirrors.
#
# This script never asks for a password. If it cannot write to INSTALL_DIR it
# installs nothing and prints the command to run instead.

set -eu

repo="yontrack/yontrack-cli"
bin="yontrack"

install_dir=${INSTALL_DIR:-/usr/local/bin}
base_url=${BASE_URL:-"https://github.com/$repo/releases"}

tmp=""
staged=""

cleanup() {
    [ -n "$tmp" ] && rm -rf "$tmp"
    [ -n "$staged" ] && rm -f "$staged"
    return 0
}
trap cleanup EXIT INT TERM

fail() {
    printf 'error: %s\n' "$1" >&2
    exit 1
}

# Maps `uname -s` onto the GOOS used in the published asset names.
detect_os() {
    detected=$(uname -s)
    case $(printf '%s' "$detected" | tr '[:upper:]' '[:lower:]') in
        linux)  printf 'linux' ;;
        darwin) printf 'darwin' ;;
        *)      fail "unsupported operating system: $detected. Binaries are published for Linux and macOS; see https://github.com/$repo/releases" ;;
    esac
}

# Maps `uname -m` onto the GOARCH used in the published asset names.
detect_arch() {
    detected=$(uname -m)
    case "$detected" in
        x86_64 | amd64)  printf 'amd64' ;;
        aarch64 | arm64) printf 'arm64' ;;
        i386 | i686)     printf '386' ;;
        *)               fail "unsupported architecture: $detected. See https://github.com/$repo/releases for what is published" ;;
    esac
}

sha256_of() {
    if command -v sha256sum > /dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum > /dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    else
        fail 'neither sha256sum nor shasum is available, so the download cannot be verified'
    fi
}

# Follows the /releases/latest redirect to the tag it points at. Resolving the
# version once, rather than downloading twice from /latest, keeps the binary
# and its checksums from straddling a release published mid-install.
resolve_latest() {
    effective=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "$base_url/latest") \
        || fail "could not reach $base_url/latest to find the latest release"
    resolved=${effective##*/}
    if [ -z "$resolved" ] || [ "$resolved" = "latest" ]; then
        fail "could not work out the latest version from $effective; set VERSION to install a specific release"
    fi
    printf '%s' "$resolved"
}

command -v curl > /dev/null 2>&1 \
    || fail 'curl is required to download the release, but is not installed'

os=$(detect_os)
arch=$(detect_arch)
asset="$bin-$os-$arch"

if [ -n "${VERSION:-}" ]; then
    version=$VERSION
else
    version=$(resolve_latest)
fi

# Checked before downloading anything, so a permission problem costs the user
# an error rather than an error and a wasted download.
if [ ! -d "$install_dir" ]; then
    mkdir -p "$install_dir" 2> /dev/null \
        || fail "$install_dir does not exist and could not be created.
       Create it, or install somewhere you own:
           INSTALL_DIR=\$HOME/.local/bin sh install.sh"
fi

if [ ! -w "$install_dir" ]; then
    fail "$install_dir is not writable.
       Re-run with elevated privileges:
           curl -fsSL https://raw.githubusercontent.com/$repo/main/install.sh | sudo sh
       or install somewhere you own:
           INSTALL_DIR=\$HOME/.local/bin sh install.sh"
fi

tmp=$(mktemp -d)
release_url="$base_url/download/$version"

printf 'Downloading %s %s\n' "$bin" "$version"

# -f matters: without it a 404 page is written out and installed as the binary.
curl -fsSL "$release_url/$asset" -o "$tmp/$asset" \
    || fail "could not download $asset from $release_url
       That release may not publish a binary for $os/$arch. See https://github.com/$repo/releases"

curl -fsSL "$release_url/checksums.txt" -o "$tmp/checksums.txt" \
    || fail "could not download checksums.txt from $release_url, so $asset cannot be verified"

expected=$(awk -v name="$asset" '$2 == name { print $1 }' "$tmp/checksums.txt")
[ -n "$expected" ] || fail "checksums.txt for $version does not cover $asset, so it cannot be verified"

actual=$(sha256_of "$tmp/$asset")
if [ "$actual" != "$expected" ]; then
    fail "checksum mismatch for $asset
       expected $expected
       got      $actual
       Nothing has been installed. This means the download was corrupted or tampered with."
fi

# Staged and renamed rather than copied over the target, so an in-use binary
# does not fail with ETXTBSY and a half-written file is never left behind.
staged="$install_dir/.$bin.$$"
cp "$tmp/$asset" "$staged"
chmod 755 "$staged"
mv -f "$staged" "$install_dir/$bin"
staged=""

printf 'Installed %s %s to %s\n' "$bin" "$version" "$install_dir/$bin"

case ":$PATH:" in
    *":$install_dir:"*) ;;
    *) printf 'warning: %s is not on your PATH\n' "$install_dir" ;;
esac
