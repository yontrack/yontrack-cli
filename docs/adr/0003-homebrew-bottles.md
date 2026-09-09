# The Homebrew formula is bottled, for Apple Silicon only

The tap publishes one bottle, tagged `arm64_ventura`, built on the newest
macOS runner. Intel Macs and Linux get no bottle and install exactly as they
did before. All three halves of that are deliberate.

## Why bottle at all

Homebrew treats a formula with no bottle as a source build, and runs its fatal
build-from-source checks before it starts:

```ruby
unless pour_bottle?
  Homebrew::Install.perform_build_from_source_checks
end
```

On macOS that list is `check_for_installed_developer_tools`,
`check_clt_minimum_version`, `check_xcode_minimum_version` and friends. So
installing demanded present, current Command Line Tools in order to copy one
static binary — and refused, with *"Your Command Line Tools are too outdated"*,
on a machine that had them but had not refreshed them after a macOS upgrade.
None of those checks have anything to do with what this formula does. Pouring
a bottle skips all of them.

## Why the tag is older than the machine that builds it

`find_older_compatible_tag` in `extend/os/mac/utils/bottles.rb` picks the
newest bottle at or below the running system, skipping candidates whose
architecture differs. A bottle tagged for the running runner would cover that
macOS version and later ones only, and would silently stop covering the
current release the moment GitHub moved the runner image on. An old tag covers
everything above it, so one bottle serves every supported Apple Silicon Mac.

This is honest only because the payload is a static Go binary that links
against nothing from an SDK. A formula that actually compiled would be lying
about what it was built against, and the bottle would break in ways the tag
promised it would not. Verified rather than assumed: on a macOS 26 machine
running `arm64_tahoe`, a formula declaring only `arm64_ventura` resolves to it
and reports `pour_bottle?` true.

## Why no Intel bottle

Every x86_64 macOS runner label — `macos-26-intel`, `macos-15-intel`,
`macos-*-large`, `macos-latest-large` — is a
[larger runner](https://docs.github.com/en/actions/reference/runners/larger-runners),
which is billed. Bottling Intel would put a billed job in every release, for a
shrinking population, to fix a papercut.

Intel Macs therefore keep the old behaviour: Homebrew takes the source path
and asks for current Command Line Tools. That is a graceful degradation, not a
regression — it is what every platform did before this — and the README says
so. If the Intel population ever justifies the spend, adding a second entry to
the matrix in `bottle.yml` is the whole change.

## Why no Linux bottle

Linux does not have this problem. The generic `fatal_build_from_source_checks`
is a single entry:

```ruby
def fatal_build_from_source_checks
  %w[
    check_for_installed_developer_tools
  ].freeze
end
```

`check_clt_minimum_version` and `check_xcode_minimum_version` live in the macOS
override and nowhere else. Anyone running Homebrew on Linux already has a
compiler, because Homebrew on Linux requires one to exist at all. A Linux
bottle would buy nothing.

It would also cost something: the hosted Ubuntu images no longer ship
Homebrew — `command -v brew` fails on `ubuntu-latest` — so bottling Linux
would mean installing Homebrew from scratch on two runners per release.

## The tag has to stay a macOS version Homebrew knows

`find_older_compatible_tag` compares with `to_macos_version`, and swallows the
failure:

```ruby
candidate.to_macos_version <= tag_version
rescue MacOSVersion::Error
  false
```

So when Ventura eventually ages out of Homebrew's known versions, the bottle
will not error — it will be *ignored*, and every Apple Silicon install will
quietly go back to the source path and the Command Line Tools check. Nothing
fails loudly when that happens.

The `Homebrew` job in `go.yml` installs from the tap on every push, so the
symptom is reachable, but it will show up as a slow install rather than a red
build. If bottles ever stop being poured for no apparent reason, this is the
first thing to check, and the fix is to move the tag in `bottle.yml` up to the
oldest macOS Homebrew still supports.

## What would change the answer

An Intel bottle, if larger-runner minutes stop mattering or the population
justifies them. Or Homebrew making the source path cheap on macOS, which would
remove the reason for any of this.
