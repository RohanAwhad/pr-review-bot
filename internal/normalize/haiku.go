package normalize

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/RohanAwhad/pr-review-bot/internal/classifier"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/vertex"
	"github.com/invopop/jsonschema"
)

type Normalizer struct {
	client anthropic.Client
	model  anthropic.Model
	Logger *slog.Logger
}

type output struct {
	Classification string  `json:"classification" jsonschema:"required,enum=human_required,enum=no_human"`
	Confidence     float64 `json:"confidence" jsonschema:"required,minimum=0,maximum=1"`
	Reason         string  `json:"reason" jsonschema:"required"`
}

func New(ctx context.Context, region string, projectID string, model string) Normalizer {
	client := anthropic.NewClient(vertex.WithGoogleAuth(ctx, region, projectID))
	return Normalizer{client: client, model: anthropic.Model(model)}
}

func (n Normalizer) Classify(ctx context.Context, stage1Output string) (classifier.Decision, error) {
	logger := n.logger().With("model", n.model)
	logger.Info("running stage-2 normalizer")

	tool := anthropic.ToolParam{
		Name:        "emit_classification",
		Description: anthropic.String("Emit the final PR routing classification."),
		InputSchema: schema[output](),
	}
	tools := []anthropic.ToolUnionParam{{OfTool: &tool}}

	msg, err := n.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     n.model,
		MaxTokens: 256,
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(
			"Convert this stage-1 PR review output into the emit_classification tool payload. If unsure, use classification=human_required.\n\n" + stage1Output,
		))},
		Tools: tools,
	})
	if err != nil {
		logger.Error("normalize call failed", "error", err)
		return classifier.Decision{}, fmt.Errorf("normalize with haiku: %w", err)
	}

	for _, block := range msg.Content {
		toolUse, ok := block.AsAny().(anthropic.ToolUseBlock)
		if !ok || toolUse.Name != "emit_classification" {
			continue
		}

		payload, err := json.Marshal(toolUse.Input)
		if err != nil {
			logger.Error("marshal tool input", "error", err)
			return classifier.Decision{}, fmt.Errorf("marshal tool input: %w", err)
		}

		var out output
		if err := json.Unmarshal(payload, &out); err != nil {
			logger.Error("decode tool input", "error", err)
			return classifier.Decision{}, fmt.Errorf("decode tool input: %w", err)
		}

		classification := classifier.Classification(out.Classification)
		if classification != classifier.ClassificationHumanRequired && classification != classifier.ClassificationNoHuman {
			logger.Error("invalid classification", "classification", out.Classification)
			return classifier.Decision{}, fmt.Errorf("invalid classification from normalizer: %s", out.Classification)
		}
		logger.Debug("stage-2 normalizer produced classification", "classification", classification, "confidence", out.Confidence)

		return classifier.Decision{
			Classification: classification,
			Confidence:     out.Confidence,
			Reason:         out.Reason,
		}, nil
	}

	logger.Error("normalizer returned no tool call")
	return classifier.Decision{}, fmt.Errorf("normalizer did not emit classification tool call")
}

func (n Normalizer) logger() *slog.Logger {
	if n.Logger != nil {
		return n.Logger
	}
	return slog.Default()
}

func schema[T any]() anthropic.ToolInputSchemaParam {
	r := jsonschema.Reflector{AllowAdditionalProperties: false, DoNotReference: true}
	var v T
	s := r.Reflect(v)
	return anthropic.ToolInputSchemaParam{Properties: s.Properties, Required: s.Required}
}
