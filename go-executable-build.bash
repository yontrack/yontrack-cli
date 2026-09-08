#!/usr/bin/env bash

version=$1
if [ version == "" ]
then
    version=Snapshot
fi
echo version=$version

package_name="yontrack"

artifacts=()

platforms=("windows/amd64" "darwin/amd64" "darwin/arm64" "linux/amd64" "linux/arm64" "linux/386")

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    output_name=$package_name'-'$GOOS'-'$GOARCH
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi

    env GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-X yontrack/config.Version=$version" -o $output_name $package
    if [ $? -ne 0 ]; then
        echo 'An error has occurred! Aborting the script execution...'
        exit 1
    fi

    artifacts+=("$output_name")
done

# Checksums over exactly the artifacts built above, in the format both
# `sha256sum -c` (Linux) and `shasum -a 256 -c` (macOS) can verify. The
# installer refuses to install anything this file does not cover.
checksums_name="checksums.txt"
rm -f "$checksums_name"

if command -v sha256sum > /dev/null 2>&1; then
    sha256sum "${artifacts[@]}" > "$checksums_name"
elif command -v shasum > /dev/null 2>&1; then
    shasum -a 256 "${artifacts[@]}" > "$checksums_name"
else
    echo 'Neither sha256sum nor shasum is available; cannot write checksums.'
    exit 1
fi

echo "wrote $checksums_name for ${#artifacts[@]} artifacts"

