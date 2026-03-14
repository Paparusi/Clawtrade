package subagent

import (
	"context"
	"time"
)

type Event struct {
	Type     string         `json:"type"`
	Source   string         `json:"source"`
	Symbol   string         `json:"symbol,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
	Priority int            `json:"priority"`
	Time     time.Time      `json:"time"`
}

type SubAgentStatus struct {
	Name       string    `json:"name"`
	Running    bool      `json:"running"`
	LastRun    time.Time `json:"last_run"`
	RunCount   int       `json:"run_count"`
	ErrorCount int       `json:"error_count"`
	LastError  string    `json:"last_error,omitempty"`
}

type SubAgent interface {
	Name() string
	Start(ctx context.Context) error
	Stop() error
	Status() SubAgentStatus
}
