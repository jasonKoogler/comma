// internal/llm/local_models.go
package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// LocalModelConfig represents configuration for embedded LLM
type LocalModelConfig struct {
	ModelPath   string
	ContextSize int
	ThreadCount int
	Temperature float64
	TopP        float64
	EnableCache bool
	CacheDir    string
}

// LocalModel manages interaction with embedded LLMs
type LocalModel struct {
	config LocalModelConfig
	binary string
}

// NewLocalModel initializes a local model provider
func NewLocalModel(configDir string) (*LocalModel, error) {
	cacheDir := filepath.Join(configDir, "model_cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Default config
	config := LocalModelConfig{
		ModelPath:   "", // Will be auto-detected
		ContextSize: 2048,
		ThreadCount: runtime.NumCPU() / 2,
		Temperature: 0.7,
		TopP:        0.9,
		EnableCache: true,
		CacheDir:    cacheDir,
	}

	// Find appropriate binary for platform
	binary, err := findLLMBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to find LLM binary: %w", err)
	}

	// Find model path
	modelPath, err := findModelPath(configDir)
	if err != nil {
		return nil, fmt.Errorf("no local model found: %w", err)
	}
	config.ModelPath = modelPath

	return &LocalModel{
		config: config,
		binary: binary,
	}, nil
}

// findLLMBinary locates the LLM binary for the current platform
func findLLMBinary() (string, error) {
	// Check common paths for llama.cpp or other compatible LLM binaries
	potentialPaths := []string{
		"/usr/local/bin/llama",
		"/usr/bin/llama",
	}

	// Check if in PATH
	path, err := exec.LookPath("llama")
	if err == nil {
		return path, nil
	}

	// Check if Ollama is installed
	ollamaPath, err := exec.LookPath("ollama")
	if err == nil {
		return ollamaPath, nil
	}

	// Check potential paths
	for _, p := range potentialPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("no compatible LLM binary found")
}

// findModelPath looks for locally installed models
func findModelPath(configDir string) (string, error) {
	modelsDir := filepath.Join(configDir, "models")

	// Common model file extensions
	extensions := []string{".gguf", ".bin", ".model"}

	// Search for models in the models directory
	var modelPaths []string
	err := filepath.Walk(modelsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			ext := filepath.Ext(path)
			for _, validExt := range extensions {
				if ext == validExt {
					modelPaths = append(modelPaths, path)
					break
				}
			}
		}

		return nil
	})

	if err != nil || len(modelPaths) == 0 {
		return "", fmt.Errorf("no models found in %s", modelsDir)
	}

	// Use the first found model
	return modelPaths[0], nil
}

// GenerateWithLocalModel uses local LLM for generation
func (lm *LocalModel) Generate(prompt string, maxTokens int) (string, error) {
	// Check if Ollama
	if strings.Contains(lm.binary, "ollama") {
		return lm.generateWithOllama(prompt, maxTokens)
	}

	// Use llama.cpp binary
	args := []string{
		"-m", lm.config.ModelPath,
		"-c", fmt.Sprintf("%d", lm.config.ContextSize),
		"-n", fmt.Sprintf("%d", maxTokens),
		"-t", fmt.Sprintf("%d", lm.config.ThreadCount),
		"--temp", fmt.Sprintf("%.2f", lm.config.Temperature),
		"--top_p", fmt.Sprintf("%.2f", lm.config.TopP),
		"--repeat_penalty", "1.1",
		"-p", prompt,
	}

	cmd := exec.Command(lm.binary, args...)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("local model inference failed: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

// generateWithOllama handles generation using Ollama
func (lm *LocalModel) generateWithOllama(prompt string, maxTokens int) (string, error) {
	// Determine model name - use a smaller one suitable for commit messages
	modelName := "llama2"

	requestBody := map[string]interface{}{
		"model":       modelName,
		"prompt":      prompt,
		"temperature": lm.config.Temperature,
		"max_tokens":  maxTokens,
		"stream":      false,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Run ollama command
	cmd := exec.Command(lm.binary, "run", "-j", modelName)
	cmd.Stdin = bytes.NewBuffer(jsonBody)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ollama inference failed: %w", err)
	}

	// Parse JSON response
	var response struct {
		Response string `json:"response"`
	}

	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		// If can't parse as JSON, just return the raw output
		return strings.TrimSpace(out.String()), nil
	}

	return response.Response, nil
}
