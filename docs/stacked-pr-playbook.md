# Stacked PR Playbook (for Large PRs)

## Goal

Turn one large PR into a reviewable stack with clear scope, fast feedback, and low merge risk.

## Core Principles

1. Slice by behavior boundaries, not by file type.
2. Keep each PR to one review story.
3. Preserve stack order and dependency flow.
4. Treat review comments as a tight fix-and-reply loop.
5. Keep diffs clean as parents merge.
6. Validate with concrete evidence, not claims.

## How to Split a Large PR

1. Identify layers that naturally depend on each other.
   - Example: core types/parser -> runtime -> normalization -> orchestration/CLI -> logging/docs.
2. Create one PR per layer.
3. Keep each PR independently understandable.
4. Keep refactors out unless required by that layer.

## Branch and PR Structure

Use a chain:

- PR1: `feature/01-*` -> `main`
- PR2: `feature/02-*` -> `feature/01-*`
- PR3: `feature/03-*` -> `feature/02-*`
- ...

This keeps each review focused on incremental change.

## PR Description Standard

For every PR in the stack, use the repository template at:

- `.github/pull_request_template.md`

Do not free-form PR descriptions. Fill the template consistently.

## Review Comment Workflow

For each comment:

1. Read exact thread context (path + line).
2. Decide if it needs:
   - code change, or
   - rationale only.
3. If code change:
   - implement minimal fix,
   - run relevant validation,
   - push commit,
   - reply in-thread with what changed and commit hash.
4. If rationale only:
   - reply in-thread with a short, concrete explanation.

Keep all responses in the same conversation thread for easy resolve.

## Keeping the Stack in Sync

When parent PR merges:

1. Rebase child branch onto updated `main` (or updated parent).
2. Drop duplicate commits if their patch is already upstream.
3. Force-push safely with `--force-with-lease`.
4. Retarget child PR base to `main` when appropriate.

Notes:

- GitHub does not auto-retarget bases after parent merge.
- Cherry-picks can create duplicate patches with different SHAs; clean them during rebase.

## Validation Expectations per PR

- Run compile/tests relevant to the layer.
- Add one concrete functional check for behavior touched by that PR.
- For logging/observability changes, include proof from the log file and line reference when possible.

## Merge Order

Merge from bottom to top of the stack:

1. PR1 -> 2. PR2 -> 3. PR3 -> ...

After each merge, sync the next PR before continuing.

## Quick Checklist

- [ ] PRs are behavior-sliced and small
- [ ] Each PR uses `.github/pull_request_template.md`
- [ ] Comments are answered in-thread
- [ ] Fix replies include commit hash when code changed
- [ ] Child branches rebased after parent merge
- [ ] Duplicate patches dropped during rebase
- [ ] Next PR base retargeted correctly
- [ ] Validation evidence captured in PR body
