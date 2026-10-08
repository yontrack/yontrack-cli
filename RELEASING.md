Releasing
=========

A release is cut by pushing a tag. Everything after that is automated by
[`.github/workflows/tag.yml`](.github/workflows/tag.yml): the binaries are
built, the changelog is generated from Yontrack, the GitHub release is created
with that changelog as its body, the build is validated and labelled in
Yontrack, and the Homebrew formula is bumped in the tap.

So the whole job is: **pick the right version, check the changelog will read
well, push the tag.**

## Release lines

There are two, and the branch decides which one a tag belongs to:

| Line | Branch | What it gets |
|------|--------|--------------|
| 6.x  | `main` | everything: features, enhancements, fixes |
| 5.x  | `v5`   | patches only, `5.9.1`, `5.9.2`… |

`main` is the CLI for Yontrack 6. Commands that exist in 5.x keep working
against a 5.x server; the new ones fail there with the server's GraphQL error.
Nothing checks the server version at run time.

**Patching 5.x.** A fix lands on `main` first, like any issue. If 5.x needs it
too, the issue carries `backport: 5.x`, and its commit is cherry-picked:

```bash
git checkout v5
git pull --ff-only
git cherry-pick -x <commit-on-main>
git push origin v5
```

`-x` notes the original commit; the subject keeps its `#N`, so the changelog
and `release-issues` still find the issue. Once `Go` is green on `v5`, tag it
from `v5` (step 4). Never merge `v5` into `main`, or `main` into `v5`.

**What a 5.x patch does not do.** `tag.yml` places each tag among the others
with [`release_line.sh`](release_line.sh). A tag that is not the highest
`MAJOR.MINOR.PATCH`, which is every 5.x patch once `6.0.0` exists:

- is published, with its changelog and binaries, but **not as GitHub's latest
  release**, which `install.sh` follows;
- **does not touch the Homebrew tap**, which has a single formula: bumping it
  would move brew users back to 5.x. No bottles are built for it either.

5.x users install a patch with the installer and an explicit version:

```bash
curl -fsSL https://raw.githubusercontent.com/yontrack/yontrack-cli/main/install.sh | VERSION=5.9.1 sh
```

Any tag that is not `MAJOR.MINOR.PATCH` fails the release in its first step.

## 1. Pick the version

Tags are plain `MAJOR.MINOR.PATCH`, no `v` prefix — `5.1.1`, `5.2.0`.

Look at what has landed since the last tag of the line (see *Release lines*):

```bash
git fetch --tags
# 6.x, from main
LAST=$(git tag -l --sort=-v:refname | head -1)
git log --oneline "$LAST"..origin/main
# 5.x, from v5
LAST=$(git tag -l '5.*' --sort=-v:refname | head -1)
git log --oneline "$LAST"..origin/v5
```

A 5.x patch takes fixes only: anything else in the range of `v5` was
cherry-picked by mistake.

Every commit is expected to start with the issue it closes (`#46 Add ...`), so
the issue labels decide the bump:

| Anything in the range is… | Bump  |
|---------------------------|-------|
| breaking                  | major — and stop, this is a human decision |
| `feature` or `enhancement` | minor |
| only `bug`                | patch |

Additive changes are a **minor**: a new command, or a new flag on an existing
one, is still a minor even though nothing broke. A patch is for fixes only.

The table is about the **public surface** — the commands, flags and output a
user of the CLI meets. An `enhancement` that touches nothing a user can observe
does not force a minor. 5.4.1 is the precedent: it carried five enhancements
(a glossary, an ADR, shellcheck in CI, a build-artifact check, an installer CI
matrix) and one bug fix, and shipped as a patch because the only change visible
from outside was the bug fix. If you are unsure whether something is visible,
ask what a user would notice differently after upgrading.

To read the labels of everything in the range:

```bash
git log --format=%s "$LAST"..origin/main \
  | grep -oE '^#[0-9]+' | tr -d '#' | sort -u \
  | xargs -I{} gh issue view {} --json number,labels \
      --jq '"#\(.number) [\([.labels[].name] | join(", "))]"'
```

## 2. Check every issue has a type label

The changelog groups issues by the labels `feature`, `enhancement` and `bug`.
An issue carrying none of them lands in the `Other` group — visible, but a sign
that the label is missing rather than a category anyone wants. Fix the label
rather than shipping a release with an `Other` section.

The command above prints the labels of every issue in the range; each one should
carry exactly one of the three.

## 3. Preview the changelog

Do this every time. It is the step that catches issues appearing in the release
notes that were never implemented.

Yontrack collects **every** `#123` in a commit message, not just the one the
subject line starts with. A commit body saying "this blocks #58" enrols #58 in
the changelog, and the notes then claim it shipped. This is how 5.4.0 came to
list three issues that did not exist in it. Write cross-references so they are
not harvested — `yontrack-cli#58`, or just "issue 58" in prose — and keep the
issue the commit closes in the subject line where it belongs.


The release body is produced by Yontrack, from the commits between the build of
the previous tag, the highest `MAJOR.MINOR.PATCH` below the one being released,
and the build being released. So `5.9.1` starts from `5.9.0`, and `6.0.0` from
the highest 5.x tag. When the previous tag has no build in Yontrack, `tag.yml`
warns and falls back to the last `RELEASE` build of the branch. To see it before
committing to a tag, against a CLI already configured for the instance:

```bash
yontrack build changelog export \
  --from <build-id of the previous tag> \
  --to <build-id> \
  --format markdown \
  --grouping "Features=feature|Enhancements=enhancement|Bugs=bug" \
  --alt-group "Other"
```

Both ids are the Yontrack builds of the commits — the same lookup `tag.yml`
does with `build search --commit <sha> --count 1 --display-id`.

An empty or thin changelog almost always means commits did not reference their
issues, not that nothing changed.

## 4. Push the tag

From the branch of the line, `main` for 6.x or `v5` for a 5.x patch:

```bash
git checkout main
git pull --ff-only
git tag 6.1.0
git push origin 6.1.0
```

Then watch the release build:

```bash
gh run list --workflow=tag.yml --limit 1
gh run watch <run-id>
```

## What the tag workflow does

For the record, so the manual steps above are not reinvented:

1. Builds every platform binary via `go-executable-build.bash <version>`, and the `checksums.txt` the installer verifies against
2. Configures the CLI against the Yontrack instance and finds the build for the
   tagged commit
3. Exports the changelog since the build of the previous tag
4. Creates the GitHub release, with that changelog as the body and the binaries
   attached; it is marked as the latest release only for the highest tag
5. Validates `GITHUB.RELEASE` on the build and sets its `release` property
6. For the highest tag only (see *Release lines*), builds the Homebrew bottle
   for Apple Silicon, checks it, and uploads it to the release
   ([`bottle.yml`](.github/workflows/bottle.yml))
7. For the highest tag only, renders the Homebrew formula with `homebrew_formula.sh` and pushes it to
   [`yontrack/homebrew-tap`](https://github.com/yontrack/homebrew-tap), which is
   what makes `brew install yontrack/tap/yontrack` offer the new version

Steps 2 to 5 need `vars.YONTRACK_URL` and `secrets.YONTRACK_TOKEN`. Step 7
needs `secrets.HOMEBREW_TAP_TOKEN`.

Step 6 failing does not hold up step 7: the formula is pushed unbottled, which
is what every release before 5.5.0 shipped and installs fine. It does mean
Apple Silicon users take Homebrew's source path until the next release, so a
`No bottle manifest` warning in the run is worth chasing rather than ignoring.

`bottle.yml` can be run by hand against any published tag from the Actions tab,
which is how to test a change to it without cutting a release. Leave `publish`
unticked and it builds and checks the bottle without uploading anything.

## The Homebrew tap token

`secrets.GITHUB_TOKEN` is scoped to this repository and cannot push to another
one, so the formula bump needs its own credential: a fine-grained personal
access token, `Contents: read and write` on `yontrack/homebrew-tap` and nothing
else, held here as `HOMEBREW_TAP_TOKEN`.

Fine-grained tokens expire. When one does, the release itself still succeeds —
the tap step runs last for that reason — and the workflow then fails with the
formula left at the previous version. Mint a new token, set the secret, and
re-run the failed job; the step is safe to repeat, and does nothing at all if
the formula is already current.

To see what the next release would publish, without releasing anything:

```bash
gh release download <last tag> -p checksums.txt
./homebrew_formula.sh <next version> checksums.txt
```

The hashes will be the previous release's, so this shows the shape rather than
the content. `./homebrew_formula_test.sh` is what checks the shape is right.

## After the release

Close the issues the release ships, and note the version on them. The GitHub
release itself is created by the workflow — there is nothing to write by hand.
