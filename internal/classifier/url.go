package classifier

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

func ParsePullRequestURL(raw string) (PullRequestRef, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return PullRequestRef{}, fmt.Errorf("parse PR URL: %w", err)
	}
	if u.Scheme != "https" || u.Host != "github.com" {
		return PullRequestRef{}, fmt.Errorf("unsupported PR URL: %s", raw)
	}

	parts := strings.Split(strings.Trim(path.Clean(u.Path), "/"), "/")
	if len(parts) < 4 || parts[2] != "pull" || parts[3] == "" {
		return PullRequestRef{}, fmt.Errorf("invalid PR URL path: %s", u.Path)
	}

	return PullRequestRef{
		Owner:  parts[0],
		Repo:   parts[1],
		Number: parts[3],
		URL:    raw,
	}, nil
}
