package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// generateWithOpenAI calls the OpenAI API to generate a commit message
func (c *Client) generateWithOpenAI(prompt string, maxTokens int) (string, error) {
	// Respect rate limit
	<-c.rateLimiter.C

	// Use default model if not specified
	model := c.model
	if model == "" {
		model = "gpt-4"
	}

	// Prepare request
	requestBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a helpful assistant that generates concise and descriptive git commit messages.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":  maxTokens,
		"temperature": c.temperature,
		"top_p":       1,
		"stream":      false,
		"stop":        nil,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", c.endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Execute request with retry
	httpClient := &http.Client{Timeout: 60 * time.Second}
	var resp *http.Response
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		resp, err = httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}

		if i < maxRetries-1 {
			// Exponential backoff
			time.Sleep(time.Duration((1<<i)*500) * time.Millisecond)
		}
	}

	if err != nil {
		return "", fmt.Errorf("request failed after %d retries: %w", maxRetries, err)
	}

	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned non-200 status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API error
	if response.Error.Message != "" {
		return "", fmt.Errorf("API error: %s", response.Error.Message)
	}

	// Extract message
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from API")
	}

	return response.Choices[0].Message.Content, nil
}
