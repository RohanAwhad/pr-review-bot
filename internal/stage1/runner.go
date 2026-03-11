package stage1

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/RohanAwhad/pr-review-bot/internal/classifier"
)

const prompt = "You are stage-1 PR risk classifier. Analyze this checked-out PR branch against main and classify if HUMAN review is required. DO NOT run installs/tests. Only inspect git history, diff, and changed files. End with exactly: CLASSIFICATION: human_required|no_human then CONFIDENCE: <0-1> then REASON: <one sentence>."

type Runner struct {
	Image    string
	RepoRoot string
}

func (r Runner) Run(ctx context.Context, pr classifier.PullRequestRef) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	adcPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if adcPath == "" {
		adcPath = filepath.Join(home, ".config", "gcloud", "application_default_credentials.json")
	}

	script := `set -eu; apk add --no-cache git github-cli >/dev/null; owner_repo="${PR_OWNER}/${PR_REPO}"; pr_number="${PR_NUMBER}"; repo_name="${PR_REPO}"; mkdir -p /work && cd /work; git clone "https://x-access-token:${GITHUB_ACCESS_TOKEN}@github.com/${owner_repo}.git" "$repo_name" >/dev/null 2>&1; cd "$repo_name"; git fetch origin "pull/${pr_number}/head:pr-${pr_number}" >/dev/null 2>&1; git checkout "pr-${pr_number}" >/dev/null 2>&1; opencode run "$STAGE1_PROMPT" --agent auto-accept --dir "/work/$repo_name"`

	args := []string{
		"run", "--rm", "--entrypoint", "sh",
		"-e", "NO_COLOR=1",
		"-e", "GITHUB_ACCESS_TOKEN",
		"-e", "ANTHROPIC_VERTEX_PROJECT_ID",
		"-e", "CLOUD_ML_REGION",
		"-e", "GOOGLE_CLOUD_PROJECT",
		"-e", "VERTEX_LOCATION",
		"-e", "GOOGLE_CLOUD_LOCATION",
		"-e", "PR_OWNER=" + pr.Owner,
		"-e", "PR_REPO=" + pr.Repo,
		"-e", "PR_NUMBER=" + pr.Number,
		"-e", "STAGE1_PROMPT=" + prompt,
		"-v", filepath.Join(r.RepoRoot, ".config", "opencode") + ":/root/.config/opencode:ro",
		"-v", filepath.Join(home, ".local", "share", "opencode", "auth.json") + ":/root/.local/share/opencode/auth.json:ro",
		"-v", adcPath + ":/root/.config/gcloud/application_default_credentials.json:ro",
		r.Image,
		"-lc", script,
	}

	cmd := exec.CommandContext(ctx, "podman", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	if err != nil {
		return out.String(), fmt.Errorf("run stage-1 container: %w", err)
	}
	return out.String(), nil
}
