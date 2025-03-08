package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// generateWithAnthropic calls the Anthropic API to generate a commit message
func (c *Client) generateWithAnthropic(prompt string, maxTokens int) (string, error) {
	// Respect rate limit
	<-c.rateLimiter.C

	// Use default model if not specified
	model := c.model
	if model == "" {
		model = "claude-3-opus-20240229"
	}

	// Prepare request
	requestBody := map[string]interface{}{
		"model":       model,
		"max_tokens":  maxTokens,
		"temperature": c.temperature,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

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
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			// Print detailed error for debugging
			if resp.StatusCode != http.StatusOK {
				fmt.Printf("Anthropic API error (attempt %d/%d): Status %d, Body: %s\n",
					i+1, maxRetries, resp.StatusCode, string(bodyBytes))
			}
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
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Model        string `json:"model"`
		StopReason   string `json:"stop_reason"`
		StopSequence string `json:"stop_sequence"`
		Usage        struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w, body: %s", err, string(bodyBytes))
	}

	// Check for API error
	if response.Error != nil && response.Error.Message != "" {
		return "", fmt.Errorf("API error: %s", response.Error.Message)
	}

	// Extract message from the text content
	for _, content := range response.Content {
		if content.Type == "text" {
			return content.Text, nil
		}
	}

	return "", fmt.Errorf("no text content returned from API")
}
