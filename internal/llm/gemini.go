package llm

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Client holds the Gemini client.
type Client struct {
	genaiClient *genai.GenerativeModel
}

// NewClient creates a new Gemini client.
func NewClient(apiKey, modelName string) (*Client, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	model := client.GenerativeModel(modelName)
	return &Client{genaiClient: model}, nil
}

// GenerateContent sends a prompt to the Gemini API and returns the response.
func (c *Client) GenerateContent(prompt string) (string, error) {
	ctx := context.Background()
	resp, err := c.genaiClient.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	// Assuming the first part is text
	content, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return "", fmt.Errorf("unexpected content type: %T", resp.Candidates[0].Content.Parts[0])
	}

	return string(content), nil
}
