package subagent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Strategy represents a trading analysis strategy loaded from a markdown file.
// The markdown file uses YAML frontmatter to define metadata and the body
// contains the prompt that teaches the LLM how to perform the analysis.
type Strategy struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Author      string   `yaml:"author"`
	Version     string   `yaml:"version"`
	Timeframes  []string `yaml:"default_timeframes"`
	Requires    []string `yaml:"requires_data"`
	Prompt      string   `yaml:"-"`
	Slug        string   `yaml:"-"`
}

// ParseStrategy parses a markdown string with YAML frontmatter into a Strategy.
// The frontmatter is delimited by "---" lines. Everything after the closing
// delimiter is used as the prompt body.
func ParseStrategy(content string) (*Strategy, error) {
	frontmatter, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, err
	}

	var s Strategy
	if err := yaml.Unmarshal([]byte(frontmatter), &s); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	s.Prompt = strings.TrimSpace(body)

	if s.Name == "" {
		return nil, fmt.Errorf("strategy missing required field: name")
	}

	return &s, nil
}

// LoadStrategies reads all .md files from the given directory and parses each
// one as a Strategy. Non-.md files are ignored. The Slug field is set to the
// filename without the .md extension.
func LoadStrategies(dir string) ([]Strategy, error) {
	pattern := filepath.Join(dir, "*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob strategies: %w", err)
	}

	var strategies []Strategy
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		s, err := ParseStrategy(string(data))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
		}

		s.Slug = strings.TrimSuffix(filepath.Base(path), ".md")
		strategies = append(strategies, *s)
	}

	return strategies, nil
}

// splitFrontmatter splits a markdown document into YAML frontmatter and body.
// The frontmatter must be enclosed between two "---" lines at the start.
func splitFrontmatter(content string) (string, string, error) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return "", "", fmt.Errorf("missing frontmatter delimiter")
	}

	// Find the closing delimiter (skip the opening "---" line).
	rest := content[3:]
	rest = strings.TrimLeft(rest, " \t")
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	} else if len(rest) > 1 && rest[0] == '\r' && rest[1] == '\n' {
		rest = rest[2:]
	}

	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", "", fmt.Errorf("missing closing frontmatter delimiter")
	}

	frontmatter := rest[:idx]
	body := rest[idx+4:] // skip "\n---"

	// Skip the rest of the closing delimiter line.
	if nl := strings.IndexByte(body, '\n'); nl >= 0 {
		body = body[nl+1:]
	} else {
		body = ""
	}

	return frontmatter, body, nil
}
