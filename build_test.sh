#!/bin/sh
#
# Checks that the build publishes a binary for every platform it is meant to,
# and that checksums.txt covers each one with the right hash.
#
# It always runs the build. An earlier version asserted against whatever was
# already on disk, which meant a stale build left by another branch could pass
# for a build script that no longer produced it — the one regression this is
# here to catch.
#
#   ./build_test.sh [version]

set -eu

root=$(cd "$(dirname "$0")" && pwd)
cd "$root"

# shellcheck source=test_lib.sh
. "$root/test_lib.sh"

version=${1:-test}

# Every binary a release is expected to carry. Written out here rather than
# read from go-executable-build.bash on purpose: deriving the list from the
# script under test would make this agree with it by construction.
expected="yontrack-darwin-amd64
yontrack-darwin-arm64
yontrack-linux-386
yontrack-linux-amd64
yontrack-linux-arm64
yontrack-windows-amd64.exe"

# Artifacts are gitignored, so they outlive a branch switch. Clearing them
# first means every assertion below is about this build and nothing else.
rm -f yontrack-* ontrack-cli-* checksums.txt

printf 'Building %s\n' "$version"
./go-executable-build.bash "$version" > /dev/null

for artifact in $expected; do
    if [ ! -f "$artifact" ]; then
        not_ok "$artifact is built" "no such file"
        continue
    fi
    ok "$artifact is built"

    recorded=$(awk -v name="$artifact" '$2 == name { print $1 }' checksums.txt)
    if [ -z "$recorded" ]; then
        not_ok "$artifact is in checksums.txt" "not listed"
    elif [ "$recorded" != "$(sha256_of "$artifact")" ]; then
        not_ok "$artifact is in checksums.txt" "the listed hash does not match the file"
    else
        ok "$artifact is in checksums.txt"
    fi
done

# Catches a platform added to the build script but not to the list above, which
# would otherwise ship unannounced and unchecked. -F because the "." in
# yontrack-windows-amd64.exe is a wildcard to grep otherwise.
unexpected=""
for artifact in yontrack-* ontrack-cli-*; do
    [ -f "$artifact" ] || continue
    printf '%s\n' "$expected" | grep -qxF "$artifact" || unexpected="$unexpected $artifact"
done
if [ -n "$unexpected" ]; then
    not_ok "nothing unexpected is built" "built but not in the expected list:$unexpected"
else
    ok "nothing unexpected is built"
fi

test_summary
