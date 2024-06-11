package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const DefaultPrompt = "Summarize content of the HTML blog post attached below. " +
	"Use only plain text in response. Use up to 5 sentences. " +
	"Skip \"this blog post outlines\" at the beginning."

type GeminiOptions struct {
	GeminiApiKey            string `arg:"--gemini-api-key,env:GEMINI_API_KEY" help:"API key for Gemini"`
	GeminiModel             string `arg:"--gemini-model,env:GEMINI_MODEL" default:"gemini-1.5-pro" help:"Gemini model to use for generating post description"`
	GeminiDescriptionPrompt string `arg:"env:GEMINI_DESCRIPTION_PROMPT" help:"prompt message to be used to generate HTML description"`
}

// DescribeContent generates a description for the provided text content using
// the Gemini API.
func DescribeContent(
	ctx context.Context,
	opts GeminiOptions,
	content string,
) (string, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(opts.GeminiApiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %v", err)
	}
	defer client.Close()

	query := DefaultPrompt
	if opts.GeminiDescriptionPrompt != "" {
		query = opts.GeminiDescriptionPrompt
	}

	model := client.GenerativeModel(opts.GeminiModel)
	resp, err := model.StartChat().SendMessage(ctx, genai.Text(
		fmt.Sprintf("%s\n\n```%s\n```", query, content),
	))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, candidate := range resp.Candidates {
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				sb.WriteString(fmt.Sprintf("%v", part))
			}
		}
	}

	return strings.TrimSpace(sb.String()), nil
}
