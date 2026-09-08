# No macOS notarization

The binaries this project publishes are not signed with an Apple Developer ID
and are not notarized, and we do not intend to change that. This is a decision,
not an omission — a reader who meets the Gatekeeper dialog and assumes nobody
got round to signing would otherwise be right to open a pull request adding an
Apple certificate to CI.

## What actually causes the dialog

The dialog — *"Apple could not verify 'yontrack' is free of malware"* — is
triggered by the `com.apple.quarantine` extended attribute, which the **browser**
attaches at download time. It is not a property of the binary. Per Apple DTS in
[Resolving Trusted Execution Problems](https://developer.apple.com/forums/thread/706442),
"Unix-y networking tools, like `curl` and `scp`, don't quarantine the files they
download", and `tar` does not propagate it either.

So `install.sh` does not work around Gatekeeper. It removes the condition that
invokes it. That is why the supported install path never meets the dialog and
needs no Apple relationship to do so.

## Ad-hoc signing is already happening and does not help

Go's linker ad-hoc signs `darwin/arm64` output automatically since 1.16
([golang/go#42684](https://github.com/golang/go/issues/42684)), including when
cross-compiling from Linux, which is what this project's CI does. That satisfies
Apple Silicon's requirement that arm64 binaries carry *a* signature. It has no
bearing on a Gatekeeper assessment, which needs a Developer ID signature **and**
a notarization ticket. Running `codesign -s -` would add nothing.

## What notarization would cost, and buy

Apple Developer Program membership at $99/year, a Developer ID Application
certificate and an App Store Connect API key held as CI secrets, and a signing
step — `quill`, since GoReleaser's native path is Pro-only and `gon` has been
archived and unmaintained since 2022.

It would fix one thing: downloading an asset from the releases page in a
browser. Two limits on even that. A notarization ticket
[cannot be stapled to a bare Mach-O binary](https://developer.apple.com/forums/thread/689337)
("Stapling is not supported for mach-O binaries"), so a first run while offline
still alerts. And double-clicking the binary in Finder fails a separate document
check no matter how well signed it is.

## Why declined

It buys an improvement only to an installation route we no longer recommend, at
a recurring cost, and it means holding Apple signing credentials in CI. `gh`,
`kubectl` and `golangci-lint` all made the same call — `gh`'s macOS `.pkg` files
are [openly unsigned](https://github.com/cli/cli/issues/9139). The manual
download path is documented in the README with the one `xattr` command that
clears the attribute, which is honest about what it does.

## What would change the answer

Shipping a `.pkg` or `.dmg`, which are stapleable — and a `.pkg` install leaves
its files unquarantined entirely. Or the releases page becoming the primary
install route again, which would mean the installer had failed at its job.
