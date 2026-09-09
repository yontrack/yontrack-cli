# The Homebrew formula ships the release binary

`yontrack/homebrew-tap` installs the binary published with the GitHub release
and pins its sha256. It does not build from source. Homebrew's own guidelines
point the other way for Go programs, so a reader following those guidelines
would otherwise be right to open a pull request adding `depends_on "go"`.

## What the guideline is actually for

The preference for building Go from source is written for **homebrew-core**,
where the maintainers reviewing a formula are not the people publishing the
software. Building from a source tarball is what lets them audit what they
ship, because they have no reason to trust an upstream's CI.

A tap owned by the same project inverts that. The formula, the release workflow
and the binary come from one place; building from source would not add a party
to trust, it would only change which artifact that party hands over.

## What building from source would cost

Homebrew would pull the whole Go toolchain as a build dependency to compile a
single static binary, turning a several-second install into a several-minute
one and several hundred megabytes of download for a 15MB program.

It would also mean `brew` delivers different bytes from `install.sh` and from
the releases page. The binary in the release is the one CI built, checksummed
and validated in Yontrack; a locally compiled one has been through none of
that. One artifact everywhere is worth more here than a build we cannot observe.

## Why the hash is enough

[`checksums.txt`](../../go-executable-build.bash) is published with every
release, and the formula pins the same hashes `install.sh` verifies against. A
tampered asset fails `brew install` exactly as it fails the installer.

## This is unrelated to Gatekeeper

It is easy to assume the binary is downloaded to dodge signing. It is not.
Homebrew quarantines **casks** — `Library/Homebrew/extend/os/mac/cask/quarantine.rb`
calls LaunchServices with `kLSQuarantineTypeWebDownload` — and does not
quarantine **formulae**, whose downloads go through `curl` and get no xattr. A
formula is unquarantined however it obtains its files, so this choice does not
touch [ADR 0001](0001-no-macos-notarization.md). It is also why the tap ships a
formula and not a cask, which would have to pass Gatekeeper.

## What would change the answer

Wanting the formula in homebrew-core rather than in our own tap. Core requires
building from source for Go, and would not take the four pinned binaries above.
It also has a notability threshold this project does not obviously clear, and
would move release timing to somebody else's review queue.
