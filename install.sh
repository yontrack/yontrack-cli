#!/bin/sh
#
# Installs the Yontrack CLI.
#
#   curl -fsSL https://raw.githubusercontent.com/yontrack/yontrack-cli/main/install.sh | sh
#
# Environment:
#
#   INSTALL_DIR  where to put the binary. Default $HOME/.local/bin.
#   VERSION      release to install, e.g. 5.4.0. Default: the latest release.
#   BASE_URL     where to fetch releases from. Default: GitHub. For mirrors.
#
# This script never asks for a password. If it cannot write to INSTALL_DIR it
# installs nothing and prints the command to run instead.

set -eu

repo="yontrack/yontrack-cli"
bin="yontrack"

base_url=${BASE_URL:-"https://github.com/$repo/releases"}
script_url="https://raw.githubusercontent.com/$repo/main/install.sh"

tmp=""
staged=""

cleanup() {
    [ -n "$tmp" ] && rm -rf "$tmp" && tmp=""
    [ -n "$staged" ] && rm -f "$staged" && staged=""
    return 0
}

# A handler that just returns lets the script carry on after the signal, with
# its temporary directory already deleted, so INT and TERM exit explicitly.
trap cleanup EXIT
trap 'cleanup; exit 130' INT
trap 'cleanup; exit 143' TERM

fail() {
    printf 'error: %s\n' "$1" >&2
    exit 1
}

# Resolved here rather than with the other settings above, because working out
# the default can fail and fail() has to exist first.
#
# The default is under $HOME: a per-user tool, with per-user config in
# ~/.yontrack-config.yaml, that installs without privileges. A system location
# would need a password on any machine where the user is not also the admin —
# every stock macOS, where /usr/local is root-owned and Homebrew lives in
# /opt/homebrew.
if [ -n "${INSTALL_DIR:-}" ]; then
    install_dir=$INSTALL_DIR
elif [ -n "${HOME:-}" ]; then
    install_dir="$HOME/.local/bin"
else
    fail 'HOME is not set, so there is no default install directory.
       Say where to install:
           INSTALL_DIR=/path/to/bin sh install.sh'
fi

# Maps `uname -s` onto the GOOS used in the published asset names.
detect_os() {
    detected=$(uname -s)
    case $(printf '%s' "$detected" | tr '[:upper:]' '[:lower:]') in
        linux)  printf 'linux' ;;
        darwin) printf 'darwin' ;;
        *)      fail "unsupported platform: $detected/$(uname -m). Binaries are published for Linux and macOS; see https://github.com/$repo/releases" ;;
    esac
}

# Maps `uname -m` onto the GOARCH used in the published asset names.
detect_arch() {
    detected=$(uname -m | tr '[:upper:]' '[:lower:]')
    case "$detected" in
        x86_64 | amd64)  printf 'amd64' ;;
        aarch64 | arm64) printf 'arm64' ;;
        i386 | i686)     printf '386' ;;
        *)               fail "unsupported platform: $(uname -s)/$detected. See https://github.com/$repo/releases for what is published" ;;
    esac
}

# Names the startup file the PATH line should go in, so the hint below can be
# pasted rather than adapted. macOS runs login shells in Terminal, which read
# ~/.bash_profile and never ~/.bashrc; Linux is the other way round. Anything
# else gets ~/.profile, which every POSIX shell reads.
# The tilde is printed for the user to read and paste into their own shell,
# which expands it; nothing here uses it as a path.
# shellcheck disable=SC2088
shell_profile() {
    case "${SHELL:-}" in
        */zsh)  printf '~/.zshrc' ;;
        */bash)
            if [ "$1" = "darwin" ]; then
                printf '~/.bash_profile'
            else
                printf '~/.bashrc'
            fi
            ;;
        *)      printf '~/.profile' ;;
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

# `|| exit` rather than leaning on `set -e`: fail() runs inside the command
# substitution, so its exit only ends the subshell.
os=$(detect_os) || exit 1
arch=$(detect_arch) || exit 1
asset="$bin-$os-$arch"

if [ -n "${VERSION:-}" ]; then
    version=$VERSION
else
    version=$(resolve_latest) || exit 1
fi

# Checked before downloading anything, so a permission problem costs the user
# an error rather than an error and a wasted download.
if [ ! -d "$install_dir" ]; then
    mkdir -p "$install_dir" 2> /dev/null \
        || fail "$install_dir does not exist and could not be created.
       Create it, or install somewhere you own:
           curl -fsSL $script_url | INSTALL_DIR=\$HOME/.local/bin sh"
fi

# Only reachable when INSTALL_DIR points somewhere the user does not own: the
# default is under their home directory. So the first suggestion is to drop
# the override, and the second is how to install into a system directory
# deliberately — downloaded and read first, never piped into sudo.
if [ ! -w "$install_dir" ]; then
    fail "$install_dir is not writable.
       Install somewhere you own, which needs no privileges — leaving
       INSTALL_DIR unset installs into \$HOME/.local/bin:
           curl -fsSL $script_url | sh
       or download this script, read it, and run that with sudo:
           curl -fsSLo install.sh $script_url
           sudo INSTALL_DIR=$install_dir sh install.sh"
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

actual=$(sha256_of "$tmp/$asset") || exit 1
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
    *)
        # Written as $HOME/... when it is under the home directory, so the
        # line stays right if the profile is shared between machines.
        path_entry=$install_dir
        if [ -n "${HOME:-}" ]; then
            case "$install_dir" in
                "$HOME"/*) path_entry="\$HOME${install_dir#"$HOME"}" ;;
            esac
        fi
        printf 'warning: %s is not on your PATH.\n' "$install_dir"
        printf '         Add it, then restart your shell:\n'
        # $PATH and $HOME are printed for the user's shell to expand later,
        # not for this one to expand now, so the quotes are deliberate.
        # shellcheck disable=SC2016
        printf '             echo '\''export PATH="%s:$PATH"'\'' >> %s\n' \
            "$path_entry" "$(shell_profile "$os")"
        ;;
esac
