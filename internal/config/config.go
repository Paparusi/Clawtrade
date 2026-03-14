package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Database      DatabaseConfig      `yaml:"database"`
	Vault         VaultConfig         `yaml:"vault"`
	Risk          RiskConfig          `yaml:"risk"`
	Exchanges     []ExchangeEntry     `yaml:"exchanges"`
	Agent         AgentConfig         `yaml:"agent"`
	MCP           MCPConfig           `yaml:"mcp"`
	Notifications NotificationsConfig `yaml:"notifications"`
}

type MCPConfig struct {
	Servers []MCPServerEntry `yaml:"servers"`
}

type MCPServerEntry struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
	Env     []string `yaml:"env,omitempty"`
	Enabled bool     `yaml:"enabled"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type VaultConfig struct {
	Path string `yaml:"path"`
}

type RiskConfig struct {
	MaxRiskPerTrade float64 `yaml:"max_risk_per_trade"`
	MaxDailyLoss    float64 `yaml:"max_daily_loss"`
	MaxPositions    int     `yaml:"max_positions"`
	MaxLeverage     float64 `yaml:"max_leverage"`
	DefaultMode     string  `yaml:"default_mode"`
}

type ExchangeEntry struct {
	Name    string            `yaml:"name"`
	Type    string            `yaml:"type"`
	Enabled bool              `yaml:"enabled"`
	Fields  map[string]string `yaml:"fields"`
}

type SubAgentEntry struct {
	Name         string `yaml:"name"`
	Enabled      bool   `yaml:"enabled"`
	ScanInterval int    `yaml:"scan_interval"`
	Model        string `yaml:"model"`
}

type AnalysisConfig struct {
	StrategiesDir    string             `yaml:"strategies_dir"`
	ActiveStrategies []string           `yaml:"active_strategies"`
	Weights          map[string]float64 `yaml:"weights"`
	MinConfluence    int                `yaml:"min_confluence"`
	Timeframes       []string           `yaml:"timeframes"`
	ExpertModel      string             `yaml:"expert_model"`
	SynthesisModel   string             `yaml:"synthesis_model"`
}

type AgentConfig struct {
	Enabled       bool             `yaml:"enabled"`
	AutoTrade     bool             `yaml:"auto_trade"`
	Confirmation  bool             `yaml:"confirmation"`
	MinConfidence float64          `yaml:"min_confidence"`
	ScanInterval  int              `yaml:"scan_interval"`
	Watchlist     []string         `yaml:"watchlist"`
	SubAgents     []SubAgentEntry  `yaml:"sub_agents"`
	Model         ModelConfig      `yaml:"model"`
	Analysis      AnalysisConfig   `yaml:"analysis"`
}

type ModelConfig struct {
	Primary     string   `yaml:"primary"`
	Fallbacks   []string `yaml:"fallbacks,omitempty"`
	MaxTokens   int      `yaml:"max_tokens"`
	Temperature float64  `yaml:"temperature"`
}

// ResolveAPIKey returns the API key for the configured provider.
// Priority: vault > env var > empty.
func (m *ModelConfig) ResolveAPIKey() string {
	provider := m.Provider()
	envMap := map[string]string{
		"anthropic":  "ANTHROPIC_API_KEY",
		"openai":     "OPENAI_API_KEY",
		"openrouter": "OPENROUTER_API_KEY",
		"ollama":     "OLLAMA_API_KEY",
		"deepseek":   "DEEPSEEK_API_KEY",
		"google":     "GOOGLE_AI_API_KEY",
	}
	if envKey, ok := envMap[provider]; ok {
		if val := os.Getenv(envKey); val != "" {
			return val
		}
	}
	return ""
}

// Provider extracts the provider name from "provider/model" format.
func (m *ModelConfig) Provider() string {
	if m.Primary == "" {
		return ""
	}
	parts := strings.SplitN(m.Primary, "/", 2)
	return parts[0]
}

// ModelName extracts the model name from "provider/model" format.
func (m *ModelConfig) ModelName() string {
	if m.Primary == "" {
		return ""
	}
	parts := strings.SplitN(m.Primary, "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return m.Primary
}

type NotificationsConfig struct {
	Telegram TelegramConfig `yaml:"telegram"`
	Discord  DiscordConfig  `yaml:"discord"`
	Alerts   AlertsConfig   `yaml:"alerts"`
}

type TelegramConfig struct {
	Enabled bool   `yaml:"enabled"`
	Token   string `yaml:"token"`
	ChatID  string `yaml:"chat_id"`
}

type DiscordConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
}

type AlertsConfig struct {
	TradeExecuted bool `yaml:"trade_executed"`
	RiskAlert     bool `yaml:"risk_alert"`
	PnlUpdate     bool `yaml:"pnl_update"`
	SystemAlert   bool `yaml:"system_alert"`
}

func defaultConfig() *Config {
	return &Config{
		Server:   ServerConfig{Host: "127.0.0.1", Port: 9090},
		Database: DatabaseConfig{Path: "data/clawtrade.db"},
		Vault:    VaultConfig{Path: "data/vault.enc"},
		Risk: RiskConfig{
			MaxRiskPerTrade: 0.02, MaxDailyLoss: 0.05,
			MaxPositions: 5, MaxLeverage: 10, DefaultMode: "paper",
		},
		Agent: AgentConfig{
			Enabled: true, AutoTrade: false, Confirmation: true,
			MinConfidence: 0.7, ScanInterval: 30,
			Watchlist: []string{"BTC/USDT", "ETH/USDT"},
			SubAgents: []SubAgentEntry{
				{Name: "market-analyst", Enabled: true, ScanInterval: 300},
				{Name: "devils-advocate", Enabled: true},
				{Name: "narrative", Enabled: true, ScanInterval: 900},
				{Name: "reflection", Enabled: true, ScanInterval: 300},
				{Name: "correlation", Enabled: true, ScanInterval: 600},
			},
			Model: ModelConfig{
				Primary:     "",
				MaxTokens:   4096,
				Temperature: 0.7,
			},
			Analysis: AnalysisConfig{
				StrategiesDir:    "",
				ActiveStrategies: []string{"price_action", "smc"},
				Weights:          map[string]float64{"price_action": 0.5, "smc": 0.5},
				MinConfluence:    2,
				Timeframes:       []string{"15m", "1h", "4h"},
			},
		},
		Notifications: NotificationsConfig{
			Alerts: AlertsConfig{
				TradeExecuted: true,
				RiskAlert:     true,
				PnlUpdate:     false,
				SystemAlert:   true,
			},
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := defaultConfig()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func Save(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// SetField sets a config value by dot-separated key path (e.g. "server.port", "risk.max_leverage")
func (c *Config) SetField(key, value string) error {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid key %q, use format: section.field (e.g. server.port)", key)
	}
	section, field := parts[0], parts[1]

	switch section {
	case "server":
		switch field {
		case "host":
			c.Server.Host = value
		case "port":
			v, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("port must be a number: %w", err)
			}
			c.Server.Port = v
		default:
			return fmt.Errorf("unknown server field: %s", field)
		}
	case "database":
		switch field {
		case "path":
			c.Database.Path = value
		default:
			return fmt.Errorf("unknown database field: %s", field)
		}
	case "vault":
		switch field {
		case "path":
			c.Vault.Path = value
		default:
			return fmt.Errorf("unknown vault field: %s", field)
		}
	case "risk":
		switch field {
		case "max_risk_per_trade":
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("must be a number: %w", err)
			}
			c.Risk.MaxRiskPerTrade = v
		case "max_daily_loss":
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("must be a number: %w", err)
			}
			c.Risk.MaxDailyLoss = v
		case "max_positions":
			v, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("must be an integer: %w", err)
			}
			c.Risk.MaxPositions = v
		case "max_leverage":
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("must be a number: %w", err)
			}
			c.Risk.MaxLeverage = v
		case "default_mode":
			if value != "paper" && value != "live" {
				return fmt.Errorf("mode must be 'paper' or 'live'")
			}
			c.Risk.DefaultMode = value
		default:
			return fmt.Errorf("unknown risk field: %s", field)
		}
	case "agent":
		switch field {
		case "enabled":
			c.Agent.Enabled = value == "true" || value == "1"
		case "auto_trade":
			c.Agent.AutoTrade = value == "true" || value == "1"
		case "confirmation":
			c.Agent.Confirmation = value == "true" || value == "1"
		case "min_confidence":
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("must be a number: %w", err)
			}
			c.Agent.MinConfidence = v
		case "scan_interval":
			v, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("must be an integer: %w", err)
			}
			c.Agent.ScanInterval = v
		default:
			// Check for agent.model.* sub-fields
			if strings.HasPrefix(field, "model.") {
				return c.setModelField(strings.TrimPrefix(field, "model."), value)
			}
			return fmt.Errorf("unknown agent field: %s", field)
		}
	case "notifications":
		// Support both notifications.telegram.token and telegram.token shorthand
		subParts := strings.SplitN(field, ".", 2)
		if len(subParts) == 2 {
			return c.setNotificationField(subParts[0], subParts[1], value)
		}
		return c.setNotificationField(field, "", value)
	case "models":
		return c.setModelField(field, value)
	case "telegram":
		return c.setNotificationField("telegram", field, value)
	case "discord":
		return c.setNotificationField("discord", field, value)
	default:
		return fmt.Errorf("unknown section: %s (available: server, database, vault, risk, agent, models, notifications, telegram, discord)", section)
	}
	return nil
}

func (c *Config) setModelField(field, value string) error {
	switch field {
	case "primary":
		// Validate provider/model format
		if !strings.Contains(value, "/") && value != "" {
			return fmt.Errorf("model must be in provider/model format (e.g. anthropic/claude-sonnet-4-6)")
		}
		c.Agent.Model.Primary = value
	case "max_tokens":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("must be an integer: %w", err)
		}
		c.Agent.Model.MaxTokens = v
	case "temperature":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("must be a number: %w", err)
		}
		if v < 0 || v > 2 {
			return fmt.Errorf("temperature must be between 0 and 2")
		}
		c.Agent.Model.Temperature = v
	default:
		return fmt.Errorf("unknown model field: %s (available: primary, max_tokens, temperature)", field)
	}
	return nil
}

func (c *Config) setNotificationField(channel, field, value string) error {
	switch channel {
	case "telegram":
		switch field {
		case "enabled":
			c.Notifications.Telegram.Enabled = value == "true" || value == "1"
		case "token":
			c.Notifications.Telegram.Token = value
		case "chat_id":
			c.Notifications.Telegram.ChatID = value
		default:
			return fmt.Errorf("unknown telegram field: %s (available: enabled, token, chat_id)", field)
		}
	case "discord":
		switch field {
		case "enabled":
			c.Notifications.Discord.Enabled = value == "true" || value == "1"
		case "webhook_url":
			c.Notifications.Discord.WebhookURL = value
		default:
			return fmt.Errorf("unknown discord field: %s (available: enabled, webhook_url)", field)
		}
	case "alerts":
		switch field {
		case "trade_executed":
			c.Notifications.Alerts.TradeExecuted = value == "true" || value == "1"
		case "risk_alert":
			c.Notifications.Alerts.RiskAlert = value == "true" || value == "1"
		case "pnl_update":
			c.Notifications.Alerts.PnlUpdate = value == "true" || value == "1"
		case "system_alert":
			c.Notifications.Alerts.SystemAlert = value == "true" || value == "1"
		default:
			return fmt.Errorf("unknown alerts field: %s", field)
		}
	default:
		return fmt.Errorf("unknown notification channel: %s (available: telegram, discord, alerts)", channel)
	}
	return nil
}
