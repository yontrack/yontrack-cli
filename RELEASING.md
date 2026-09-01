Releasing
=========

A release is cut by pushing a tag. Everything after that is automated by
[`.github/workflows/tag.yml`](.github/workflows/tag.yml): the binaries are
built, the changelog is generated from Yontrack, the GitHub release is created
with that changelog as its body, and the build is validated and labelled in
Yontrack.

So the whole job is: **pick the right version, check the changelog will read
well, push the tag.**

## 1. Pick the version

Tags are plain `MAJOR.MINOR.PATCH`, no `v` prefix — `5.1.1`, `5.2.0`.

Look at what has landed since the last tag:

```bash
git fetch --tags
LAST=$(git tag -l --sort=-v:refname | head -1)
git log --oneline "$LAST"..main
```

Every commit is expected to start with the issue it closes (`#46 Add ...`), so
the issue labels decide the bump:

| Anything in the range is… | Bump  |
|---------------------------|-------|
| breaking                  | major — and stop, this is a human decision |
| `feature` or `enhancement` | minor |
| only `bug`                | patch |

Additive changes are a **minor**: a new command, or a new flag on an existing
one, is still a minor even though nothing broke. A patch is for fixes only.

To read the labels of everything in the range:

```bash
git log --format=%s "$LAST"..main \
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

The release body is produced by Yontrack, from the commits between the last
build promoted to `RELEASE` and the one being released. To see it before
committing to a tag, against a CLI already configured for the instance:

```bash
yontrack build changelog export \
  --from-promotion RELEASE \
  --to <build-id> \
  --format markdown \
  --grouping "Features=feature|Enhancements=enhancement|Bugs=bug" \
  --alt-group "Other"
```

`<build-id>` is the Yontrack build for the commit being released — the same
lookup `tag.yml` does with `build search --commit`.

An empty or thin changelog almost always means commits did not reference their
issues, not that nothing changed.

## 4. Push the tag

```bash
git checkout main
git pull --ff-only
git tag 5.2.0
git push origin 5.2.0
```

Then watch the release build:

```bash
gh run list --workflow=tag.yml --limit 1
gh run watch <run-id>
```

## What the tag workflow does

For the record, so the manual steps above are not reinvented:

1. Builds every platform binary via `go-executable-build.bash <version> release`
2. Configures the CLI against the Yontrack instance and finds the build for the
   tagged commit
3. Exports the changelog since the last `RELEASE`-promoted build
4. Creates the GitHub release, with that changelog as the body and the binaries
   attached
5. Validates `GITHUB.RELEASE` on the build and sets its `release` property

Steps 2 to 5 need `vars.ONTRACK_URL` and `secrets.ONTRACK_TOKEN`.

## After the release

Close the issues the release ships, and note the version on them. The GitHub
release itself is created by the workflow — there is nothing to write by hand.
