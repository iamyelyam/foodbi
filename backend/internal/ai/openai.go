package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OpenAIClient struct {
	apiKey string
	client *http.Client
}

func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		apiKey: apiKey,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	Temperature    float64       `json:"temperature"`
	ResponseFormat *respFormat   `json:"response_format,omitempty"`
}

type respFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// AISuggestion is the structured output we expect from the model.
type AISuggestion struct {
	Type        string  `json:"type"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Solution    string  `json:"solution"`
	Impact      string  `json:"impact"`
	LossAmount  float64 `json:"loss_amount"`
	GainAmount  float64 `json:"gain_amount"`
}

func (c *OpenAIClient) GenerateSuggestions(ctx context.Context, dataJSON string) ([]AISuggestion, string, error) {
	systemPrompt := `You are a restaurant business analyst for restaurants in Kazakhstan.
You receive JSON data about a restaurant location: product sales (with revenue and cost), current stock levels, and purchase history.

Analyze the data and return 3-5 actionable business suggestions in Russian language.

Return a JSON array of objects with these fields:
- type: one of "ai_menu", "ai_cost", "ai_stock", "ai_purchase", "ai_general"
- title: short title (max 60 chars)
- description: detailed explanation (2-3 sentences)
- solution: specific actionable recommendation (1-2 sentences)
- impact: "high", "medium", or "low"
- loss_amount: estimated monetary loss (0 if not applicable), in KZT, no subunits
- gain_amount: estimated monetary gain if suggestion is followed (0 if not applicable), in KZT, no subunits

Focus on:
1. Menu optimization (promote high-margin items, reconsider low-margin ones)
2. Cost issues (products with suspicious margins, missing cost prices)
3. Stock problems (negative amounts, items sitting too long)
4. Purchase optimization (consolidate suppliers, negotiate volume discounts)
5. General business insights from the data patterns

Return ONLY the JSON array, no markdown or wrapping.`

	req := chatRequest{
		Model: "gpt-4o-mini",
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: "Here is the restaurant data:\n" + dataJSON},
		},
		Temperature:    0.7,
		ResponseFormat: &respFormat{Type: "json_object"},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read response: %w", err)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, string(respBody), fmt.Errorf("unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, string(respBody), fmt.Errorf("openai error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, string(respBody), fmt.Errorf("openai returned no choices")
	}

	raw := chatResp.Choices[0].Message.Content

	// JSON mode wraps in {"suggestions": [...]} — try both shapes.
	var suggestions []AISuggestion
	if err := json.Unmarshal([]byte(raw), &suggestions); err != nil {
		// Try wrapped format
		var wrapped struct {
			Suggestions []AISuggestion `json:"suggestions"`
		}
		if err2 := json.Unmarshal([]byte(raw), &wrapped); err2 != nil {
			return nil, raw, fmt.Errorf("parse suggestions: %w (raw: %s)", err, raw)
		}
		suggestions = wrapped.Suggestions
	}

	return suggestions, raw, nil
}
