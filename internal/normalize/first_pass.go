package normalize

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/RohanAwhad/pr-review-bot/internal/classifier"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/vertex"
)

type FirstPassNormalizer struct {
	client anthropic.Client
	model  anthropic.Model
	Logger *slog.Logger
}

type firstPassOutput struct {
	IntentUnderstanding firstPassIntent      `json:"intent_understanding" jsonschema:"required"`
	Optimality          firstPassOptimality  `json:"optimality" jsonschema:"required"`
	FocusAreas          []firstPassFocusArea `json:"focus_areas" jsonschema:"required"`
	BlockingQuestions   []string             `json:"blocking_questions" jsonschema:"required"`
}

type firstPassIntent struct {
	Verdict          string  `json:"verdict" jsonschema:"required,enum=yes,enum=partial,enum=no"`
	Confidence       float64 `json:"confidence" jsonschema:"required,minimum=0,maximum=1"`
	Reason           string  `json:"reason" jsonschema:"required"`
	UnderstoodIntent *string `json:"understood_intent" jsonschema:"required"`
}

type firstPassOptimality struct {
	Verdict      string   `json:"verdict" jsonschema:"required,enum=optimal,enum=acceptable,enum=suboptimal,enum=unknown"`
	Reason       string   `json:"reason" jsonschema:"required"`
	Alternatives []string `json:"alternatives" jsonschema:"required"`
}

type firstPassFocusArea struct {
	Path     string `json:"path" jsonschema:"required"`
	Why      string `json:"why" jsonschema:"required"`
	Priority string `json:"priority" jsonschema:"required,enum=high,enum=medium,enum=low"`
}

func NewFirstPass(ctx context.Context, region string, projectID string, model string) FirstPassNormalizer {
	client := anthropic.NewClient(vertex.WithGoogleAuth(ctx, region, projectID))
	return FirstPassNormalizer{client: client, model: anthropic.Model(model)}
}

func (n FirstPassNormalizer) Classify(ctx context.Context, stage1Output string) (classifier.FirstPassReview, error) {
	logger := n.logger().With("model", n.model)
	logger.Info("running first-pass stage-2 normalizer")

	tool := anthropic.ToolParam{
		Name:        "emit_first_pass",
		Description: anthropic.String("Emit the normalized first-pass review payload."),
		InputSchema: schema[firstPassOutput](),
	}
	tools := []anthropic.ToolUnionParam{{OfTool: &tool}}

	msg, err := n.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     n.model,
		MaxTokens: 512,
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(
			"Convert this stage-1 PR review output into the emit_first_pass tool payload. Keep understood_intent as raw text from stage-1 (or null when intent verdict is no). Return empty arrays for alternatives/focus_areas/blocking_questions when none are present.\n\n" + stage1Output,
		))},
		Tools: tools,
	})
	if err != nil {
		logger.Error("first-pass normalize call failed", "error", err)
		return classifier.FirstPassReview{}, fmt.Errorf("normalize first-pass output: %w", err)
	}

	for _, block := range msg.Content {
		toolUse, ok := block.AsAny().(anthropic.ToolUseBlock)
		if !ok || toolUse.Name != "emit_first_pass" {
			continue
		}

		payload, err := json.Marshal(toolUse.Input)
		if err != nil {
			logger.Error("marshal first-pass tool input", "error", err)
			return classifier.FirstPassReview{}, fmt.Errorf("marshal first-pass tool input: %w", err)
		}

		var out firstPassOutput
		if err := json.Unmarshal(payload, &out); err != nil {
			logger.Error("decode first-pass tool input", "error", err)
			return classifier.FirstPassReview{}, fmt.Errorf("decode first-pass tool input: %w", err)
		}

		intentVerdict := parseIntentVerdict(out.IntentUnderstanding.Verdict)
		optimalityVerdict := parseOptimalityVerdict(out.Optimality.Verdict)

		if out.IntentUnderstanding.Confidence < 0 || out.IntentUnderstanding.Confidence > 1 {
			logger.Error("intent confidence out of range", "confidence", out.IntentUnderstanding.Confidence)
			return classifier.FirstPassReview{}, fmt.Errorf("intent confidence out of range: %.4f", out.IntentUnderstanding.Confidence)
		}

		focusAreas := make([]classifier.FocusArea, 0, len(out.FocusAreas))
		for _, area := range out.FocusAreas {
			priority := parseFocusPriority(area.Priority)
			focusAreas = append(focusAreas, classifier.FocusArea{
				Path:     strings.TrimSpace(area.Path),
				Why:      strings.TrimSpace(area.Why),
				Priority: priority,
			})
		}

		understoodIntent := out.IntentUnderstanding.UnderstoodIntent
		if understoodIntent != nil {
			trimmed := strings.TrimSpace(*understoodIntent)
			understoodIntent = &trimmed
		}

		review := classifier.FirstPassReview{
			IntentUnderstanding: classifier.IntentUnderstanding{
				Verdict:          intentVerdict,
				Confidence:       out.IntentUnderstanding.Confidence,
				Reason:           strings.TrimSpace(out.IntentUnderstanding.Reason),
				UnderstoodIntent: understoodIntent,
			},
			Optimality: classifier.OptimalityAssessment{
				Verdict:      optimalityVerdict,
				Reason:       strings.TrimSpace(out.Optimality.Reason),
				Alternatives: trimStrings(out.Optimality.Alternatives),
			},
			FocusAreas:        focusAreas,
			BlockingQuestions: trimStrings(out.BlockingQuestions),
		}

		if review.IntentUnderstanding.Reason == "" {
			review.IntentUnderstanding.Reason = "Normalizer could not extract an intent reason from stage-1 output"
		}
		if review.Optimality.Reason == "" {
			review.Optimality.Reason = "Normalizer could not extract an optimality reason from stage-1 output"
		}

		if review.IntentUnderstanding.Verdict == classifier.IntentVerdictNo {
			review.IntentUnderstanding.UnderstoodIntent = nil
		}

		logger.Debug("first-pass stage-2 normalizer produced output", "intent_verdict", review.IntentUnderstanding.Verdict, "optimality_verdict", review.Optimality.Verdict)
		return review, nil
	}

	logger.Error("first-pass normalizer returned no tool call")
	return classifier.FirstPassReview{}, fmt.Errorf("first-pass normalizer did not emit tool call")
}

func (n FirstPassNormalizer) logger() *slog.Logger {
	if n.Logger != nil {
		return n.Logger
	}
	return slog.Default()
}

func parseIntentVerdict(raw string) classifier.IntentVerdict {
	switch normalizeToken(raw) {
	case "yes", "pass", "understood", "clear", "true":
		return classifier.IntentVerdictYes
	case "partial", "partially", "mixed":
		return classifier.IntentVerdictPartial
	case "no", "fail", "false", "unknown", "":
		return classifier.IntentVerdictNo
	default:
		return classifier.IntentVerdictNo
	}
}

func parseOptimalityVerdict(raw string) classifier.OptimalityVerdict {
	switch normalizeToken(raw) {
	case "optimal", "pass", "best", "most_optimal":
		return classifier.OptimalityVerdictOptimal
	case "acceptable", "mostly_optimal", "mostly_acceptable", "conditional_ready", "good_enough":
		return classifier.OptimalityVerdictAcceptable
	case "suboptimal", "fail", "not_optimal", "poor":
		return classifier.OptimalityVerdictSuboptimal
	case "unknown", "unclear", "na", "n_a", "":
		return classifier.OptimalityVerdictUnknown
	default:
		return classifier.OptimalityVerdictUnknown
	}
}

func parseFocusPriority(raw string) classifier.FocusPriority {
	switch normalizeToken(raw) {
	case "high", "critical", "p1", "sev1":
		return classifier.FocusPriorityHigh
	case "medium", "med", "p2", "sev2", "":
		return classifier.FocusPriorityMedium
	case "low", "p3", "p4", "sev3":
		return classifier.FocusPriorityLow
	default:
		return classifier.FocusPriorityMedium
	}
}

func normalizeToken(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "/", "_")
	return value
}

func trimStrings(values []string) []string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		candidate := strings.TrimSpace(value)
		if candidate == "" {
			continue
		}
		trimmed = append(trimmed, candidate)
	}
	return trimmed
}
