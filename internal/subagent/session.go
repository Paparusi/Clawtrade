package subagent

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
)

// subSessionID identifies this process to providers that route by session
// (opencode.ai/zen requires the x-opencode-session header).
var subSessionID = newSubSessionID()

// llmSessionID returns the process-wide session identifier.
func llmSessionID() string { return subSessionID }

func newSubSessionID() string {
	if override := strings.TrimSpace(os.Getenv("CLAWTRADE_LLM_SESSION")); override != "" {
		return override
	}
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "clawtrade-subagent"
	}
	return "clawtrade-sub-" + hex.EncodeToString(b)
}
