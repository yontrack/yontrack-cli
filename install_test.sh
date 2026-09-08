#!/bin/sh
#
# Tests for install.sh.
#
# The installer is exercised end to end as a subprocess: `uname` is faked on
# PATH so the detected platform is deterministic, and BASE_URL points at a
# local fixture served over file://, so no test touches the network.
#
#   ./install_test.sh

set -eu

root=$(cd "$(dirname "$0")" && pwd)
installer="$root/install.sh"

passed=0
failed=0

ok() {
    passed=$((passed + 1))
    printf 'ok   %s\n' "$1"
}

not_ok() {
    failed=$((failed + 1))
    printf 'FAIL %s\n       %s\n' "$1" "$2"
}

skip() {
    printf 'skip %s (%s)\n' "$1" "$2"
}

# root can write to any directory, so the permission tests have nothing to
# assert when the suite runs as root — as it does in most containers.
is_root() {
    [ "$(id -u)" = "0" ]
}

sha256_of() {
    if command -v sha256sum > /dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    else
        shasum -a 256 "$1" | awk '{print $1}'
    fi
}

# Builds a workspace holding a fake release and a fake `uname`, and echoes it.
#   $1 the value `uname -s` should report
#   $2 the value `uname -m` should report
workspace() {
    ws=$(mktemp -d)
    mkdir -p "$ws/fakebin" "$ws/target" "$ws/release/download/1.2.3"

    cat > "$ws/fakebin/uname" <<EOF
#!/bin/sh
case "\$1" in
    -s) echo "$1" ;;
    -m) echo "$2" ;;
esac
EOF
    chmod +x "$ws/fakebin/uname"

    printf '#!/bin/sh\necho fake yontrack\n' > "$ws/release/download/1.2.3/yontrack-linux-amd64"
    printf '#!/bin/sh\necho wrong platform\n' > "$ws/release/download/1.2.3/yontrack-darwin-arm64"

    ( cd "$ws/release/download/1.2.3" \
        && for f in yontrack-*; do printf '%s  %s\n' "$(sha256_of "$f")" "$f"; done \
        > checksums.txt )

    printf '%s' "$ws"
}

# Runs the installer against a workspace, capturing output and exit status.
# TEST_PATH replaces the PATH the installer sees, for the case where a tool it
# depends on is absent. /bin/sh is invoked by absolute path so the installer
# still starts when TEST_PATH holds nothing useful.
run_installer() {
    run_ws=$1
    set +e
    out=$(PATH="${TEST_PATH:-$run_ws/fakebin:$PATH}" \
        BASE_URL="file://$run_ws/release" \
        VERSION="${TEST_VERSION-1.2.3}" \
        INSTALL_DIR="$run_ws/target" \
        /bin/sh "$installer" 2>&1)
    status=$?
    set -e
}

# --- installs the binary for the detected platform -------------------------

t="installs the binary for the detected platform"
ws=$(workspace Linux x86_64)
run_installer "$ws"
if [ "$status" -ne 0 ]; then
    not_ok "$t" "exited $status: $out"
elif [ ! -x "$ws/target/yontrack" ]; then
    not_ok "$t" "no executable at target/yontrack"
elif ! grep -q 'fake yontrack' "$ws/target/yontrack"; then
    not_ok "$t" "installed the wrong asset"
else
    ok "$t"
fi
rm -rf "$ws"

t="installs it as mode 755"
ws=$(workspace Linux x86_64)
run_installer "$ws"
# The filename is fixed and known, so `ls` parsing is safe here.
# shellcheck disable=SC2012
mode=$(ls -l "$ws/target/yontrack" | cut -c1-10)
if [ "$mode" = "-rwxr-xr-x" ]; then
    ok "$t"
else
    not_ok "$t" "mode is $mode, expected -rwxr-xr-x"
fi
rm -rf "$ws"

t="reports the version it installed"
ws=$(workspace Linux x86_64)
run_installer "$ws"
if printf '%s' "$out" | grep -q '1\.2\.3'; then
    ok "$t"
else
    not_ok "$t" "output never mentions 1.2.3: $out"
fi
rm -rf "$ws"

# --- platform detection ----------------------------------------------------

t="normalises aarch64 and Darwin to the published asset name"
ws=$(workspace Darwin aarch64)
run_installer "$ws"
if [ "$status" -ne 0 ]; then
    not_ok "$t" "exited $status: $out"
elif ! grep -q 'wrong platform' "$ws/target/yontrack"; then
    not_ok "$t" "did not pick yontrack-darwin-arm64"
else
    ok "$t"
fi
rm -rf "$ws"

t="normalises arm64, the value macOS actually reports"
ws=$(workspace Darwin arm64)
run_installer "$ws"
if [ "$status" -ne 0 ]; then
    not_ok "$t" "exited $status: $out"
elif ! grep -q 'wrong platform' "$ws/target/yontrack"; then
    not_ok "$t" "did not pick yontrack-darwin-arm64"
else
    ok "$t"
fi
rm -rf "$ws"

t="normalises i686 to the published 386 asset"
ws=$(workspace Linux i686)
printf '#!/bin/sh\necho 386 build\n' > "$ws/release/download/1.2.3/yontrack-linux-386"
( cd "$ws/release/download/1.2.3" \
    && for f in yontrack-*; do printf '%s  %s\n' "$(sha256_of "$f")" "$f"; done \
    > checksums.txt )
run_installer "$ws"
if [ "$status" -ne 0 ]; then
    not_ok "$t" "exited $status: $out"
elif ! grep -q '386 build' "$ws/target/yontrack"; then
    not_ok "$t" "did not pick yontrack-linux-386"
else
    ok "$t"
fi
rm -rf "$ws"

t="refuses an unsupported operating system by name"
ws=$(workspace SunOS x86_64)
run_installer "$ws"
if [ "$status" -eq 0 ]; then
    not_ok "$t" "exited 0 on SunOS"
elif ! printf '%s' "$out" | grep -qi 'sunos'; then
    not_ok "$t" "error does not name the detected OS: $out"
else
    ok "$t"
fi
rm -rf "$ws"

t="refuses an unsupported architecture by name"
ws=$(workspace Linux mips64)
run_installer "$ws"
if [ "$status" -eq 0 ]; then
    not_ok "$t" "exited 0 on mips64"
elif ! printf '%s' "$out" | grep -qi 'mips64'; then
    not_ok "$t" "error does not name the detected architecture: $out"
else
    ok "$t"
fi
rm -rf "$ws"

# --- resolving the latest release ------------------------------------------

t="resolves the latest release when VERSION is not set"
ws=$(workspace Linux x86_64)
real_curl=$(command -v curl)
# The installer probes the latest-release redirect with -I and reads the tag
# off the effective URL. Answer that one call; pass everything else through to
# the real curl, so the download and verification are not mocked away.
cat > "$ws/fakebin/curl" <<EOF
#!/bin/sh
for arg in "\$@"; do
    if [ "\$arg" = "-fsSLI" ]; then
        printf 'https://example.invalid/releases/tag/1.2.3'
        exit 0
    fi
done
exec $real_curl "\$@"
EOF
chmod +x "$ws/fakebin/curl"
TEST_VERSION=""
run_installer "$ws"
unset TEST_VERSION
if [ "$status" -ne 0 ]; then
    not_ok "$t" "exited $status: $out"
elif [ ! -x "$ws/target/yontrack" ]; then
    not_ok "$t" "resolved the version but installed nothing"
elif ! printf '%s' "$out" | grep -q '1\.2\.3'; then
    not_ok "$t" "never reported the version it resolved: $out"
else
    ok "$t"
fi
rm -rf "$ws"

t="fails clearly when the latest release cannot be resolved"
ws=$(workspace Linux x86_64)
# A redirect that never happened: the effective URL still ends in "latest".
cat > "$ws/fakebin/curl" <<'EOF'
#!/bin/sh
for arg in "$@"; do
    if [ "$arg" = "-fsSLI" ]; then
        printf 'https://example.invalid/releases/latest'
        exit 0
    fi
done
exit 1
EOF
chmod +x "$ws/fakebin/curl"
TEST_VERSION=""
run_installer "$ws"
unset TEST_VERSION
if [ "$status" -eq 0 ]; then
    not_ok "$t" "exited 0 without resolving a version"
elif [ -e "$ws/target/yontrack" ]; then
    not_ok "$t" "installed something anyway"
else
    ok "$t"
fi
rm -rf "$ws"

# --- checksum verification -------------------------------------------------

t="refuses a binary whose checksum does not match"
ws=$(workspace Linux x86_64)
printf 'tampered\n' >> "$ws/release/download/1.2.3/yontrack-linux-amd64"
run_installer "$ws"
if [ "$status" -eq 0 ]; then
    not_ok "$t" "exited 0 on a checksum mismatch"
elif [ -e "$ws/target/yontrack" ]; then
    not_ok "$t" "installed the binary anyway"
else
    ok "$t"
fi
rm -rf "$ws"

t="refuses to install when checksums.txt is missing"
ws=$(workspace Linux x86_64)
rm "$ws/release/download/1.2.3/checksums.txt"
run_installer "$ws"
if [ "$status" -eq 0 ]; then
    not_ok "$t" "exited 0 with no checksums.txt"
elif [ -e "$ws/target/yontrack" ]; then
    not_ok "$t" "installed the binary anyway"
else
    ok "$t"
fi
rm -rf "$ws"

t="refuses to install when checksums.txt does not cover the asset"
ws=$(workspace Linux x86_64)
grep -v 'yontrack-linux-amd64' "$ws/release/download/1.2.3/checksums.txt" > "$ws/sums" \
    && mv "$ws/sums" "$ws/release/download/1.2.3/checksums.txt"
run_installer "$ws"
if [ "$status" -eq 0 ]; then
    not_ok "$t" "exited 0 with the asset absent from checksums.txt"
elif [ -e "$ws/target/yontrack" ]; then
    not_ok "$t" "installed the binary anyway"
else
    ok "$t"
fi
rm -rf "$ws"

# --- missing asset ---------------------------------------------------------

t="fails clearly when the release has no asset for the platform"
ws=$(workspace Linux x86_64)
rm "$ws/release/download/1.2.3/yontrack-linux-amd64"
run_installer "$ws"
if [ "$status" -eq 0 ]; then
    not_ok "$t" "exited 0 with no asset published"
elif ! printf '%s' "$out" | grep -q 'yontrack-linux-amd64'; then
    not_ok "$t" "error does not name the missing asset: $out"
else
    ok "$t"
fi
rm -rf "$ws"

t="says so plainly when curl is missing"
ws=$(workspace Linux x86_64)
# A PATH with no curl on it. The check runs before the installer needs any
# other external tool, so nothing else has to be present for this to be fair.
# Set and unset around the call: prefixing a *function* call with an
# assignment leaves it set afterwards, unlike prefixing a command.
TEST_PATH="$ws/fakebin"
run_installer "$ws"
unset TEST_PATH
if [ "$status" -eq 0 ]; then
    not_ok "$t" "exited 0 with no curl available"
elif ! printf '%s' "$out" | grep -q 'curl'; then
    not_ok "$t" "error does not mention curl: $out"
else
    ok "$t"
fi
rm -rf "$ws"

# --- privilege -------------------------------------------------------------

t="never escalates: refuses an unwritable directory and installs nothing"
ws=$(workspace Linux x86_64)
chmod 555 "$ws/target"
run_installer "$ws"
if is_root; then
    skip "$t" "running as root, which can write anywhere"
elif [ "$status" -eq 0 ]; then
    not_ok "$t" "exited 0 against an unwritable directory"
elif [ -e "$ws/target/yontrack" ]; then
    not_ok "$t" "installed into an unwritable directory"
else
    ok "$t"
fi
chmod 755 "$ws/target"
rm -rf "$ws"

t="does not tell anyone to pipe this script into sudo"
ws=$(workspace Linux x86_64)
chmod 555 "$ws/target"
run_installer "$ws"
if is_root; then
    skip "$t" "running as root, which can write anywhere"
elif printf '%s' "$out" | grep -q 'curl.*| *sudo'; then
    not_ok "$t" "told the user to pipe a downloaded script into sudo: $out"
elif ! printf '%s' "$out" | grep -q 'INSTALL_DIR'; then
    not_ok "$t" "no privilege-free alternative offered: $out"
elif ! printf '%s' "$out" | grep -q 'sudo sh install.sh'; then
    not_ok "$t" "no reviewable sudo route offered: $out"
else
    ok "$t"
fi
chmod 755 "$ws/target"
rm -rf "$ws"

t="creates the install directory when it does not exist"
ws=$(workspace Linux x86_64)
rmdir "$ws/target"
run_installer "$ws"
if [ "$status" -ne 0 ]; then
    not_ok "$t" "exited $status: $out"
elif [ ! -x "$ws/target/yontrack" ]; then
    not_ok "$t" "did not create the directory and install into it"
else
    ok "$t"
fi
rm -rf "$ws"

# --- upgrade ---------------------------------------------------------------

t="overwrites an existing installation"
ws=$(workspace Linux x86_64)
printf 'old version\n' > "$ws/target/yontrack"
chmod 755 "$ws/target/yontrack"
run_installer "$ws"
if [ "$status" -ne 0 ]; then
    not_ok "$t" "exited $status: $out"
elif ! grep -q 'fake yontrack' "$ws/target/yontrack"; then
    not_ok "$t" "did not replace the existing binary"
else
    ok "$t"
fi
rm -rf "$ws"

# --- summary ---------------------------------------------------------------

printf '\n%d passed, %d failed\n' "$passed" "$failed"
[ "$failed" -eq 0 ]
