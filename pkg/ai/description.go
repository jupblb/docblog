package ai

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const Query = "Summarize content of the HTML blog post attached below. " +
	"Use only plain text in response. Use up to 5 sentences. " +
	"Skip \"this blog post outlines\" at the beginning."

func DescribeContent(ctx context.Context, content []byte) (string, error) {
	client, err := genai.NewClient(
		ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-pro")
	resp, err := model.StartChat().SendMessage(ctx, genai.Text(
		fmt.Sprintf("%s\n\n```%s\n```", Query, string(content)),
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
