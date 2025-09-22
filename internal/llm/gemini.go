package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/generative-ai-go/genai"
	"golang.org/x/time/rate"
	"google.golang.org/api/option"
)

// Client holds the Gemini client and a rate limiter.
type Client struct {
	genaiClient *genai.GenerativeModel
	limiter     *rate.Limiter
}

// NewClient creates a new Gemini client with a rate limiter.
func NewClient(apiKey, modelName string, rpm int) (*Client, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	// Create a rate limiter based on the configured RPM.
	// The interval is 1 minute / RPM.
	limit := rate.Every(time.Minute / time.Duration(rpm))
	limiter := rate.NewLimiter(limit, 1) // Allow 1 burst

	model := client.GenerativeModel(modelName)
	return &Client{genaiClient: model, limiter: limiter}, nil
}

// GenerateContent sends a prompt to the Gemini API and returns the response, respecting the rate limit.
func (c *Client) GenerateContent(prompt string) (string, error) {
	ctx := context.Background()

	// Wait for the rate limiter.
	if err := c.limiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limiter wait failed: %w", err)
	}

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
