# Devlogs

## 2026-03-10 - Phase 0 kickoff

- Added repo-scoped OpenCode config under `.config/opencode` for reproducible container runs.
- Started Go implementation for phase-0 PR URL classification pipeline.
- Locked architecture: Podman stage-1 review + Haiku JSON normalization fallback policy.

## 2026-03-11 - Phase 0 implementation

- Added Go pipeline: PR URL parsing, stage-1 runner, stage-2 normalizer, and fallback policy.
- Added stage-1 image Dockerfile and build script for reproducible dependencies (`git`, `gh`, `bash`, `jq`).
- Added URL parser tests and README usage for running classification end-to-end.

## 2026-03-11 - Logging baseline

- Added structured logging with stderr + file sinks at `logs/classify-pr.log`.
- Added `LOGGING_LEVEL` support with default `warn` and optional `debug|info|warn|error`.
- Added pipeline/stage-1/stage-2 logging for failures and fallback decisions.

## 2026-03-11 - Smoke test runner

- Added `scripts/smoke_phase0.sh` to run 3 classification smoke tests in parallel.
- Captured each run to separate files, then printed section-wise output summaries.
- Added auto-cleanup for smoke artifacts with optional `KEEP_SMOKE_LOGS=1` retention.
- Expanded smoke coverage to include `first-pass` expectations for `new-math-mnist#9` and `agentic-rl#1`.

## 2026-03-11 - First-pass review pipeline

- Added `first-pass` CLI to generate initial PR review output from a GitHub PR URL.
- Added stage-2 normalizer schema for intent understanding, optimality verdict, focus areas, and blocking questions.
- Added deterministic merge readiness policy: low confidence/errors => `needs_human_review`, otherwise route to `wip` or `ready_to_merge`.
- Added stage-1 runner options for model override (`STAGE1_MODEL`) and context-file mode used by first-pass.
- Added unit tests for first-pass routing logic and fallback behavior.
