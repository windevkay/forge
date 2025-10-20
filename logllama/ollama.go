package logllama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	ollamaURL      = "http://localhost:11434/api/generate"
	ollamaModel    = "llama2"
	maxRetries     = 2
	requestTimeout = 30 * time.Second
)

// ollamaRequest represents the request payload for Ollama API.
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// ollamaResponse represents a single chunk of response from Ollama API.
type ollamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// AnalyzeErrorWithHistory sends the error and span history to Ollama for analysis
// and logs the model's solution to stdout. It runs in a background goroutine
// and retries up to 2 times on failure before giving up.
func AnalyzeErrorWithHistory(spanID string, errorMsg string, history []logEntry) {
	go func() {
		prompt := buildPrompt(errorMsg, history)

		var resp string
		var err error

		// Retry logic: try initial attempt + 2 retries
		for attempt := 0; attempt <= maxRetries; attempt++ {
			resp, err = queryOllama(prompt)
			if err == nil {
				break
			}
			if attempt < maxRetries {
				time.Sleep(time.Duration((attempt+1)*500) * time.Millisecond)
			}
		}

		if err != nil {
			slog.Error("failed to get model solution after retries",
				slog.String("span_id", spanID),
				slog.String("error", err.Error()))
			return
		}

		// Output the model's solution with span_id reference
		fmt.Printf("[MODEL_SOLUTION] span_id=%s\n%s\n\n", spanID, resp)
	}()
}

// buildPrompt constructs the prompt for the Ollama model.
func buildPrompt(errorMsg string, history []logEntry) string {
	historyText := ""
	for _, entry := range history {
		historyText += fmt.Sprintf("[%s] %s: %s\n",
			entry.Time.Format("15:04:05.000"),
			entry.Level.String(),
			entry.Message)
		if len(entry.Attrs) > 0 {
			for _, attr := range entry.Attrs {
				historyText += fmt.Sprintf("  %s: %v\n", attr.Key, attr.Value)
			}
		}
	}

	prompt := fmt.Sprintf(`You are a debugging assistant. Analyze the following error and its execution context to suggest a solution.

ERROR:
%s

EXECUTION HISTORY (leading up to the error):
%s

Based on the error message and the sequence of events in the history, provide a concise analysis and suggest possible solutions or next steps to resolve this issue.`, errorMsg, historyText)

	return prompt
}

// queryOllama sends a prompt to the Ollama API and returns the full response.
func queryOllama(prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	reqBody := ollamaRequest{
		Model:  ollamaModel,
		Prompt: prompt,
		Stream: false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ollamaURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var ollamaResp ollamaResponse
	err = json.Unmarshal(body, &ollamaResp)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return ollamaResp.Response, nil
}
