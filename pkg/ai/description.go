// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
