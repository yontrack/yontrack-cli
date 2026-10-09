---
name: release-issues
description: Close out the issues a Yontrack CLI release shipped — move every status:ready issue to status:released, comment "Available in <version>", and close it. Lists them for approval first. Use after a release tag has been cut and its pipeline is green.
user-invocable: true
---

# /release-issues — Close Out the Issues a Release Shipped

Arguments passed: `$ARGUMENTS`

`$ARGUMENTS` is the version that shipped — a plain `MAJOR.MINOR.PATCH` tag, no `v` prefix, e.g.
`5.4.2`. If it is missing, or is not that shape, ask for it. Never guess it from the issue list, and
never invent one.

This skill runs **after** a release: the tag is pushed, `.github/workflows/tag.yml` has finished, and
the GitHub release exists. It does not cut releases — that is [`RELEASING.md`](../../../RELEASING.md).

---

## Step 1 — Check the version really shipped

```bash
git fetch --tags
git tag -l "$VERSION"
gh release view "$VERSION" --json tagName,publishedAt,isDraft --jq '"\(.tagName) published=\(.publishedAt) draft=\(.isDraft)"'
```

If the tag does not exist, or the release is a draft, **stop and say so**. Marking issues
`status:released` against a version nobody can install is the one mistake this skill can make that
the operator cannot see afterwards.

If the operator insists the release is real anyway — a tag pushed moments ago, a run still finishing —
say what you found, ask once, and proceed on their answer.

---

## Step 2 — Collect the issues

```bash
gh issue list --state open --label "status:ready" --limit 100 \
  --json number,title,labels
```

`status:ready` means: landed on `main`, CI green, not yet in anyone's hands. Those are exactly the
issues a release ships.

If the query returns nothing, say so plainly — no open issue carries `status:ready` — and stop. Do not
widen the query to `status:wip`, to closed issues, or to a milestone on your own.

---

## Step 3 — Cross-check against what the tag actually contains

An issue at `status:ready` landed on `main`, but a release only ships what was on `main` **when the
tag was cut**. Anything that landed afterwards is ready for the *next* release, not this one.

```bash
PREV=$(git tag -l --sort=-v:refname | grep -A1 -x "$VERSION" | tail -1)
git log --format=%s "$PREV".."$VERSION" | grep -oE '^#[0-9]+' | tr -d '#' | sort -un
```

Compare that list of issue numbers with Step 2's:

- **in both** → ships in this version; include it
- **`status:ready` but not in the tag range** → it landed after the tag. **Exclude it** and say so
  by name — it belongs to the next release
- **in the tag range but not `status:ready`** → it is already released, or was never labelled. Report
  it; do not touch it

---

## Step 4 — Submit for approval — STOP HERE

Present the issues that will be touched as a table, then **wait**. Do not edit a label, post a
comment, or close anything until the operator has approved.

| # | Title | Labels |
|---|-------|--------|
| 69 | Add a `validate number` subcommand | enhancement, status:ready, initiative: security-scans |

Alongside the table give:
- the version, and the tag range you compared against
- the issues you **excluded** and why (landed after the tag; in the range but not `status:ready`)
- anything that looks off: an issue with no `feature` / `enhancement` / `bug` label — that is what
  lands work in the changelog's `Other` group, and it is worth fixing *before* closing the issue

Then ask for approval, and accept any of these answers:
- approve the whole list
- approve a **subset** — drop the issues the operator names, leave them untouched at `status:ready`
- reject — stop, having changed nothing

---

## Step 5 — Apply, one issue at a time

For each approved issue, in number order, read where it stands first:

```bash
gh issue view {number} --json state,labels,comments --jq '
  "state=\(.state)",
  "labels=\([.labels[].name] | join(","))",
  "commented=\([.comments[].body | select(. == "Available in '"$VERSION"'")] | length)"'
```

Then run the three steps **in this order**, each only if it is not done yet. The comment goes on
before the close, so the notification a subscriber gets carries the version.

1. **Labels**, building the one `gh issue edit` from what the issue carries:
   - `--add-label "status:released"` only if it does not carry `status:released`
   - `--remove-label "status:ready"` only if it carries `status:ready`
   - neither needed → skip the edit

   ```bash
   gh issue edit {number} --add-label "status:released" --remove-label "status:ready"
   ```

   GitHub fails the whole edit with a generic `GraphQL: Something went wrong` when it is asked to
   remove a label the issue does not carry, so never pass `--remove-label` blind.
2. **Comment**, only if `commented=0`:

   ```bash
   gh issue comment {number} --body "Available in $VERSION"
   ```
3. **Close**, only if `state=OPEN`:

   ```bash
   gh issue close {number}
   ```

- An issue carries exactly **one** `status:*` label, so when both `--add` and `--remove` are needed
  they go in the same command — never two.
- The comment body is exactly `Available in <version>`, no decoration, no trailer, no link. It is a
  marker people grep for.
- Close with no `--reason`; these are completed, which is `gh`'s default.

Checking first is what makes a run safe to repeat: re-running it on a half-done issue finishes it,
and an issue already done is left alone.

If any one of the three fails for an issue, **stop the whole run**. A failed `gh issue edit` can still
have applied part of the change, so re-read the issue with the `gh issue view` above and report what
it actually carries, not which commands succeeded. Do not carry on and leave a trail of issues in
mixed states.

---

## Step 6 — Verify, then report

Do not trust the commands' exit codes alone:

```bash
for n in {numbers}; do
  gh issue view $n --json number,state,labels \
    --jq '"#\(.number) \(.state) [\([.labels[].name] | join(", "))]"'
done
```

Every line must read `CLOSED` and carry `status:released` and not `status:ready`. Then give the
operator one table:

| # | Title | Labels | State |
|---|-------|--------|-------|

and below it: the issues you excluded in Step 3 and why, and any missing type label you flagged in
Step 4 that the operator may still want to fix.

---

## Guardrails

- **Never** invent the version — it comes from `$ARGUMENTS` or from the operator
- **Never** touch an issue that is not `status:ready`, and never reopen or relabel a closed one. The
  one exception is an issue the operator approved in this run and a failed step left half-done: it
  may have lost `status:ready` already, and Step 5 finishes it once the operator says to retry
- **Never** close an issue whose comment or label step failed
- **Never** widen the query past open `status:ready` issues
- **Never** cut a tag, push, or edit a release from this skill — see [`RELEASING.md`](../../../RELEASING.md)
- **Never** hand-write or edit a changelog — `tag.yml` generates it from Yontrack
