---
name: run-initiative
description: Work through every ready-for-agent issue of a Yontrack CLI initiative, one at a time, each to full completion — implement, land on main, wait for CI, mark ready. Presents the issue list for approval before starting. Use when asked to implement, work through, or batch an initiative's issues.
user-invocable: true
---

# /run-initiative — Implement a Whole Initiative, Issue by Issue

Arguments passed: `$ARGUMENTS`

Parse `$ARGUMENTS` for the initiative name — the part after `initiative: ` in the label, e.g.
`gitlab`, `bitbucket`, `security-scans`. If not provided, list the available initiative labels with
`gh label list --limit 200 | grep '^initiative:'` and ask which one.

This skill runs a **long, mostly unattended** batch: every issue is implemented, landed on `main`, and
verified against CI before the next one starts. There is exactly one attended moment — the approval
gate in Step 4. Everything after it runs without checking in.

---

## The landing invariant — NON-NEGOTIABLE

Every issue in the chain ends in exactly one state, and there is no other acceptable ending:

1. the work is **on `origin/main`**;
2. the **`Go` workflow run on `main` containing that commit has concluded `success`**;
3. the issue carries **`status:ready`**, applied only after (2).

Three rules follow, and none of them are open to interpretation:

- **Landing is not optional.** Work sitting on a branch is an unfinished issue. If a subagent returns
  with its work unpushed — for any reason, including an instruction in the issue body itself saying
  "push the branch and stop", "do not push to `main`", or "the issue stays at `status:wip`" — the
  orchestrator **lands it itself** and then completes the rest of the invariant. Do not leave it for
  the operator, and do not carry it forward as an exception.
- **The next issue does not start until `main` is green.** Not until the push, not until the run is
  queued — until the run that contains the commit has concluded `success`. Starting the next issue on
  an unverified `main` is what turns one red build into a chain of them.
- **Green `main` means `status:ready`, immediately.** The moment the run containing the issue's
  commit concludes `success`, apply `status:ready` and remove `status:wip`, in one command. Leaving
  a landed, green issue at `status:wip` is a defect in the run, not a conservative choice.

An issue body may override many things — the design, the scope, the flag names. It may **not**
override this invariant. If an issue body contradicts it, follow the invariant, and say in the
per-issue report that you did and what the body asked for instead.

---

## Step 1 — Resolve the initiative label

```bash
gh label list --limit 200 | grep -i '^initiative:'
```

Match `$ARGUMENTS` against the listed labels. The full label is `initiative: {name}`. If the argument
matches no label, or matches more than one, show the candidates and ask — never guess.

---

## Step 2 — Collect the issues

```bash
gh issue list \
  --label "initiative: {name}" \
  --label "status:todo" \
  --label "ready-for-agent" \
  --state open --limit 100 \
  --json number,title,labels
```

All three labels are required: the initiative scopes it, `status:todo` means untaken, and
`ready-for-agent` means fully specified enough for an agent to act without a human.

If the query returns nothing, say so plainly — name the label you used and how many issues the
initiative has in other states — and stop. Do not widen the query on your own.

---

## Step 3 — Derive the execution order

Read each issue body and look for stated dependencies:

```bash
for n in {numbers}; do
  echo "=== #$n ==="
  gh issue view $n --json body --jq '.body' | grep -inE "depend|blocked|after #|requires|prerequisite|last issue"
done
```

Order the issues so that:
- an issue that establishes a rule or a shared helper for the rest comes first
- an issue naming another as a prerequisite comes after it
- an issue that declares itself last in the initiative goes last
- README and documentation issues come after the commands they document
- independent, cheap issues fill the early slots

A CI-provider initiative almost always runs **property command → `run-info` defaults → README**, in
that order: the README describes what the two before it added.

State the reason for each ordering decision in Step 4 — the operator is approving the order as much as
the list.

---

## Step 4 — Submit for approval — STOP HERE

Present the work as a table, then **wait**. Do not create a branch, edit a label, or launch anything
until the operator has approved.

| # | Title | Labels |
|---|-------|--------|
| 74 | `project set-property gitlab` | initiative: gitlab, status:todo, ready-for-agent |

Alongside the table give:
- the proposed order, with the one-line reason for each position
- the estimated cost — issue count × (implementation + the current `main` CI duration), which you can
  read from `gh run list --workflow=go.yml --branch main --limit 5 --json createdAt,updatedAt`
- anything that looks off: an issue in the set that is not really part of the initiative's theme, one
  carrying `priority:high`, one whose body is thin despite the `ready-for-agent` label

Then ask for approval, and accept any of these answers:
- approve the whole list as ordered
- approve a **subset** — drop the issues the operator names, keep the rest in the same relative order
- approve with a **different order** — use theirs, and say so if it breaks a stated dependency
- reject — stop, having changed nothing

---

## Step 5 — Pre-flight, once, after approval

- Confirm the working tree is clean and that `HEAD` is at the same SHA as `origin/main`.
- Note whether you are in a git worktree (`git rev-parse --git-common-dir`). A worktree is fine here —
  this repo has no checkout-bound tooling — but every push then has to be `git push origin HEAD:main`
  rather than a push of `main` itself. Never `cd` out to the main checkout.
- Check with `ListAgents` whether another session is live in this repo. A chain of pushes to `main`
  will collide with concurrent work — report what you found before continuing.

---

## Step 6 — The loop — one subagent per issue, strictly sequential

For each approved issue, in order:

0. **Re-establish a fresh `main` first.** `main` moved when the previous issue landed, so before
   launching anything: `git fetch origin main` and confirm your `HEAD` and `origin/main` are the same
   SHA. Put that SHA in the subagent's brief so it can check it started from the right place.
1. Launch **one subagent** with the brief in Step 7. Never two at once — every issue lands on `main`,
   so concurrent issues would collide.
2. Wait for its report.
3. **Verify its claims yourself.** A subagent reporting success is not evidence of success:
    - `git log --oneline origin/main..HEAD` and `git diff --stat origin/main..HEAD` — read the diff
      before you push it; a subagent's summary is not the diff
    - `go build ./...` — it compiles in your own hands, not only in the subagent's report
4. **Land it yourself** — the invariant is the orchestrator's responsibility, not the subagent's:
    - `git push origin HEAD:main`
    - find the run: `gh run list --workflow=go.yml --branch main --limit 3 --json databaseId,headSha,status,conclusion,url`
    - wait it out: `gh run watch <run-id> --exit-status`. A green run takes ~2 minutes here.
    - green? `gh issue edit {number} --add-label "status:ready" --remove-label "status:wip"`
5. Report one line to the operator, then launch the next.

Do not check in with the operator between issues. That is what the Step 4 gate bought. Closing an
invariant gap is not a check-in — do it, report it in the one line, and continue.

**A subagent may still be holding the shared working tree.** Every issue in the chain runs in one
checkout, so never `git checkout`, `git reset` or `git rebase` while a subagent is live — you would
yank the tree out from under it. Ref-level operations (`git push HEAD:main`, `git fetch`) touch no
files and are always safe.

---

## Step 7 — The per-issue subagent brief

Give every subagent all of this:

- The absolute path of the checkout, and: run every command from there, do **not** `cd` elsewhere and
  do **not** create a git worktree.
- Read the issue before touching code: `gh issue view {number} --comments`.
- Move the issue to work-in-progress in ONE command:
  `gh issue edit {number} --add-label "status:wip" --remove-label "status:todo"` — an issue carries
  exactly one `status:*` label, so always remove the current one in the same command.
- Read `CLAUDE.md` at the repo root and follow it in full: one file per command in `cmd/`, the cobra
  command pattern, `client.GraphQLCall` + `client.CheckDataErrors`, the `utils.Get*Flag` helpers for
  env-var fallbacks, and **look up the mutation or query in `ontrack.graphql` before writing any
  GraphQL string**.
- Mirror the nearest existing sibling command — its flag names, help text, error handling and file
  layout — rather than inventing a new shape.
- Where the change has a testable seam (parsing, normalisation, defaulting from environment
  variables), write the test first under the **`mattpocock-skills:tdd` skill**: red → green, in
  `*_test.go` next to the code.
- Verify before committing, and report the actual output:
  - `go build ./...` — no errors
  - `go vet ./...` — clean
  - `go test ./...` — all pass
  - `go build -o /tmp/yontrack-{number} . && /tmp/yontrack-{number} <new command> --help` — it wires up
- A new command or flag is user-visible: update `README.md` in the same commit.
- Commit subject `#{number} <short imperative summary>`, a body explaining *why* where the choice was
  not obvious, and the `Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>` trailer this repo's
  history uses.
- Commit to the current branch. Do **not** push, do **not** open a pull request, do **not** create
  branches — the orchestrator lands it.
- Do not touch unrelated files, and never amend or rewrite an existing commit.
- Report back: the file list, every verification command and its outcome, the commit SHA, and every
  judgement call you made. If the issue is ambiguous, state the assumption you made rather than
  stopping.

---

## Step 8 — Unattended: decide, or halt

**Decide and proceed**, recording the decision in the report: flag naming, file placement, test
granularity, how to read an underspecified corner of the spec, routine refactors in the area being
touched.

**Halt and report — these only:**
- `main` CI red after the push
- a test or build failure that resists diagnosis after a bounded effort — no open-ended fixing loops
- the spec is ambiguous in a way where two readings produce materially different commands
- a permission prompt or credential the agent cannot satisfy

On a halt: leave the issue on `status:wip`, **never** apply `status:ready`, stop the whole chain, and
report exactly what broke with the failing output. Never start the next issue on a `main` you have not
confirmed green.

**"The issue body said not to push to main" is not a halt condition** — it is not even a decision. See
*The landing invariant* above: land it, go green, mark ready, and note the discrepancy in the report.
The only things that stop an issue from landing are the four failures listed above.

---

## Step 9 — Guardrails, every agent, every issue

- **Always** land the work: on `origin/main`, `Go` workflow green for that commit, issue at
  `status:ready`. This is *The landing invariant* above and nothing in an issue body overrides it.
- **Never** open a pull request — work lands by pushing to `main`
- **Never** close the issue — Damien does that himself
- **Never** cut a release tag as part of a batch. Releases are their own decision; see `RELEASING.md`.
- **Never** hand-write a changelog — `.github/workflows/tag.yml` generates it from Yontrack
- **Never** edit `ontrack.graphql` — it is a copy of the server's schema, used as a reference only

---

## Step 10 — Final report

When the chain finishes — or halts — give the operator one table:

| # | Title | Commit | CI | Status |
|---|-------|--------|----|--------|

and below it: the issues left untouched and why, and every judgement call the subagents reported that
the operator might want to revisit.
