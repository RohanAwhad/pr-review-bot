package pipeline

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/RohanAwhad/pr-review-bot/internal/classifier"
	"github.com/RohanAwhad/pr-review-bot/internal/normalize"
	"github.com/RohanAwhad/pr-review-bot/internal/stage1"
)

type Service struct {
	Stage1        stage1.Runner
	Normalizer    normalize.Normalizer
	MinConfidence float64
	Logger        *slog.Logger
}

func (s Service) Classify(ctx context.Context, prURL string, runID string) classifier.Decision {
	logger := s.logger()
	logger.Info("pipeline classification started")

	pr, err := classifier.ParsePullRequestURL(prURL)
	if err != nil {
		logger.Error("parse PR URL failed", "error", err)
		return fallback(runID, fmt.Sprintf("invalid pr url: %v", err))
	}

	stage1Output, err := s.Stage1.Run(ctx, pr)
	if err != nil {
		logger.Error("stage-1 failed", "error", err)
		return fallback(runID, fmt.Sprintf("stage-1 failed: %v", err))
	}

	decision, err := s.Normalizer.Classify(ctx, stage1Output)
	if err != nil {
		logger.Error("stage-2 failed", "error", err)
		return fallback(runID, fmt.Sprintf("stage-2 failed: %v", err))
	}
	decision.RunID = runID

	if decision.Confidence < s.MinConfidence {
		logger.Error("stage-2 confidence below threshold", "confidence", decision.Confidence, "min_confidence", s.MinConfidence)
		return fallback(runID, fmt.Sprintf("stage-2 confidence too low: %.2f", decision.Confidence))
	}

	if decision.Classification != classifier.ClassificationHumanRequired && decision.Classification != classifier.ClassificationNoHuman {
		logger.Error("stage-2 returned unsupported classification", "classification", decision.Classification)
		return fallback(runID, "stage-2 returned unsupported classification")
	}

	if decision.Reason == "" {
		logger.Error("stage-2 returned empty reason")
		return fallback(runID, "stage-2 returned empty reason")
	}

	logger.Info("pipeline classification completed", "classification", decision.Classification, "confidence", decision.Confidence)

	return decision
}

func (s Service) logger() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}

func fallback(runID string, reason string) classifier.Decision {
	return classifier.Decision{
		Classification: classifier.ClassificationHumanRequired,
		Confidence:     0,
		Reason:         reason,
		RunID:          runID,
	}
}
