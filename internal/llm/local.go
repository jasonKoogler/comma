package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// generateWithLocal calls a local LLM API to generate a commit message
func (c *Client) generateWithLocal(prompt string, maxTokens int) (string, error) {
	// If no endpoint is specified, use default ollama endpoint
	endpoint := c.endpoint
	if endpoint == "" {
		endpoint = "http://localhost:11434/api/generate"
	}

	// Use default model if not specified
	model := c.model
	if model == "" {
		model = "llama3"
	}

	// Prepare request
	requestBody := map[string]interface{}{
		"model":       model,
		"prompt":      prompt,
		"temperature": c.temperature,
		"max_tokens":  maxTokens,
		"stream":      false,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	httpClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned non-200 status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var response struct {
		Response string `json:"response"`
		Error    string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API error
	if response.Error != "" {
		return "", fmt.Errorf("API error: %s", response.Error)
	}

	return response.Response, nil
}
