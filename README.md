# PR Review Bot (Phase 0)

Phase 0 implements:
- Input: GitHub PR URL
- Output: JSON classification (`human_required` or `no_human`)

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

Output shape:

```json
{
  "classification": "human_required",
  "confidence": 0,
  "reason": "stage-2 confidence too low: 0.42",
  "run_id": "run-1741740000-123456"
}
```

## Notes

- Stage-1 timeout is fixed at 30 minutes.
- Confidence threshold defaults to `0.5` and can be overridden with `MIN_CONFIDENCE`.
- Normalizer model defaults to `claude-haiku-4-5@20251001` and can be overridden with `NORMALIZER_MODEL`.
- Logs are written to `logs/classify-pr.log` and mirrored to stderr.
- `LOGGING_LEVEL` controls verbosity (`debug`, `info`, `warn`, `error`); default is `warn`.
