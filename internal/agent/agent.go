// internal/agent/agent.go
package agent

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/clawtrade/clawtrade/internal/adapter"
	"github.com/clawtrade/clawtrade/internal/config"
	"github.com/clawtrade/clawtrade/internal/engine"
	"github.com/clawtrade/clawtrade/internal/memory"
	"github.com/clawtrade/clawtrade/internal/risk"
)

const maxToolIterations = 10

// Message represents a chat message with optional tool use.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`  // assistant requested tool calls
	ToolResult string     `json:"tool_result,omitempty"` // result of a tool call
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// Response is what the agent returns to the API handler.
type Response struct {
	Content   string     `json:"content"`
	Model     string     `json:"model"`
	ToolsUsed []ToolCall `json:"tools_used,omitempty"`
	Usage     *Usage     `json:"usage,omitempty"`
}

// Usage tracks token consumption.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Agent orchestrates the LLM + tool use loop.
type Agent struct {
	cfg      *config.Config
	tools    *ToolRegistry
	context  *ContextBuilder
	memory   *memory.Store
}

// New creates a new AI trading agent.
func New(cfg *config.Config, adapters map[string]adapter.TradingAdapter, riskEngine *risk.Engine, mem *memory.Store, bus *engine.EventBus, db *sql.DB) *Agent {
	return &Agent{
		cfg:     cfg,
		tools:   NewToolRegistry(adapters, riskEngine, bus, db),
		context: NewContextBuilder(cfg, adapters, riskEngine, mem),
		memory:  mem,
	}
}

// SetMCPBridge connects external MCP tools to the agent.
func (a *Agent) SetMCPBridge(bridge MCPBridge) {
	a.tools.SetMCPBridge(bridge)
}

// SetAlertService connects the alert service to agent tools.
func (a *Agent) SetAlertService(svc AlertService) {
	a.tools.SetAlertService(svc)
}

// Run executes the agent loop: build context → send to LLM → handle tool calls → repeat.
func (a *Agent) Run(ctx context.Context, messages []Message) (*Response, error) {
	provider := a.cfg.Agent.Model.Provider()
	apiKey := a.cfg.Agent.Model.ResolveAPIKey()
	if apiKey == "" {
		apiKey = VaultAPIKey(a.cfg, provider)
	}
	model := a.cfg.Agent.Model.ModelName()

	if provider == "" || model == "" {
		return nil, fmt.Errorf("no LLM model configured. Run: clawtrade models setup")
	}
	if apiKey == "" && provider != "ollama" {
		return nil, fmt.Errorf("no API key found for %s", provider)
	}

	// Build rich system prompt with real-time data
	systemPrompt := a.context.BuildSystemPrompt(ctx)
	toolDefs := a.tools.Definitions()

	var allToolsUsed []ToolCall
	var totalUsage Usage

	switch provider {
	case "anthropic":
		return a.runAnthropic(ctx, apiKey, model, systemPrompt, messages, toolDefs, allToolsUsed, totalUsage)
	case "openai", "deepseek", "openrouter", "ollama", "opencode", "opencode-go", "openai-compatible", "custom":
		baseURL := OpenAIBaseURL(provider)
		return a.runOpenAI(ctx, baseURL, apiKey, model, systemPrompt, messages, toolDefs, allToolsUsed, totalUsage)
	case "google":
		return a.runGoogle(ctx, apiKey, model, systemPrompt, messages, toolDefs, allToolsUsed, totalUsage)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// ─── Anthropic Agent Loop ────────────────────────────────────────────

func (a *Agent) runAnthropic(ctx context.Context, apiKey, model, systemPrompt string, messages []Message, toolDefs []ToolDef, allToolsUsed []ToolCall, totalUsage Usage) (*Response, error) {
	// Build Anthropic messages
	var anthMsgs []map[string]any
	for _, m := range messages {
		if m.Role == "system" {
			continue
		}
		anthMsgs = append(anthMsgs, map[string]any{"role": m.Role, "content": m.Content})
	}

	// Build Anthropic tools
	var anthTools []map[string]any
	for _, t := range toolDefs {
		anthTools = append(anthTools, map[string]any{
			"name":         t.Name,
			"description":  t.Description,
			"input_schema": t.InputSchema,
		})
	}

	for i := 0; i < maxToolIterations; i++ {
		body := map[string]any{
			"model":      model,
			"max_tokens": a.cfg.Agent.Model.MaxTokens,
			"system":     systemPrompt,
			"messages":   anthMsgs,
			"tools":      anthTools,
		}

		raw, _ := json.Marshal(body)
		req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("anthropic request failed: %w", err)
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("anthropic API error (%d): %s", resp.StatusCode, string(respBody))
		}

		var ar struct {
			Content []struct {
				Type  string          `json:"type"`
				Text  string          `json:"text,omitempty"`
				ID    string          `json:"id,omitempty"`
				Name  string          `json:"name,omitempty"`
				Input json.RawMessage `json:"input,omitempty"`
			} `json:"content"`
			Model    string `json:"model"`
			StopReason string `json:"stop_reason"`
			Usage    struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(respBody, &ar); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}

		totalUsage.InputTokens += ar.Usage.InputTokens
		totalUsage.OutputTokens += ar.Usage.OutputTokens

		// Check if we have tool calls
		var textParts []string
		var toolCalls []ToolCall
		var assistantContent []map[string]any

		for _, block := range ar.Content {
			switch block.Type {
			case "text":
				textParts = append(textParts, block.Text)
				assistantContent = append(assistantContent, map[string]any{
					"type": "text",
					"text": block.Text,
				})
			case "tool_use":
				var input map[string]any
				json.Unmarshal(block.Input, &input)
				tc := ToolCall{ID: block.ID, Name: block.Name, Input: input}
				toolCalls = append(toolCalls, tc)
				allToolsUsed = append(allToolsUsed, tc)
				assistantContent = append(assistantContent, map[string]any{
					"type":  "tool_use",
					"id":    block.ID,
					"name":  block.Name,
					"input": input,
				})
			}
		}

		// If no tool calls, we're done
		if len(toolCalls) == 0 || ar.StopReason == "end_turn" {
			return &Response{
				Content:   strings.Join(textParts, "\n"),
				Model:     ar.Model,
				ToolsUsed: allToolsUsed,
				Usage:     &totalUsage,
			}, nil
		}

		// Add assistant message with tool_use blocks
		anthMsgs = append(anthMsgs, map[string]any{
			"role":    "assistant",
			"content": assistantContent,
		})

		// Execute tools and add results
		var toolResults []map[string]any
		for _, tc := range toolCalls {
			result := a.tools.Execute(ctx, tc)
			toolResults = append(toolResults, map[string]any{
				"type":        "tool_result",
				"tool_use_id": tc.ID,
				"content":     result.Content,
				"is_error":    result.IsError,
			})
		}

		anthMsgs = append(anthMsgs, map[string]any{
			"role":    "user",
			"content": toolResults,
		})
	}

	return &Response{
		Content:   "Agent reached maximum tool iterations. Please try a simpler request.",
		ToolsUsed: allToolsUsed,
		Usage:     &totalUsage,
	}, nil
}

// ─── OpenAI-compatible Agent Loop ────────────────────────────────────

func (a *Agent) runOpenAI(ctx context.Context, baseURL, apiKey, model, systemPrompt string, messages []Message, toolDefs []ToolDef, allToolsUsed []ToolCall, totalUsage Usage) (*Response, error) {
	// Build OpenAI messages
	oaiMsgs := []map[string]any{
		{"role": "system", "content": systemPrompt},
	}
	for _, m := range messages {
		if m.Role == "system" {
			continue
		}
		oaiMsgs = append(oaiMsgs, map[string]any{"role": m.Role, "content": m.Content})
	}

	// Build OpenAI tools (function calling format)
	var oaiTools []map[string]any
	for _, t := range toolDefs {
		oaiTools = append(oaiTools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.InputSchema,
			},
		})
	}

	for i := 0; i < maxToolIterations; i++ {
		body := map[string]any{
			"model":       model,
			"messages":    oaiMsgs,
			"tools":       oaiTools,
			"max_tokens":  a.cfg.Agent.Model.MaxTokens,
			"temperature": a.cfg.Agent.Model.Temperature,
		}

		raw, _ := json.Marshal(body)
		req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
		if strings.Contains(baseURL, "opencode.ai") {
			req.Header.Set("x-opencode-session", LLMSessionID())
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
		}

		var or struct {
			Choices []struct {
				Message struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls,omitempty"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Model string `json:"model"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(respBody, &or); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}

		totalUsage.InputTokens += or.Usage.PromptTokens
		totalUsage.OutputTokens += or.Usage.CompletionTokens

		if len(or.Choices) == 0 {
			return nil, fmt.Errorf("no response from model")
		}

		choice := or.Choices[0]

		// If no tool calls, we're done
		if len(choice.Message.ToolCalls) == 0 || choice.FinishReason == "stop" {
			return &Response{
				Content:   choice.Message.Content,
				Model:     or.Model,
				ToolsUsed: allToolsUsed,
				Usage:     &totalUsage,
			}, nil
		}

		// Add assistant message with tool calls
		assistantMsg := map[string]any{
			"role":       "assistant",
			"content":    choice.Message.Content,
			"tool_calls": choice.Message.ToolCalls,
		}
		oaiMsgs = append(oaiMsgs, assistantMsg)

		// Execute tools and add results
		for _, tc := range choice.Message.ToolCalls {
			var input map[string]any
			json.Unmarshal([]byte(tc.Function.Arguments), &input)

			toolCall := ToolCall{ID: tc.ID, Name: tc.Function.Name, Input: input}
			allToolsUsed = append(allToolsUsed, toolCall)

			result := a.tools.Execute(ctx, toolCall)
			oaiMsgs = append(oaiMsgs, map[string]any{
				"role":         "tool",
				"tool_call_id": tc.ID,
				"content":      result.Content,
			})
		}
	}

	return &Response{
		Content:   "Agent reached maximum tool iterations.",
		ToolsUsed: allToolsUsed,
		Usage:     &totalUsage,
	}, nil
}

// ─── Google Gemini Agent Loop ────────────────────────────────────────

func (a *Agent) runGoogle(ctx context.Context, apiKey, model, systemPrompt string, messages []Message, toolDefs []ToolDef, allToolsUsed []ToolCall, totalUsage Usage) (*Response, error) {
	// Build Gemini contents
	var contents []map[string]any
	for _, m := range messages {
		if m.Role == "system" {
			continue
		}
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, map[string]any{
			"role":  role,
			"parts": []map[string]any{{"text": m.Content}},
		})
	}

	// Build Gemini tools
	var funcDecls []map[string]any
	for _, t := range toolDefs {
		funcDecls = append(funcDecls, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  t.InputSchema,
		})
	}

	geminiTools := []map[string]any{
		{"function_declarations": funcDecls},
	}

	for i := 0; i < maxToolIterations; i++ {
		body := map[string]any{
			"contents":          contents,
			"systemInstruction": map[string]any{"parts": []map[string]any{{"text": systemPrompt}}},
			"tools":             geminiTools,
			"generationConfig": map[string]any{
				"maxOutputTokens": a.cfg.Agent.Model.MaxTokens,
				"temperature":     a.cfg.Agent.Model.Temperature,
			},
		}

		raw, _ := json.Marshal(body)
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("google request failed: %w", err)
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("google API error (%d): %s", resp.StatusCode, string(respBody))
		}

		var gr struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text         string          `json:"text,omitempty"`
						FunctionCall *struct {
							Name string          `json:"name"`
							Args json.RawMessage `json:"args"`
						} `json:"functionCall,omitempty"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
			UsageMetadata struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
			} `json:"usageMetadata"`
		}
		if err := json.Unmarshal(respBody, &gr); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}

		totalUsage.InputTokens += gr.UsageMetadata.PromptTokenCount
		totalUsage.OutputTokens += gr.UsageMetadata.CandidatesTokenCount

		if len(gr.Candidates) == 0 {
			return nil, fmt.Errorf("no response from model")
		}

		var textParts []string
		var functionCalls []struct {
			Name string
			Args map[string]any
		}

		for _, part := range gr.Candidates[0].Content.Parts {
			if part.Text != "" {
				textParts = append(textParts, part.Text)
			}
			if part.FunctionCall != nil {
				var args map[string]any
				json.Unmarshal(part.FunctionCall.Args, &args)
				functionCalls = append(functionCalls, struct {
					Name string
					Args map[string]any
				}{Name: part.FunctionCall.Name, Args: args})
			}
		}

		if len(functionCalls) == 0 {
			return &Response{
				Content:   strings.Join(textParts, "\n"),
				Model:     model,
				ToolsUsed: allToolsUsed,
				Usage:     &totalUsage,
			}, nil
		}

		// Add model response to contents
		var modelParts []map[string]any
		for _, t := range textParts {
			modelParts = append(modelParts, map[string]any{"text": t})
		}
		for _, fc := range functionCalls {
			modelParts = append(modelParts, map[string]any{
				"functionCall": map[string]any{"name": fc.Name, "args": fc.Args},
			})
		}
		contents = append(contents, map[string]any{
			"role":  "model",
			"parts": modelParts,
		})

		// Execute and add function responses
		var responseParts []map[string]any
		for _, fc := range functionCalls {
			tc := ToolCall{ID: fc.Name, Name: fc.Name, Input: fc.Args}
			allToolsUsed = append(allToolsUsed, tc)
			result := a.tools.Execute(ctx, tc)
			responseParts = append(responseParts, map[string]any{
				"functionResponse": map[string]any{
					"name":     fc.Name,
					"response": map[string]any{"result": result.Content},
				},
			})
		}
		contents = append(contents, map[string]any{
			"role":  "user",
			"parts": responseParts,
		})
	}

	return &Response{
		Content:   "Agent reached maximum tool iterations.",
		ToolsUsed: allToolsUsed,
		Usage:     &totalUsage,
	}, nil
}
