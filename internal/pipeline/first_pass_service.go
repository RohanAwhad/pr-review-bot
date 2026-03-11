package pipeline

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/RohanAwhad/pr-review-bot/internal/classifier"
)

type firstPassStage1Runner interface {
	Run(ctx context.Context, pr classifier.PullRequestRef) (string, error)
}

type firstPassStage2Normalizer interface {
	Classify(ctx context.Context, stage1Output string) (classifier.FirstPassReview, error)
}

type FirstPassService struct {
	Stage1        firstPassStage1Runner
	Normalizer    firstPassStage2Normalizer
	MinConfidence float64
	Logger        *slog.Logger
}

func (s FirstPassService) Review(ctx context.Context, prURL string, runID string) classifier.FirstPassReview {
	logger := s.logger()
	logger.Info("first-pass pipeline started")

	pr, err := classifier.ParsePullRequestURL(prURL)
	if err != nil {
		logger.Error("parse PR URL failed", "error", err)
		return firstPassFallback(runID, fmt.Sprintf("invalid pr url: %v", err))
	}

	stage1Output, err := s.Stage1.Run(ctx, pr)
	if err != nil {
		logger.Error("first-pass stage-1 failed", "error", err)
		return firstPassFallback(runID, fmt.Sprintf("stage-1 failed: %v", err))
	}

	review, err := s.Normalizer.Classify(ctx, stage1Output)
	if err != nil {
		logger.Error("first-pass stage-2 failed", "error", err)
		return firstPassFallback(runID, fmt.Sprintf("stage-2 failed: %v", err))
	}
	review.RunID = runID

	if review.IntentUnderstanding.Confidence < s.MinConfidence {
		logger.Error("first-pass stage-2 confidence below threshold", "confidence", review.IntentUnderstanding.Confidence, "min_confidence", s.MinConfidence)
		return firstPassFallback(runID, fmt.Sprintf("stage-2 confidence too low: %.2f", review.IntentUnderstanding.Confidence))
	}

	review.MergeReadiness = deriveMergeReadiness(review)
	logger.Info("first-pass pipeline completed", "merge_readiness", review.MergeReadiness, "intent_verdict", review.IntentUnderstanding.Verdict, "optimality_verdict", review.Optimality.Verdict)

	return review
}

func (s FirstPassService) logger() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}

func deriveMergeReadiness(review classifier.FirstPassReview) classifier.MergeReadiness {
	if review.IntentUnderstanding.Verdict != classifier.IntentVerdictYes {
		return classifier.MergeReadinessWIP
	}

	if review.Optimality.Verdict == classifier.OptimalityVerdictSuboptimal || review.Optimality.Verdict == classifier.OptimalityVerdictUnknown {
		return classifier.MergeReadinessWIP
	}

	if len(review.BlockingQuestions) > 0 {
		return classifier.MergeReadinessWIP
	}

	return classifier.MergeReadinessReadyToMerge
}

func firstPassFallback(runID string, reason string) classifier.FirstPassReview {
	return classifier.FirstPassReview{
		IntentUnderstanding: classifier.IntentUnderstanding{
			Verdict:          classifier.IntentVerdictNo,
			Confidence:       0,
			Reason:           reason,
			UnderstoodIntent: nil,
		},
		Optimality: classifier.OptimalityAssessment{
			Verdict:      classifier.OptimalityVerdictUnknown,
			Reason:       reason,
			Alternatives: []string{},
		},
		MergeReadiness:    classifier.MergeReadinessNeedsHumanReview,
		FocusAreas:        []classifier.FocusArea{},
		BlockingQuestions: []string{reason},
		RunID:             runID,
	}
}
