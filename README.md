# PR Review Bot (Phase 0)

Phase 0 implements:
- Input: GitHub PR URL
- Output: JSON classification (`human_required` or `no_human`)
- Output: JSON first-pass review (`ready_to_merge`, `wip`, or `needs_human_review`)

Architecture:
- Stage 1 (inside Podman): `opencode` (Opus 4.6) reviews checked-out PR
- Stage 2 (in Go process): Haiku normalizes stage-1 output to structured JSON
- Failure policy: any error/timeout/low confidence => `human_required`

## Prerequisites

- `podman`
- read-only `GITHUB_ACCESS_TOKEN`
- Google ADC at `~/.config/gcloud/application_default_credentials.json`
- env vars:
  - `GOOGLE_CLOUD_PROJECT`
  - `CLOUD_ML_REGION`

The repository-scoped OpenCode config lives in `.config/opencode`.

## Build stage-1 image

```bash
./scripts/build_stage1_image.sh pr-review-bot-stage1:latest
```

## Run classifier

```bash
PR_REVIEW_BOT_STAGE1_IMAGE=pr-review-bot-stage1:latest \
./bin/classify-pr "https://github.com/RohanAwhad/new-math-mnist/pull/9"
```

If `PR_REVIEW_BOT_STAGE1_IMAGE` is not set, the CLI falls back to `STAGE1_IMAGE`, then `pr-review-bot-stage1:latest`.

## Run first-pass review

```bash
PR_REVIEW_BOT_STAGE1_IMAGE=pr-review-bot-stage1:latest \
./bin/first-pass "https://github.com/RohanAwhad/new-math-mnist/pull/9"
```

First-pass output shape:

```json
{
  "intent_understanding": {
    "verdict": "yes",
    "confidence": 0.92,
    "reason": "Intent is explicit in commit and diff history",
    "understood_intent": "Migrate project to package-first API and update docs/tests accordingly"
  },
  "optimality": {
    "verdict": "acceptable",
    "reason": "Approach is mostly sound but leaves one follow-up refactor",
    "alternatives": ["Move dataset loader to dedicated module in this PR"]
  },
  "merge_readiness": "wip",
  "focus_areas": [
    {
      "path": "new_math_ops/dataset.py",
      "why": "Still re-exports from evaluate module",
      "priority": "medium"
    }
  ],
  "blocking_questions": [
    "Is removing top-level compatibility shims approved for this release?"
  ],
  "run_id": "run-1741740000-123456"
}
```

Classifier output shape:

```json
{
  "classification": "human_required",
  "confidence": 0,
  "reason": "stage-2 confidence too low: 0.42",
  "run_id": "run-1741740000-123456"
}
```

## Run smoke tests

```bash
./scripts/smoke_phase0.sh
```

This runs 3 classification smoke cases in parallel, captures each output separately, then prints section-wise results.

By default, temporary smoke outputs are deleted after the run. Keep them for debugging with:

```bash
KEEP_SMOKE_LOGS=1 ./scripts/smoke_phase0.sh
```

## Notes

- Stage-1 timeout is fixed at 30 minutes.
- Confidence threshold defaults to `0.5` and can be overridden with `MIN_CONFIDENCE`.
- Normalizer model defaults to `claude-haiku-4-5@20251001` and can be overridden with `NORMALIZER_MODEL`.
- Stage-1 model can be overridden with `STAGE1_MODEL` (otherwise OpenCode config default is used).
- Logs are written to `logs/classify-pr.log` and `logs/first-pass.log`, mirrored to stderr.
- `LOGGING_LEVEL` controls verbosity (`debug`, `info`, `warn`, `error`); default is `warn`.
