package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/RohanAwhad/pr-review-bot/internal/normalize"
	"github.com/RohanAwhad/pr-review-bot/internal/pipeline"
	"github.com/RohanAwhad/pr-review-bot/internal/stage1"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: classify-pr <github-pr-url>\n")
		os.Exit(2)
	}
	prURL := os.Args[1]

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve working directory: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	project := envOr("GOOGLE_CLOUD_PROJECT", os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID"))
	region := envOr("CLOUD_ML_REGION", "us-east5")
	model := envOr("NORMALIZER_MODEL", "claude-haiku-4-5@20251001")
	image := envOr("STAGE1_IMAGE", "ghcr.io/anomalyco/opencode:latest")

	service := pipeline.Service{
		Stage1: stage1.Runner{
			Image:    image,
			RepoRoot: wd,
		},
		Normalizer:    normalize.New(ctx, region, project, model),
		MinConfidence: confidenceThreshold(),
	}

	decision := service.Classify(ctx, prURL, runID())
	out, err := json.MarshalIndent(decision, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode decision JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}

func runID() string {
	return fmt.Sprintf("run-%d-%06d", time.Now().Unix(), rand.Intn(1000000))
}

func confidenceThreshold() float64 {
	value := envOr("MIN_CONFIDENCE", "0.65")
	v, err := strconv.ParseFloat(value, 64)
	if err != nil || v < 0 || v > 1 {
		return 0.65
	}
	return v
}

func envOr(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
