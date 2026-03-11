package pipeline

import (
	"context"
	"fmt"

	"github.com/RohanAwhad/pr-review-bot/internal/classifier"
	"github.com/RohanAwhad/pr-review-bot/internal/normalize"
	"github.com/RohanAwhad/pr-review-bot/internal/stage1"
)

type Service struct {
	Stage1        stage1.Runner
	Normalizer    normalize.Normalizer
	MinConfidence float64
}

func (s Service) Classify(ctx context.Context, prURL string, runID string) classifier.Decision {
	pr, err := classifier.ParsePullRequestURL(prURL)
	if err != nil {
		return fallback(runID, fmt.Sprintf("invalid pr url: %v", err))
	}

	stage1Output, err := s.Stage1.Run(ctx, pr)
	if err != nil {
		return fallback(runID, fmt.Sprintf("stage-1 failed: %v", err))
	}

	decision, err := s.Normalizer.Classify(ctx, stage1Output)
	if err != nil {
		return fallback(runID, fmt.Sprintf("stage-2 failed: %v", err))
	}
	decision.RunID = runID

	if decision.Confidence < s.MinConfidence {
		return fallback(runID, fmt.Sprintf("stage-2 confidence too low: %.2f", decision.Confidence))
	}

	if decision.Classification != classifier.ClassificationHumanRequired && decision.Classification != classifier.ClassificationNoHuman {
		return fallback(runID, "stage-2 returned unsupported classification")
	}

	if decision.Reason == "" {
		return fallback(runID, "stage-2 returned empty reason")
	}

	return decision
}

func fallback(runID string, reason string) classifier.Decision {
	return classifier.Decision{
		Classification: classifier.ClassificationHumanRequired,
		Confidence:     0,
		Reason:         reason,
		RunID:          runID,
	}
}
