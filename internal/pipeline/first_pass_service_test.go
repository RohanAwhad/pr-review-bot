package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/RohanAwhad/pr-review-bot/internal/classifier"
)

type fakeFirstPassStage1 struct {
	output string
	err    error
}

func (f fakeFirstPassStage1) Run(_ context.Context, _ classifier.PullRequestRef) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.output, nil
}

type fakeFirstPassNormalizer struct {
	review classifier.FirstPassReview
	err    error
}

func (f fakeFirstPassNormalizer) Classify(_ context.Context, _ string) (classifier.FirstPassReview, error) {
	if f.err != nil {
		return classifier.FirstPassReview{}, f.err
	}
	return f.review, nil
}

func TestFirstPassReviewReadyToMerge(t *testing.T) {
	review := baselineFirstPassReview()
	review.Optimality.Verdict = classifier.OptimalityVerdictAcceptable

	service := FirstPassService{
		Stage1:        fakeFirstPassStage1{output: "ok"},
		Normalizer:    fakeFirstPassNormalizer{review: review},
		MinConfidence: 0.5,
	}

	result := service.Review(context.Background(), "https://github.com/RohanAwhad/new-math-mnist/pull/9", "run-1")

	if result.MergeReadiness != classifier.MergeReadinessReadyToMerge {
		t.Fatalf("expected ready_to_merge, got %s", result.MergeReadiness)
	}
	if result.RunID != "run-1" {
		t.Fatalf("expected run_id to be set, got %s", result.RunID)
	}
}

func TestFirstPassReviewWIPRules(t *testing.T) {
	tests := []struct {
		name   string
		review classifier.FirstPassReview
	}{
		{
			name: "intent partial",
			review: func() classifier.FirstPassReview {
				candidate := baselineFirstPassReview()
				candidate.IntentUnderstanding.Verdict = classifier.IntentVerdictPartial
				return candidate
			}(),
		},
		{
			name: "optimality unknown",
			review: func() classifier.FirstPassReview {
				candidate := baselineFirstPassReview()
				candidate.Optimality.Verdict = classifier.OptimalityVerdictUnknown
				return candidate
			}(),
		},
		{
			name: "blocking questions present",
			review: func() classifier.FirstPassReview {
				candidate := baselineFirstPassReview()
				candidate.BlockingQuestions = []string{"Can this break in prod?"}
				return candidate
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := FirstPassService{
				Stage1:        fakeFirstPassStage1{output: "ok"},
				Normalizer:    fakeFirstPassNormalizer{review: tt.review},
				MinConfidence: 0.5,
			}

			result := service.Review(context.Background(), "https://github.com/RohanAwhad/new-math-mnist/pull/9", "run-2")
			if result.MergeReadiness != classifier.MergeReadinessWIP {
				t.Fatalf("expected wip, got %s", result.MergeReadiness)
			}
		})
	}
}

func TestFirstPassReviewFallbackRules(t *testing.T) {
	t.Run("stage1 failure", func(t *testing.T) {
		service := FirstPassService{
			Stage1:        fakeFirstPassStage1{err: errors.New("boom")},
			Normalizer:    fakeFirstPassNormalizer{review: baselineFirstPassReview()},
			MinConfidence: 0.5,
		}

		result := service.Review(context.Background(), "https://github.com/RohanAwhad/new-math-mnist/pull/9", "run-3")
		if result.MergeReadiness != classifier.MergeReadinessNeedsHumanReview {
			t.Fatalf("expected needs_human_review, got %s", result.MergeReadiness)
		}
	})

	t.Run("stage2 failure", func(t *testing.T) {
		service := FirstPassService{
			Stage1:        fakeFirstPassStage1{output: "ok"},
			Normalizer:    fakeFirstPassNormalizer{err: errors.New("normalize failed")},
			MinConfidence: 0.5,
		}

		result := service.Review(context.Background(), "https://github.com/RohanAwhad/new-math-mnist/pull/9", "run-4")
		if result.MergeReadiness != classifier.MergeReadinessNeedsHumanReview {
			t.Fatalf("expected needs_human_review, got %s", result.MergeReadiness)
		}
	})

	t.Run("low confidence", func(t *testing.T) {
		lowConfidence := baselineFirstPassReview()
		lowConfidence.IntentUnderstanding.Confidence = 0.2

		service := FirstPassService{
			Stage1:        fakeFirstPassStage1{output: "ok"},
			Normalizer:    fakeFirstPassNormalizer{review: lowConfidence},
			MinConfidence: 0.5,
		}

		result := service.Review(context.Background(), "https://github.com/RohanAwhad/new-math-mnist/pull/9", "run-5")
		if result.MergeReadiness != classifier.MergeReadinessNeedsHumanReview {
			t.Fatalf("expected needs_human_review, got %s", result.MergeReadiness)
		}
	})
}

func baselineFirstPassReview() classifier.FirstPassReview {
	intent := "Package-first migration with API exports"
	return classifier.FirstPassReview{
		IntentUnderstanding: classifier.IntentUnderstanding{
			Verdict:          classifier.IntentVerdictYes,
			Confidence:       0.91,
			Reason:           "Intent is explicit in commits and files",
			UnderstoodIntent: &intent,
		},
		Optimality: classifier.OptimalityAssessment{
			Verdict:      classifier.OptimalityVerdictOptimal,
			Reason:       "Approach minimizes churn and clarifies imports",
			Alternatives: []string{},
		},
		FocusAreas:        []classifier.FocusArea{},
		BlockingQuestions: []string{},
	}
}
