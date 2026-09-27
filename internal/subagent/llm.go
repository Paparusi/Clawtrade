// internal/subagent/llm.go
package subagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// LLMCaller makes single-shot LLM API calls (system+user -> text response).
// Unlike the main agent, it has no tool calling or message history —
// sub-agents use it for analysis only.
type LLMCaller struct {
	Provider  string
	Model     string
	APIKey    string
	MaxTokens int
	BaseURL   string
}

// NewLLMCaller creates an LLMCaller from a "provider/model" string.
// For openrouter models with nested slashes (e.g. "openrouter/meta-llama/llama-3-70b"),
// everything after the first slash is treated as the model name.
func NewLLMCaller(modelString, apiKey string, maxTokens int) *LLMCaller {
	parts := strings.SplitN(modelString, "/", 2)
	provider := parts[0]
	model := ""
	if len(parts) > 1 {
		model = parts[1]
	}

	openAICompatible := map[string]bool{
		"openai": true, "deepseek": true, "openrouter": true, "ollama": true,
		"opencode": true, "opencode-go": true, "openai-compatible": true, "custom": true,
	}
	baseURL := map[string]string{
		"anthropic":         "https://api.anthropic.com/v1/messages",
		"openai":            "https://api.openai.com/v1/chat/completions",
		"deepseek":          "https://api.deepseek.com/v1/chat/completions",
		"openrouter":        "https://openrouter.ai/api/v1/chat/completions",
		"ollama":            "http://localhost:11434/v1/chat/completions",
		"google":            "https://generativelanguage.googleapis.com/v1beta",
		"opencode":          "https://opencode.ai/zen/go/v1/chat/completions",
		"opencode-go":       "https://opencode.ai/zen/go/v1/chat/completions",
		"openai-compatible": "https://opencode.ai/zen/go/v1/chat/completions",
		"custom":            "https://opencode.ai/zen/go/v1/chat/completions",
	}[provider]
	if openAICompatible[provider] {
		if override := strings.TrimSpace(os.Getenv("CLAWTRADE_LLM_BASE_URL")); override != "" {
			baseURL = strings.TrimRight(override, "/") + "/chat/completions"
		}
	}

	return &LLMCaller{
		Provider:  provider,
		Model:     model,
		APIKey:    apiKey,
		MaxTokens: maxTokens,
		BaseURL:   baseURL,
	}
}

// Call sends a single-shot request to the configured LLM provider and returns
// the text response. No tool calling, no message history.
func (c *LLMCaller) Call(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	switch c.Provider {
	case "anthropic":
		return c.callAnthropic(ctx, systemPrompt, userMessage)
	case "openai", "deepseek", "openrouter", "ollama", "opencode", "opencode-go", "openai-compatible", "custom":
		return c.callOpenAI(ctx, systemPrompt, userMessage)
	case "google":
		return c.callGoogle(ctx, systemPrompt, userMessage)
	default:
		return "", fmt.Errorf("unsupported provider: %s", c.Provider)
	}
}

// ─── Anthropic ───────────────────────────────────────────────────────

func (c *LLMCaller) buildAnthropicBody(system, user string) map[string]any {
	return map[string]any{
		"model":      c.Model,
		"max_tokens": c.MaxTokens,
		"system":     system,
		"messages": []map[string]any{
			{"role": "user", "content": user},
		},
	}
}

func (c *LLMCaller) callAnthropic(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	body := c.buildAnthropicBody(systemPrompt, userMessage)
	raw, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("anthropic API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var ar struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &ar); err != nil {
		return "", fmt.Errorf("parse anthropic response: %w", err)
	}

	var textParts []string
	for _, block := range ar.Content {
		if block.Type == "text" {
			textParts = append(textParts, block.Text)
		}
	}
	return strings.Join(textParts, "\n"), nil
}

// ─── OpenAI-compatible (OpenAI, DeepSeek, OpenRouter, Ollama) ────────

func (c *LLMCaller) buildOpenAIBody(system, user string) map[string]any {
	return map[string]any{
		"model": c.Model,
		"messages": []map[string]any{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"max_tokens": c.MaxTokens,
	}
}

func (c *LLMCaller) callOpenAI(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	body := c.buildOpenAIBody(systemPrompt, userMessage)
	raw, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" && c.Provider != "ollama" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	if strings.Contains(c.BaseURL, "opencode.ai") {
		req.Header.Set("x-opencode-session", llmSessionID())
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var or struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &or); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(or.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}
	return or.Choices[0].Message.Content, nil
}

// ─── Google Gemini ───────────────────────────────────────────────────

func (c *LLMCaller) buildGoogleBody(system, user string) map[string]any {
	return map[string]any{
		"contents": []map[string]any{
			{
				"role":  "user",
				"parts": []map[string]any{{"text": user}},
			},
		},
		"systemInstruction": map[string]any{
			"parts": []map[string]any{{"text": system}},
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": c.MaxTokens,
		},
	}
}

func (c *LLMCaller) callGoogle(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	body := c.buildGoogleBody(systemPrompt, userMessage)
	raw, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.BaseURL, c.Model, c.APIKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("google request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("google API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var gr struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text,omitempty"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBody, &gr); err != nil {
		return "", fmt.Errorf("parse google response: %w", err)
	}

	if len(gr.Candidates) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	var textParts []string
	for _, part := range gr.Candidates[0].Content.Parts {
		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}
	}
	return strings.Join(textParts, "\n"), nil
}
