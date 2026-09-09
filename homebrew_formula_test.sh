#!/bin/sh
#
# Tests for homebrew_formula.sh.
#
# The generator is run as a subprocess against a fixture checksums.txt, and
# the rendered formula is asserted against as text. Nothing here needs
# Homebrew, or a network, or a release.
#
#   ./homebrew_formula_test.sh

set -eu

root=$(cd "$(dirname "$0")" && pwd)
generator="$root/homebrew_formula.sh"

# shellcheck source=test_lib.sh
. "$root/test_lib.sh"

# A distinct, recognisable hash per asset, so a formula that pins the right
# hash to the wrong platform fails rather than passing on four equal values.
hash_for() {
    case "$1" in
        yontrack-darwin-arm64)  printf 'aa%s' "$(pad 62)" ;;
        yontrack-darwin-amd64)  printf 'bb%s' "$(pad 62)" ;;
        yontrack-linux-arm64)   printf 'cc%s' "$(pad 62)" ;;
        yontrack-linux-amd64)   printf 'dd%s' "$(pad 62)" ;;
        *)                      printf 'ee%s' "$(pad 62)" ;;
    esac
}

# n zeroes, to bring a marker up to the 64 characters a sha256 has.
pad() {
    i=0
    while [ "$i" -lt "$1" ]; do
        printf '0'
        i=$((i + 1))
    done
}

# Writes a checksums.txt covering every asset a release publishes, in the
# `sha256sum` format, and echoes the directory holding it.
#
# Any asset named in $1 is left out, which is how the missing-asset cases are
# built without hand-writing a second fixture.
workspace() {
    omit=${1:-}
    ws=$(mktemp -d)
    for asset in \
        yontrack-darwin-amd64 \
        yontrack-darwin-arm64 \
        yontrack-linux-386 \
        yontrack-linux-amd64 \
        yontrack-linux-arm64 \
        yontrack-windows-amd64.exe
    do
        [ "$asset" = "$omit" ] && continue
        printf '%s  %s\n' "$(hash_for "$asset")" "$asset" >> "$ws/checksums.txt"
    done
    printf '%s' "$ws"
}

# Runs the generator, capturing its combined output and exit status.
run_generator() {
    set +e
    out=$("$generator" "$@" 2>&1)
    status=$?
    set -e
}

# The hash pinned immediately below the url for one asset. Reading the pair
# rather than the two independently is the point: it catches a formula that
# lists every right value against the wrong platform.
sha_under_url() {
    printf '%s\n' "$out" | awk -v asset="$1" '
        index($0, "/" asset "\"") { hit = 1; next }
        hit && $1 == "sha256" { gsub(/"/, "", $2); print $2; hit = 0 }
    '
}

# --- what it renders -------------------------------------------------------

t="renders a formula for the version it was given"
ws=$(workspace)
run_generator 9.9.9 "$ws/checksums.txt"
if [ "$status" -ne 0 ]; then
    not_ok "$t" "exited $status: $out"
elif ! printf '%s\n' "$out" | grep -q '^class Yontrack < Formula$'; then
    not_ok "$t" "no formula class in the output: $out"
elif ! printf '%s\n' "$out" | grep -q '^  version "9\.9\.9"$'; then
    not_ok "$t" "the version stanza does not say 9.9.9: $out"
else
    ok "$t"
fi
rm -rf "$ws"

t="points every url at the release being published"
ws=$(workspace)
run_generator 9.9.9 "$ws/checksums.txt"
urls=$(printf '%s\n' "$out" | grep -c 'releases/download/9\.9\.9/yontrack-' || true)
if [ "$urls" -eq 4 ]; then
    ok "$t"
else
    not_ok "$t" "expected 4 urls under the 9.9.9 release, found $urls: $out"
fi
rm -rf "$ws"

t="pins each asset to its own hash"
ws=$(workspace)
run_generator 9.9.9 "$ws/checksums.txt"
mismatched=""
for asset in \
    yontrack-darwin-amd64 \
    yontrack-darwin-arm64 \
    yontrack-linux-amd64 \
    yontrack-linux-arm64
do
    [ "$(sha_under_url "$asset")" = "$(hash_for "$asset")" ] \
        || mismatched="$mismatched $asset"
done
if [ -n "$mismatched" ]; then
    not_ok "$t" "wrong or missing hash for:$mismatched"
else
    ok "$t"
fi
rm -rf "$ws"

# Homebrew runs on 64-bit Linux and macOS only. Offering an asset it cannot
# install would fail at `brew install`, on the user's machine, not here.
t="leaves out the assets Homebrew cannot install"
ws=$(workspace)
run_generator 9.9.9 "$ws/checksums.txt"
if printf '%s\n' "$out" | grep -q 'yontrack-linux-386'; then
    not_ok "$t" "the formula offers linux/386, which Homebrew does not support"
elif printf '%s\n' "$out" | grep -q 'yontrack-windows'; then
    not_ok "$t" "the formula offers a Windows asset"
else
    ok "$t"
fi
rm -rf "$ws"

t="declares the licence and the homepage"
ws=$(workspace)
run_generator 9.9.9 "$ws/checksums.txt"
if ! printf '%s\n' "$out" | grep -q '^  license "MIT"$'; then
    not_ok "$t" "no license stanza: $out"
elif ! printf '%s\n' "$out" | grep -q '^  homepage "https://github.com/yontrack/yontrack-cli"$'; then
    not_ok "$t" "no homepage stanza: $out"
else
    ok "$t"
fi
rm -rf "$ws"

# --- what it refuses to render ---------------------------------------------

# The failure that matters. A release missing one platform's asset would
# otherwise render a formula pinning an empty hash, which every user on that
# platform meets and nobody else does.
t="refuses to render when the release does not cover an asset"
ws=$(workspace yontrack-darwin-arm64)
run_generator 9.9.9 "$ws/checksums.txt"
if [ "$status" -eq 0 ]; then
    not_ok "$t" "rendered a formula anyway: $out"
elif ! printf '%s' "$out" | grep -q 'yontrack-darwin-arm64'; then
    not_ok "$t" "the error does not name the missing asset: $out"
else
    ok "$t"
fi
rm -rf "$ws"

t="refuses to render from a truncated hash"
ws=$(workspace yontrack-linux-amd64)
printf 'deadbeef  yontrack-linux-amd64\n' >> "$ws/checksums.txt"
run_generator 9.9.9 "$ws/checksums.txt"
if [ "$status" -eq 0 ]; then
    not_ok "$t" "rendered a formula pinning a hash that is not a sha256: $out"
else
    ok "$t"
fi
rm -rf "$ws"

t="refuses to render without a version"
ws=$(workspace)
run_generator "" "$ws/checksums.txt"
if [ "$status" -eq 0 ]; then
    not_ok "$t" "rendered a formula with no version: $out"
else
    ok "$t"
fi
rm -rf "$ws"

t="reports a checksums file that is not there"
run_generator 9.9.9 /nonexistent/checksums.txt
if [ "$status" -eq 0 ]; then
    not_ok "$t" "exited 0: $out"
elif ! printf '%s' "$out" | grep -q 'does not exist'; then
    not_ok "$t" "unhelpful error: $out"
else
    ok "$t"
fi

# --- the bottle block ------------------------------------------------------

# Writes a bottle manifest naming one tag per line, and echoes its path.
bottles() {
    bf="$1/bottles.txt"
    : > "$bf"
    shift
    for tag in "$@"; do
        printf '%s  %s\n' "$(hash_for "bottle-$tag")" "$tag" >> "$bf"
    done
    printf '%s' "$bf"
}

# The formula has to be renderable without bottles, because that is the
# version the release installs in order to build them. A bottle block naming
# hashes that do not exist yet would make it uninstallable.
t="renders no bottle block when given no manifest"
ws=$(workspace)
run_generator 9.9.9 "$ws/checksums.txt"
if [ "$status" -ne 0 ]; then
    not_ok "$t" "exited $status: $out"
elif printf '%s\n' "$out" | grep -q 'bottle do'; then
    not_ok "$t" "rendered a bottle block anyway: $out"
else
    ok "$t"
fi
rm -rf "$ws"

t="renders a bottle block from a manifest"
ws=$(workspace)
bf=$(bottles "$ws" arm64_ventura x86_64_linux arm64_linux)
run_generator 9.9.9 "$ws/checksums.txt" "$bf"
if [ "$status" -ne 0 ]; then
    not_ok "$t" "exited $status: $out"
elif ! printf '%s\n' "$out" | grep -q '^  bottle do$'; then
    not_ok "$t" "no bottle block: $out"
elif ! printf '%s\n' "$out" | grep -q 'root_url "https://github.com/yontrack/yontrack-cli/releases/download/9\.9\.9"'; then
    not_ok "$t" "the root_url does not point at the release: $out"
else
    ok "$t"
fi
rm -rf "$ws"

t="pins each bottle tag to its own hash"
ws=$(workspace)
bf=$(bottles "$ws" arm64_ventura x86_64_linux arm64_linux)
run_generator 9.9.9 "$ws/checksums.txt" "$bf"
wrong=""
for tag in arm64_ventura x86_64_linux arm64_linux; do
    line=$(printf '%s\n' "$out" | grep "$tag:" || true)
    printf '%s' "$line" | grep -q "$(hash_for "bottle-$tag")" || wrong="$wrong $tag"
done
if [ -n "$wrong" ]; then
    not_ok "$t" "wrong or missing bottle hash for:$wrong"
else
    ok "$t"
fi
rm -rf "$ws"

# A standalone Go binary references nothing under the Homebrew prefix, so the
# bottle is valid wherever the prefix happens to be. Without this, Homebrew
# refuses to pour it into a non-default prefix.
t="declares the bottles as relocatable"
ws=$(workspace)
bf=$(bottles "$ws" arm64_ventura x86_64_linux arm64_linux)
run_generator 9.9.9 "$ws/checksums.txt" "$bf"
relocatable=$(printf '%s\n' "$out" | grep -c 'cellar: :any_skip_relocation' || true)
if [ "$relocatable" -eq 3 ]; then
    ok "$t"
else
    not_ok "$t" "expected 3 relocatable bottles, found $relocatable: $out"
fi
rm -rf "$ws"

# Only the tags the manifest names. A tag rendered without a bottle behind it
# sends Homebrew to a 404 rather than to the source path.
t="renders only the tags the manifest names"
ws=$(workspace)
bf=$(bottles "$ws" arm64_ventura)
run_generator 9.9.9 "$ws/checksums.txt" "$bf"
if printf '%s\n' "$out" | grep -qE 'x86_64_linux:|arm64_linux:'; then
    not_ok "$t" "rendered a tag the manifest does not name: $out"
elif [ "$(printf '%s\n' "$out" | grep -c 'cellar:' || true)" -ne 1 ]; then
    not_ok "$t" "expected exactly one bottle: $out"
else
    ok "$t"
fi
rm -rf "$ws"

t="refuses to render from a bottle hash that is not a sha256"
ws=$(workspace)
bf="$ws/bottles.txt"
printf 'deadbeef  arm64_ventura\n' > "$bf"
run_generator 9.9.9 "$ws/checksums.txt" "$bf"
if [ "$status" -eq 0 ]; then
    not_ok "$t" "rendered a bottle block pinning a hash that is not a sha256: $out"
else
    ok "$t"
fi
rm -rf "$ws"

t="reports a bottle manifest that is not there"
ws=$(workspace)
run_generator 9.9.9 "$ws/checksums.txt" /nonexistent/bottles.txt
if [ "$status" -eq 0 ]; then
    not_ok "$t" "exited 0: $out"
elif ! printf '%s' "$out" | grep -q 'does not exist'; then
    not_ok "$t" "unhelpful error: $out"
else
    ok "$t"
fi
rm -rf "$ws"

# An empty manifest is what a release with no bottles produces. It must render
# a working formula, not an empty bottle block, which Homebrew rejects.
t="renders no bottle block from an empty manifest"
ws=$(workspace)
: > "$ws/bottles.txt"
run_generator 9.9.9 "$ws/checksums.txt" "$ws/bottles.txt"
if [ "$status" -ne 0 ]; then
    not_ok "$t" "exited $status: $out"
elif printf '%s\n' "$out" | grep -q 'bottle do'; then
    not_ok "$t" "rendered an empty bottle block: $out"
else
    ok "$t"
fi
rm -rf "$ws"

# --- the formula is Ruby ---------------------------------------------------

# Homebrew evaluates the formula as Ruby, so a syntax error is an install
# failure for everyone. Ruby ships with macOS and with the CI runners, but the
# suite still has to pass without it.
t="renders syntactically valid Ruby"
if command -v ruby > /dev/null 2>&1; then
    ws=$(workspace)
    bf=$(bottles "$ws" arm64_ventura x86_64_linux arm64_linux)
    invalid=""
    # Both shapes, because the release renders each of them in turn.
    for args in "$ws/checksums.txt" "$ws/checksums.txt $bf"; do
        # shellcheck disable=SC2086
        run_generator 9.9.9 $args
        printf '%s\n' "$out" > "$ws/yontrack.rb"
        syntax=$(ruby -c "$ws/yontrack.rb" 2>&1) || invalid="$invalid
$syntax"
    done
    if [ -n "$invalid" ]; then
        not_ok "$t" "$invalid"
    else
        ok "$t"
    fi
    rm -rf "$ws"
else
    skip "$t" "ruby is not installed"
fi

test_summary
