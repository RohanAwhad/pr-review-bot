package classifier

import "testing"

func TestParsePullRequestURL(t *testing.T) {
	pr, err := ParsePullRequestURL("https://github.com/RohanAwhad/new-math-mnist/pull/9")
	if err != nil {
		t.Fatalf("expected URL parse success: %v", err)
	}
	if pr.Owner != "RohanAwhad" || pr.Repo != "new-math-mnist" || pr.Number != "9" {
		t.Fatalf("unexpected parsed PR ref: %+v", pr)
	}
}

func TestParsePullRequestURLRejectsInvalidPath(t *testing.T) {
	_, err := ParsePullRequestURL("https://github.com/RohanAwhad/new-math-mnist/issues/9")
	if err == nil {
		t.Fatal("expected parse error for non-PR URL")
	}
}
