#!/bin/sh
#
# Shared helpers for the shell test suites. Sourced, not executed.
#
# A suite reports with ok/not_ok/skip and ends with `test_summary`, which
# prints the tally and exits non-zero if anything failed.

test_passed=0
test_failed=0

ok() {
    test_passed=$((test_passed + 1))
    printf 'ok   %s\n' "$1"
}

not_ok() {
    test_failed=$((test_failed + 1))
    printf 'FAIL %s\n       %s\n' "$1" "$2"
}

skip() {
    printf 'skip %s (%s)\n' "$1" "$2"
}

test_summary() {
    printf '\n%d passed, %d failed\n' "$test_passed" "$test_failed"
    [ "$test_failed" -eq 0 ]
}

sha256_of() {
    if command -v sha256sum > /dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum > /dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    else
        printf 'error: neither sha256sum nor shasum is available\n' >&2
        exit 1
    fi
}

# root can write to any directory, so the permission tests have nothing to
# assert when a suite runs as root — as it does in most containers.
is_root() {
    [ "$(id -u)" = "0" ]
}
