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
	Notifications NotificationsConfig `yaml:"notifications"`
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

type AgentConfig struct {
	Enabled       bool     `yaml:"enabled"`
	AutoTrade     bool     `yaml:"auto_trade"`
	Confirmation  bool     `yaml:"confirmation"`
	MinConfidence float64  `yaml:"min_confidence"`
	ScanInterval  int      `yaml:"scan_interval"`
	Watchlist     []string `yaml:"watchlist"`
	SubAgents     []string `yaml:"sub_agents"`
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
			SubAgents: []string{"market-scanner", "risk-manager", "portfolio-optimizer", "news-analyst", "execution-engine"},
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
			return fmt.Errorf("unknown agent field: %s", field)
		}
	case "notifications":
		// Support both notifications.telegram.token and telegram.token shorthand
		subParts := strings.SplitN(field, ".", 2)
		if len(subParts) == 2 {
			return c.setNotificationField(subParts[0], subParts[1], value)
		}
		return c.setNotificationField(field, "", value)
	case "telegram":
		return c.setNotificationField("telegram", field, value)
	case "discord":
		return c.setNotificationField("discord", field, value)
	default:
		return fmt.Errorf("unknown section: %s (available: server, database, vault, risk, agent, notifications, telegram, discord)", section)
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
