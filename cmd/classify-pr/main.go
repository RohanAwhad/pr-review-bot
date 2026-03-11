package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/RohanAwhad/pr-review-bot/internal/logging"
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
	loadDotEnv(filepath.Join(wd, ".env"))

	logger, logSink, logPath, err := logging.New("classify-pr")
	if err != nil {
		fmt.Fprintf(os.Stderr, "configure logger: %v\n", err)
		os.Exit(1)
	}
	defer logSink.Close()
	slog.SetDefault(logger)

	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Error("panic recovered", "panic", recovered, "stack", string(debug.Stack()))
			os.Exit(1)
		}
	}()

	id := runID()
	runLogger := logger.With("run_id", id, "pr_url", prURL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	project := envOr("GOOGLE_CLOUD_PROJECT", os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID"))
	region := envOr("CLOUD_ML_REGION", "us-east5")
	model := envOr("NORMALIZER_MODEL", "claude-haiku-4-5@20251001")
	image := envOr("PR_REVIEW_BOT_STAGE1_IMAGE", envOr("STAGE1_IMAGE", "pr-review-bot-stage1:latest"))
	stage1Model := os.Getenv("STAGE1_MODEL")
	runLogger.Info("starting classification", "image", image, "normalizer_model", model, "stage1_model", stage1Model)

	normalizer := normalize.New(ctx, region, project, model)
	normalizer.Logger = runLogger

	service := pipeline.Service{
		Stage1: stage1.Runner{
			Image:    image,
			RepoRoot: wd,
			Model:    stage1Model,
			Logger:   runLogger,
		},
		Normalizer:    normalizer,
		MinConfidence: confidenceThreshold(),
		Logger:        runLogger,
	}

	decision := service.Classify(ctx, prURL, id)
	runLogger.Info("classification completed", "classification", decision.Classification, "confidence", decision.Confidence)
	out, err := json.MarshalIndent(decision, "", "  ")
	if err != nil {
		runLogger.Error("encode decision JSON", "error", err)
		fmt.Fprintf(os.Stderr, "encode decision JSON: %v\n", err)
		os.Exit(1)
	}
	runLogger.Debug("writing decision JSON to stdout", "log_path", logPath)
	fmt.Println(string(out))
}

func runID() string {
	return fmt.Sprintf("run-%d-%06d", time.Now().Unix(), rand.Intn(1000000))
}

func confidenceThreshold() float64 {
	value := envOr("MIN_CONFIDENCE", "0.5")
	v, err := strconv.ParseFloat(value, 64)
	if err != nil || v < 0 || v > 1 {
		return 0.5
	}
	return v
}

func envOr(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		if key == "" || os.Getenv(key) != "" {
			continue
		}

		value := strings.TrimSpace(parts[1])
		value = strings.TrimPrefix(value, "\"")
		value = strings.TrimSuffix(value, "\"")
		_ = os.Setenv(key, value)
	}
}
