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
