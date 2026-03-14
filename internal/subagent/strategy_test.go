package subagent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseStrategy(t *testing.T) {
	content := `---
name: Test Strategy
description: A test strategy
author: test
version: "1.0"
default_timeframes: ["1h", "4h"]
requires_data: ["candles", "volume"]
---

You are an expert test analyst.

## Analysis Steps
1. Check the trend
2. Find key levels
`
	s, err := ParseStrategy(content)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if s.Name != "Test Strategy" {
		t.Errorf("expected name 'Test Strategy', got %q", s.Name)
	}
	if len(s.Timeframes) != 2 {
		t.Errorf("expected 2 timeframes, got %d", len(s.Timeframes))
	}
	if s.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
	if s.Prompt[0] == '-' {
		t.Error("prompt should not include frontmatter")
	}
}

func TestLoadStrategies(t *testing.T) {
	dir := t.TempDir()

	s1 := `---
name: Strategy A
description: Test A
default_timeframes: ["1h"]
requires_data: ["candles"]
---

Analyze using method A.
`
	s2 := `---
name: Strategy B
description: Test B
default_timeframes: ["4h"]
requires_data: ["candles", "volume"]
---

Analyze using method B.
`
	os.WriteFile(filepath.Join(dir, "strat_a.md"), []byte(s1), 0644)
	os.WriteFile(filepath.Join(dir, "strat_b.md"), []byte(s2), 0644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not a strategy"), 0644)

	strategies, err := LoadStrategies(dir)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if len(strategies) != 2 {
		t.Errorf("expected 2 strategies, got %d", len(strategies))
	}
}
