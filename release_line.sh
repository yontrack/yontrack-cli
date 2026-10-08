#!/bin/sh
#
# Places a release tag among the others, for tag.yml. See RELEASING.md,
# "Release lines": main carries the current major, v5 patches the previous
# one, so the tag being released is not necessarily the newest.
#
# Reads the repository's tags on stdin and prints, for the version given as
# argument, two lines in the format of $GITHUB_OUTPUT:
#
#   previous=<highest MAJOR.MINOR.PATCH tag below the version, or empty>
#   latest=<true when the version is the highest MAJOR.MINOR.PATCH tag>
#
# `previous` is what the changelog starts from: 5.9.0 for 5.9.1, 5.9.1 for
# 6.0.0 if it exists, 6.0.0 for 6.0.1. `latest` decides whether the release
# is GitHub's latest, which install.sh follows, and whether the Homebrew tap
# moves to it. A 5.x patch cut after 6.0.0 must do neither.
#
#   git tag -l | ./release_line.sh 5.9.1

set -eu

version=${1:-}
if ! printf '%s\n' "$version" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    printf 'usage: release_line.sh MAJOR.MINOR.PATCH < tags\n' >&2
    exit 1
fi

# Numeric sort on each part rather than `sort -V`, which the macOS sort the
# tests also run on does not guarantee. Other tags (pre-releases, anything
# not MAJOR.MINOR.PATCH) take no part in the ordering.
sorted=$({ cat; printf '%s\n' "$version"; } \
    | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' \
    | sort -u -t. -k1,1n -k2,2n -k3,3n)

previous=$(printf '%s\n' "$sorted" | grep -B1 -x "$version" | grep -v -x "$version" || true)

if [ "$(printf '%s\n' "$sorted" | tail -n 1)" = "$version" ]; then
    latest=true
else
    latest=false
fi

printf 'previous=%s\n' "$previous"
printf 'latest=%s\n' "$latest"
