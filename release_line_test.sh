#!/bin/sh
#
# Tests for release_line.sh, against fixture tag lists. No git, no network.
#
#   ./release_line_test.sh

set -eu

root=$(cd "$(dirname "$0")" && pwd)
script="$root/release_line.sh"

# shellcheck source=test_lib.sh
. "$root/test_lib.sh"

# check NAME TAGS VERSION EXPECTED: TAGS is newline-separated, EXPECTED the
# two output lines joined by a space.
check() {
    actual=$(printf '%s\n' "$2" | "$script" "$3" | tr '\n' ' ' | sed 's/ $//')
    if [ "$actual" = "$4" ]; then
        ok "$1"
    else
        not_ok "$1" "expected '$4', got '$actual'"
    fi
}

one_line='5.8.0
5.8.1
5.9.0'

check "a release on the only line follows the previous tag and is the latest" \
    "$one_line" 6.0.0 "previous=5.9.0 latest=true"

check "the tag being released is already in the list, as it is in CI" \
    "$one_line
6.0.0" 6.0.0 "previous=5.9.0 latest=true"

two_lines="$one_line
6.0.0
6.0.1"

check "a 5.x patch after 6.0.0 follows 5.9.0 and is not the latest" \
    "$two_lines
5.9.1" 5.9.1 "previous=5.9.0 latest=false"

check "a 6.x release after a 5.x patch follows the patch" \
    "$one_line
5.9.1
6.0.0" 6.0.0 "previous=5.9.1 latest=true"

check "a 6.x patch follows the previous 6.x tag" \
    "$two_lines
5.9.1
6.0.2" 6.0.2 "previous=6.0.1 latest=true"

check "parts compare as numbers, not as text" \
    "5.9.0
5.10.0
5.2.0" 5.10.0 "previous=5.9.0 latest=true"

check "pre-releases and other tags take no part" \
    "5.9.0
6.0-alpha.0
v4
7.0.0-rc.1" 6.0.0 "previous=5.9.0 latest=true"

check "the first tag ever has no previous one" \
    "" 0.1.0 "previous= latest=true"

if printf '' | "$script" v6.0.0 > /dev/null 2>&1; then
    not_ok "a version that is not MAJOR.MINOR.PATCH is refused" "it was accepted"
else
    ok "a version that is not MAJOR.MINOR.PATCH is refused"
fi

test_summary
