# OpenCode Project Config

This directory stores project-scoped OpenCode config so stage-1 runs are reproducible across machines.

Included:
- `opencode.json`: provider + model defaults for `google-vertex-anthropic/claude-opus-4-6@default`
- `agents/auto-accept.md`: non-interactive agent profile so stage-1 runs do not block on tool permission prompts

Do not commit:
- `auth.json`
- any local credential files
- generated caches or local state

Required runtime env vars (provided outside git):
- `GITHUB_ACCESS_TOKEN`
- `GOOGLE_CLOUD_PROJECT`
- `CLOUD_ML_REGION`

Required runtime mount:
- Google ADC file at `/root/.config/gcloud/application_default_credentials.json`
