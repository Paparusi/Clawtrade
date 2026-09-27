package agent

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"

	"github.com/clawtrade/clawtrade/internal/config"
	"github.com/clawtrade/clawtrade/internal/security"
)

// vaultPassword is the vault master passphrase used across the CLI
// (see storeInVault in cmd/clawtrade/main.go).
const vaultPassword = "clawtrade"

// openAICompatibleBaseURLs maps a provider to its OpenAI-compatible API root.
var openAICompatibleBaseURLs = map[string]string{
	"openai":            "https://api.openai.com/v1",
	"deepseek":          "https://api.deepseek.com/v1",
	"openrouter":        "https://openrouter.ai/api/v1",
	"ollama":            "http://localhost:11434/v1",
	"opencode":          "https://opencode.ai/zen/go/v1",
	"opencode-go":       "https://opencode.ai/zen/go/v1",
	"openai-compatible": "https://opencode.ai/zen/go/v1",
	"custom":            "https://opencode.ai/zen/go/v1",
}

// OpenAIBaseURL resolves the OpenAI-compatible API root for a provider.
// CLAWTRADE_LLM_BASE_URL overrides it for any provider.
func OpenAIBaseURL(provider string) string {
	if override := strings.TrimSpace(os.Getenv("CLAWTRADE_LLM_BASE_URL")); override != "" {
		return strings.TrimRight(override, "/")
	}
	return openAICompatibleBaseURLs[provider]
}

// VaultAPIKey looks up a provider's API key in the encrypted vault
// (namespace = provider name, key = "api_key"). Used when no environment
// variable is set. Returns "" when nothing is stored.
func VaultAPIKey(cfg *config.Config, provider string) string {
	if cfg == nil || provider == "" {
		return ""
	}
	path := cfg.Vault.Path
	if path == "" {
		path = "data/vault.enc"
	}
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	v, err := security.OpenVault(path, vaultPassword)
	if err != nil {
		return ""
	}
	namespaces := []string{provider, strings.ReplaceAll(provider, "-", "_"), "llm"}
	for _, ns := range namespaces {
		val, err := v.Get(ns, "api_key")
		if err == nil && strings.TrimSpace(val) != "" {
			return strings.TrimSpace(val)
		}
	}
	return ""
}

// llmSessionID identifies this process to providers that route by session
// (opencode.ai/zen requires the x-opencode-session header).
var llmSessionID = newSessionID()

// LLMSessionID returns the process-wide LLM session identifier.
func LLMSessionID() string {
	if override := strings.TrimSpace(os.Getenv("CLAWTRADE_LLM_SESSION")); override != "" {
		return override
	}
	return llmSessionID
}

func newSessionID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "clawtrade-session"
	}
	return "clawtrade-" + hex.EncodeToString(b)
}
