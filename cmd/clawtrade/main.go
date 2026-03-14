package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/clawtrade/clawtrade/internal"
	"github.com/clawtrade/clawtrade/internal/adapter"
	"github.com/clawtrade/clawtrade/internal/adapter/binance"
	"github.com/clawtrade/clawtrade/internal/adapter/bybit"
	"github.com/clawtrade/clawtrade/internal/api"
	"github.com/clawtrade/clawtrade/internal/config"
	"github.com/clawtrade/clawtrade/internal/database"
	"github.com/clawtrade/clawtrade/internal/engine"
	"github.com/clawtrade/clawtrade/internal/memory"
	"github.com/clawtrade/clawtrade/internal/risk"
	"github.com/clawtrade/clawtrade/internal/security"
)

var configPath = "config/default.yaml"

func init() {
	if envPath := os.Getenv("CLAWTRADE_CONFIG"); envPath != "" {
		configPath = envPath
	}
}

var reader = bufio.NewReader(os.Stdin)

func prompt(label string) string {
	fmt.Print(label)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func promptDefault(label, def string) string {
	fmt.Printf("%s [%s]: ", label, def)
	line, _ := reader.ReadString('\n')
	val := strings.TrimSpace(line)
	if val == "" {
		return def
	}
	return val
}

func promptYN(label string, def bool) bool {
	defStr := "Y/n"
	if !def {
		defStr = "y/N"
	}
	fmt.Printf("%s [%s]: ", label, defStr)
	line, _ := reader.ReadString('\n')
	val := strings.TrimSpace(strings.ToLower(line))
	if val == "" {
		return def
	}
	return val == "y" || val == "yes"
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "version":
		fmt.Printf("Clawtrade %s\n", internal.Version)
	case "serve":
		err = serve()
	case "init":
		err = initSetup()
	case "config":
		err = handleConfig(os.Args[2:])
	case "exchange":
		err = handleExchange(os.Args[2:])
	case "risk":
		err = handleRisk(os.Args[2:])
	case "agent":
		err = handleAgent(os.Args[2:])
	case "models":
		err = handleModels(os.Args[2:])
	case "telegram":
		err = handleTelegram(os.Args[2:])
	case "notify":
		err = handleNotify(os.Args[2:])
	case "status":
		err = handleStatus()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`Clawtrade - AI Trading Agent Platform
Version: %s

Usage: clawtrade <command> [options]

Core:
  serve                  Start the Clawtrade server
  init                   Interactive setup wizard (configure everything)
  version                Show version
  status                 Show system status

Configuration:
  config show            Show all configuration
  config set K V         Set a config value (e.g. server.port 8080)
  config reset           Reset to defaults

Exchanges:
  exchange list          List configured exchanges
  exchange add NAME      Add exchange credentials (interactive)
  exchange remove NAME   Remove an exchange
  exchange test NAME     Validate exchange config

Risk:
  risk show              Show risk parameters
  risk set K V           Set a risk parameter

Agent:
  agent show             Show agent configuration
  agent set K V          Set an agent parameter
  agent watchlist add S  Add symbol to watchlist
  agent watchlist rm S   Remove symbol from watchlist

Models (LLM Provider):
  models setup           Interactive LLM provider setup
  models set P/M         Set model (e.g. anthropic/claude-sonnet-4-6)
  models list            List available providers
  models status          Show current model config

Notifications:
  telegram setup         Setup Telegram bot (interactive)
  telegram test          Send test message
  telegram show          Show Telegram config
  notify show            Show all notification settings
  notify set K V         Set notification config

Environment:
  CLAWTRADE_CONFIG       Config file path (default: config/default.yaml)
`, internal.Version)
}

// ─── serve ───────────────────────────────────────────────────────────

func serve() error {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("Warning: Could not load config from %s, using defaults\n", configPath)
		cfg, _ = config.Load("")
	}

	dataDir := filepath.Dir(cfg.Database.Path)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	bus := engine.NewEventBus()
	memStore := memory.NewStore(db)
	auditLog := security.NewAuditLog(db)
	adapters := make(map[string]adapter.TradingAdapter)

	// Initialize configured exchanges
	for _, ex := range cfg.Exchanges {
		if !ex.Enabled {
			continue
		}
		switch ex.Name {
		case "binance":
			apiKey := ex.Fields["api_key"]
			apiSecret := ex.Fields["api_secret"]
			ba := binance.New(apiKey, apiSecret)
			if ex.Fields["environment"] == "testnet" {
				ba.SetTestnet(true)
			}
			adapters["binance"] = ba
			fmt.Printf("Exchange: binance loaded\n")
		case "bybit":
			apiKey := ex.Fields["api_key"]
			apiSecret := ex.Fields["api_secret"]
			ba := bybit.New(apiKey, apiSecret)
			if ex.Fields["environment"] == "testnet" {
				ba.SetTestnet(true)
			}
			adapters["bybit"] = ba
			fmt.Printf("Exchange: bybit loaded\n")
		}
	}

	// Initialize risk engine
	riskEngine := risk.NewEngine(risk.RiskLimits{
		MaxPositionSizePct:  cfg.Risk.MaxRiskPerTrade * 5, // 10% default
		MaxTotalExposurePct: 0.50,
		MaxRiskPerTradePct:  cfg.Risk.MaxRiskPerTrade,
		MaxOpenPositions:    cfg.Risk.MaxPositions,
		MaxDailyLossPct:     cfg.Risk.MaxDailyLoss,
		MaxOrderSize:        10000,
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := api.NewServer(cfg, bus, memStore, auditLog, adapters, riskEngine)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	fmt.Printf("Clawtrade %s starting...\n", internal.Version)
	fmt.Printf("API server: http://%s\n", addr)
	fmt.Printf("Database: %s\n", cfg.Database.Path)

	if cfg.Notifications.Telegram.Enabled && cfg.Notifications.Telegram.Token != "" {
		fmt.Println("Telegram: enabled")
	}

	// Start WebSocket price streaming for configured exchanges
	for name, adp := range adapters {
		if ba, ok := adp.(*binance.Adapter); ok {
			ba.OnPrice(func(price adapter.Price) {
				bus.Publish(engine.Event{
					Type: engine.EventPriceUpdate,
					Data: map[string]any{
						"symbol":     price.Symbol,
						"bid":        price.Bid,
						"ask":        price.Ask,
						"last":       price.Last,
						"volume_24h": price.Volume24h,
						"exchange":   "binance",
					},
				})
			})
			if err := ba.SubscribePrices(ctx, cfg.Agent.Watchlist); err != nil {
				fmt.Printf("Warning: %s WebSocket failed: %v\n", name, err)
			} else {
				fmt.Printf("WebSocket: %s streaming %d symbols\n", name, len(cfg.Agent.Watchlist))
			}
		}
	}

	if err := srv.Start(ctx, addr); err != nil && err.Error() != "http: Server closed" {
		return fmt.Errorf("server error: %w", err)
	}

	fmt.Println("Goodbye!")
	return nil
}

// ─── init (comprehensive setup wizard) ───────────────────────────────

func initSetup() error {
	fmt.Println()
	fmt.Println("  ╔═══════════════════════════════════════════╗")
	fmt.Println("  ║     Clawtrade Setup Wizard                ║")
	fmt.Println("  ║     AI Trading Agent Platform             ║")
	fmt.Println("  ╚═══════════════════════════════════════════╝")
	fmt.Println()

	// 1. Initialize directories and database
	fmt.Println("  [1/7] Initializing system...")
	fmt.Println("  ─────────────────────────────")

	if err := os.MkdirAll("data", 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if err := os.MkdirAll("config", 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	dbPath := "data/clawtrade.db"
	db, err := database.Open(dbPath)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	db.Close()
	fmt.Printf("  ✓ Database: %s\n", dbPath)
	fmt.Printf("  ✓ Config: %s\n", configPath)
	fmt.Println()

	cfg, _ := config.Load(configPath)

	// 2. Server
	fmt.Println("  [2/7] Server Configuration")
	fmt.Println("  ─────────────────────────────")
	cfg.Server.Host = promptDefault("  Bind host", cfg.Server.Host)
	portStr := promptDefault("  Bind port", fmt.Sprintf("%d", cfg.Server.Port))
	fmt.Sscanf(portStr, "%d", &cfg.Server.Port)
	fmt.Println()

	// 3. Trading mode & Risk
	fmt.Println("  [3/7] Risk Management")
	fmt.Println("  ─────────────────────────────")
	fmt.Println("  Trading modes:")
	fmt.Println("    1. paper  - Simulated trading (no real money)")
	fmt.Println("    2. live   - Real trading (use with caution)")
	mode := promptDefault("  Select mode (paper/live)", cfg.Risk.DefaultMode)
	if mode == "1" {
		mode = "paper"
	} else if mode == "2" {
		mode = "live"
	}
	cfg.Risk.DefaultMode = mode

	if mode == "live" {
		fmt.Println()
		fmt.Println("  ⚠ LIVE MODE - Configure safety limits:")
		riskStr := promptDefault("  Max risk per trade %", fmt.Sprintf("%.0f", cfg.Risk.MaxRiskPerTrade*100))
		fmt.Sscanf(riskStr, "%f", &cfg.Risk.MaxRiskPerTrade)
		if cfg.Risk.MaxRiskPerTrade > 1 {
			cfg.Risk.MaxRiskPerTrade /= 100
		}

		lossStr := promptDefault("  Max daily loss %", fmt.Sprintf("%.0f", cfg.Risk.MaxDailyLoss*100))
		fmt.Sscanf(lossStr, "%f", &cfg.Risk.MaxDailyLoss)
		if cfg.Risk.MaxDailyLoss > 1 {
			cfg.Risk.MaxDailyLoss /= 100
		}

		posStr := promptDefault("  Max open positions", fmt.Sprintf("%d", cfg.Risk.MaxPositions))
		fmt.Sscanf(posStr, "%d", &cfg.Risk.MaxPositions)

		levStr := promptDefault("  Max leverage", fmt.Sprintf("%.0f", cfg.Risk.MaxLeverage))
		fmt.Sscanf(levStr, "%f", &cfg.Risk.MaxLeverage)
	} else {
		fmt.Println("  ✓ Paper trading mode - safe to experiment")
	}
	fmt.Println()

	// 4. Exchange setup
	fmt.Println("  [4/7] Exchange Setup")
	fmt.Println("  ─────────────────────────────")
	if promptYN("  Add an exchange now?", true) {
		for {
			fmt.Println()
			fmt.Println("  Available exchanges:")
			fmt.Println("    1. binance       Binance (Crypto CEX)")
			fmt.Println("    2. bybit         Bybit (Crypto CEX)")
			fmt.Println("    3. okx           OKX (Crypto CEX)")
			fmt.Println("    4. mt5           MetaTrader 5 (Forex/CFD)")
			fmt.Println("    5. ibkr          Interactive Brokers (Stocks)")
			fmt.Println("    6. hyperliquid   Hyperliquid (Crypto DEX)")
			fmt.Println("    7. uniswap       Uniswap (DeFi DEX)")
			fmt.Println()
			choice := prompt("  Select exchange (name or number): ")

			// Convert number to name
			switch choice {
			case "1":
				choice = "binance"
			case "2":
				choice = "bybit"
			case "3":
				choice = "okx"
			case "4":
				choice = "mt5"
			case "5":
				choice = "ibkr"
			case "6":
				choice = "hyperliquid"
			case "7":
				choice = "uniswap"
			}

			choice = strings.ToLower(choice)
			fields, ok := exchangeFields[choice]
			if !ok {
				fmt.Printf("  ✗ Unknown exchange: %s\n", choice)
				continue
			}

			fmt.Printf("\n  Configuring %s (%s)\n", choice, getExchangeType(choice))
			fieldValues := make(map[string]string)

			for _, f := range fields {
				label := fmt.Sprintf("  %s", f.label)
				if f.required {
					label += " *"
				}
				val := prompt(label + ": ")
				if val == "" && f.required {
					fmt.Printf("  ✗ %s is required, skipping exchange\n", f.label)
					fieldValues = nil
					break
				}
				if val != "" {
					fieldValues[f.key] = val
				}
			}

			if fieldValues != nil {
				entry := config.ExchangeEntry{
					Name:    choice,
					Type:    getExchangeType(choice),
					Enabled: true,
					Fields:  fieldValues,
				}
				cfg.Exchanges = append(cfg.Exchanges, entry)
				fmt.Printf("  ✓ %s added\n", choice)

				// Store secrets in vault
				if err := storeInVault(cfg.Vault.Path, choice, fieldValues); err != nil {
					fmt.Printf("  ⚠ Vault: %v (credentials in config file)\n", err)
				}
			}

			fmt.Println()
			if !promptYN("  Add another exchange?", false) {
				break
			}
		}
	} else {
		fmt.Println("  Skipped. Add later: clawtrade exchange add <name>")
	}
	fmt.Println()

	// 5. LLM Provider
	fmt.Println("  [5/7] AI Model Provider")
	fmt.Println("  ─────────────────────────────")
	fmt.Println("  Your AI agent needs an LLM to analyze markets and make decisions.")
	fmt.Println()
	fmt.Println("  Popular choices:")
	fmt.Println("    1. anthropic     Anthropic Claude (recommended)")
	fmt.Println("    2. openai        OpenAI GPT")
	fmt.Println("    3. openrouter    OpenRouter (multi-model access)")
	fmt.Println("    4. deepseek      DeepSeek")
	fmt.Println("    5. google        Google Gemini")
	fmt.Println("    6. ollama        Ollama (local, free)")
	fmt.Println("    7. skip          Configure later")
	fmt.Println()
	providerChoice := promptDefault("  Select provider", "1")

	skipModel := false
	switch providerChoice {
	case "7", "skip":
		skipModel = true
		fmt.Println("  Skipped. Setup later: clawtrade models setup")
	default:
		var selectedProvider *providerInfo
		for i, p := range providers {
			if providerChoice == fmt.Sprintf("%d", i+1) || strings.EqualFold(providerChoice, p.name) {
				selectedProvider = &providers[i]
				break
			}
		}

		if selectedProvider == nil {
			fmt.Printf("  ✗ Unknown provider: %s, skipping\n", providerChoice)
			skipModel = true
		} else {
			fmt.Printf("\n  Setting up %s\n", selectedProvider.label)

			if !selectedProvider.isLocal {
				existingKey := os.Getenv(selectedProvider.envKey)
				if existingKey != "" {
					masked := existingKey[:8] + "..." + existingKey[len(existingKey)-4:]
					fmt.Printf("  Found %s: %s\n", selectedProvider.envKey, masked)
				} else {
					apiKey := prompt("  API Key: ")
					if apiKey != "" {
						if err := storeInVault(cfg.Vault.Path, selectedProvider.name, map[string]string{"api_key": apiKey}); err != nil {
							fmt.Printf("  ⚠ Vault: %v\n", err)
						} else {
							fmt.Println("  ✓ API key stored in vault")
						}
					} else {
						fmt.Printf("  ⚠ No key provided. Set %s env var or run: clawtrade models setup\n", selectedProvider.envKey)
					}
				}
			} else {
				fmt.Printf("  Make sure %s is running at %s\n", selectedProvider.name, selectedProvider.baseURL)
			}

			// Pick default model
			defaultModel := selectedProvider.models[0]
			fmt.Printf("\n  Models: %s\n", strings.Join(selectedProvider.models, ", "))
			modelPick := promptDefault("  Model", defaultModel)
			// Check if it's a number
			for i, m := range selectedProvider.models {
				if modelPick == fmt.Sprintf("%d", i+1) {
					modelPick = m
					break
				}
			}
			if !strings.Contains(modelPick, "/") {
				modelPick = selectedProvider.name + "/" + modelPick
			}
			cfg.Agent.Model.Primary = modelPick
			fmt.Printf("  ✓ Model: %s\n", modelPick)
		}
	}
	_ = skipModel
	fmt.Println()

	// 6. Notifications (Telegram / Discord)
	fmt.Println("  [6/7] Notifications")
	fmt.Println("  ─────────────────────────────")
	fmt.Println("  Get alerts on trades, risk events, and system status.")
	fmt.Println()

	// Telegram
	if promptYN("  Setup Telegram bot?", false) {
		fmt.Println()
		fmt.Println("  How to create a Telegram bot:")
		fmt.Println("    1. Open Telegram, search @BotFather")
		fmt.Println("    2. Send /newbot and follow instructions")
		fmt.Println("    3. Copy the bot token")
		fmt.Println("    4. Start a chat with your bot")
		fmt.Println("    5. Get your Chat ID from @userinfobot")
		fmt.Println()

		token := prompt("  Bot Token: ")
		chatID := prompt("  Chat ID: ")

		if token != "" && chatID != "" {
			cfg.Notifications.Telegram.Enabled = true
			cfg.Notifications.Telegram.Token = token
			cfg.Notifications.Telegram.ChatID = chatID
			fmt.Println("  ✓ Telegram configured")

			// Store token in vault
			if err := storeInVault(cfg.Vault.Path, "telegram", map[string]string{"token": token}); err != nil {
				fmt.Printf("  ⚠ Vault: %v\n", err)
			}
		} else {
			fmt.Println("  ✗ Skipped (token and chat ID required)")
		}
	}

	// Discord
	if promptYN("  Setup Discord webhook?", false) {
		fmt.Println()
		fmt.Println("  How to create a Discord webhook:")
		fmt.Println("    1. Open Discord channel settings")
		fmt.Println("    2. Go to Integrations -> Webhooks")
		fmt.Println("    3. Create webhook and copy URL")
		fmt.Println()

		webhookURL := prompt("  Webhook URL: ")
		if webhookURL != "" {
			cfg.Notifications.Discord.Enabled = true
			cfg.Notifications.Discord.WebhookURL = webhookURL
			fmt.Println("  ✓ Discord configured")
		} else {
			fmt.Println("  ✗ Skipped")
		}
	}

	if !cfg.Notifications.Telegram.Enabled && !cfg.Notifications.Discord.Enabled {
		fmt.Println("  No notifications configured. Add later:")
		fmt.Println("    clawtrade telegram setup")
		fmt.Println("    clawtrade notify set discord.webhook_url <url>")
	}

	// Alert types
	fmt.Println()
	fmt.Println("  Alert types (all enabled by default):")
	cfg.Notifications.Alerts.TradeExecuted = promptYN("    Trade executions?", true)
	cfg.Notifications.Alerts.RiskAlert = promptYN("    Risk alerts?", true)
	cfg.Notifications.Alerts.PnlUpdate = promptYN("    P&L updates?", false)
	cfg.Notifications.Alerts.SystemAlert = promptYN("    System alerts?", true)
	fmt.Println()

	// 7. Agent config
	fmt.Println("  [7/7] AI Agent")
	fmt.Println("  ─────────────────────────────")
	cfg.Agent.Enabled = promptYN("  Enable AI agent?", true)

	if cfg.Agent.Enabled {
		cfg.Agent.AutoTrade = promptYN("  Auto-execute trades?", false)
		if cfg.Agent.AutoTrade {
			cfg.Agent.Confirmation = promptYN("  Require confirmation before trade?", true)
		}

		fmt.Println()
		fmt.Println("  Watchlist (symbols the agent monitors):")
		fmt.Printf("  Current: %s\n", strings.Join(cfg.Agent.Watchlist, ", "))
		extra := prompt("  Add symbols (comma-separated, or Enter to keep): ")
		if extra != "" {
			for _, s := range strings.Split(extra, ",") {
				sym := strings.TrimSpace(strings.ToUpper(s))
				if sym != "" {
					found := false
					for _, existing := range cfg.Agent.Watchlist {
						if existing == sym {
							found = true
							break
						}
					}
					if !found {
						cfg.Agent.Watchlist = append(cfg.Agent.Watchlist, sym)
					}
				}
			}
		}
	}
	fmt.Println()

	// Save everything
	if err := config.Save(cfg, configPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// Summary
	fmt.Println("  ╔═══════════════════════════════════════════╗")
	fmt.Println("  ║           Setup Complete!                 ║")
	fmt.Println("  ╚═══════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Config saved: %s\n", configPath)
	fmt.Printf("  Mode:         %s\n", cfg.Risk.DefaultMode)
	fmt.Printf("  Exchanges:    %d configured\n", len(cfg.Exchanges))

	notifyCount := 0
	if cfg.Notifications.Telegram.Enabled {
		notifyCount++
	}
	if cfg.Notifications.Discord.Enabled {
		notifyCount++
	}
	fmt.Printf("  Notifications: %d channels\n", notifyCount)
	fmt.Printf("  Agent:        %v\n", cfg.Agent.Enabled)
	modelStr := cfg.Agent.Model.Primary
	if modelStr == "" {
		modelStr = "(not set — run: clawtrade models setup)"
	}
	fmt.Printf("  Model:        %s\n", modelStr)
	fmt.Printf("  Watchlist:    %s\n", strings.Join(cfg.Agent.Watchlist, ", "))
	fmt.Println()
	fmt.Println("  Start trading:")
	fmt.Println("    clawtrade serve")
	fmt.Println()
	fmt.Println("  Open dashboard:")
	fmt.Printf("    http://%s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Println()

	return nil
}

// ─── config ──────────────────────────────────────────────────────────

func handleConfig(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: clawtrade config <show|set|reset>")
		return nil
	}

	switch args[0] {
	case "show":
		return configShow()
	case "set":
		if len(args) < 3 {
			fmt.Println("Usage: clawtrade config set <key> <value>")
			fmt.Println()
			fmt.Println("Keys:")
			fmt.Println("  server.host              Server bind host")
			fmt.Println("  server.port              Server bind port")
			fmt.Println("  database.path            Database file path")
			fmt.Println("  vault.path               Encrypted vault path")
			fmt.Println("  risk.max_risk_per_trade   Max risk per trade (0.0-1.0)")
			fmt.Println("  risk.max_daily_loss       Max daily loss (0.0-1.0)")
			fmt.Println("  risk.max_positions        Max concurrent positions")
			fmt.Println("  risk.max_leverage         Max leverage multiplier")
			fmt.Println("  risk.default_mode         Trading mode (paper|live)")
			fmt.Println("  agent.enabled             Enable AI agent (true|false)")
			fmt.Println("  agent.auto_trade          Auto-execute trades (true|false)")
			fmt.Println("  agent.confirmation        Require confirmation (true|false)")
			fmt.Println("  agent.min_confidence      Min confidence (0.0-1.0)")
			fmt.Println("  agent.scan_interval       Scan interval (seconds)")
			fmt.Println("  telegram.enabled          Enable Telegram (true|false)")
			fmt.Println("  telegram.token            Telegram bot token")
			fmt.Println("  telegram.chat_id          Telegram chat ID")
			fmt.Println("  discord.enabled           Enable Discord (true|false)")
			fmt.Println("  discord.webhook_url       Discord webhook URL")
			return nil
		}
		return configSet(args[1], args[2])
	case "reset":
		return configReset()
	default:
		fmt.Println("Usage: clawtrade config <show|set|reset>")
		return nil
	}
}

func configShow() error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	fmt.Println("┌──────────────────────────────────────────────┐")
	fmt.Println("│          Clawtrade Configuration              │")
	fmt.Println("├──────────────────────────────────────────────┤")
	fmt.Println("│ Server                                        │")
	fmt.Printf("│   host:                %-23s│\n", cfg.Server.Host)
	fmt.Printf("│   port:                %-23d│\n", cfg.Server.Port)
	fmt.Println("│                                               │")
	fmt.Println("│ Database                                      │")
	fmt.Printf("│   path:                %-23s│\n", cfg.Database.Path)
	fmt.Println("│                                               │")
	fmt.Println("│ Risk                                          │")
	fmt.Printf("│   max_risk_per_trade:  %-23s│\n", fmt.Sprintf("%.1f%%", cfg.Risk.MaxRiskPerTrade*100))
	fmt.Printf("│   max_daily_loss:      %-23s│\n", fmt.Sprintf("%.1f%%", cfg.Risk.MaxDailyLoss*100))
	fmt.Printf("│   max_positions:       %-23d│\n", cfg.Risk.MaxPositions)
	fmt.Printf("│   max_leverage:        %-23s│\n", fmt.Sprintf("%.0fx", cfg.Risk.MaxLeverage))
	fmt.Printf("│   default_mode:        %-23s│\n", cfg.Risk.DefaultMode)
	fmt.Println("│                                               │")
	fmt.Println("│ Agent                                         │")
	fmt.Printf("│   enabled:             %-23v│\n", cfg.Agent.Enabled)
	fmt.Printf("│   auto_trade:          %-23v│\n", cfg.Agent.AutoTrade)
	fmt.Printf("│   confirmation:        %-23v│\n", cfg.Agent.Confirmation)
	fmt.Printf("│   min_confidence:      %-23s│\n", fmt.Sprintf("%.0f%%", cfg.Agent.MinConfidence*100))
	fmt.Printf("│   scan_interval:       %-23s│\n", fmt.Sprintf("%ds", cfg.Agent.ScanInterval))
	if len(cfg.Agent.Watchlist) > 0 {
		wl := strings.Join(cfg.Agent.Watchlist, ", ")
		if len(wl) > 23 {
			wl = wl[:20] + "..."
		}
		fmt.Printf("│   watchlist:           %-23s│\n", wl)
	}
	modelDisplay := cfg.Agent.Model.Primary
	if modelDisplay == "" {
		modelDisplay = "(not set)"
	}
	if len(modelDisplay) > 23 {
		modelDisplay = modelDisplay[:20] + "..."
	}
	fmt.Printf("│   model:               %-23s│\n", modelDisplay)
	fmt.Println("│                                               │")
	fmt.Println("│ Exchanges                                     │")
	if len(cfg.Exchanges) == 0 {
		fmt.Println("│   (none configured)                           │")
	} else {
		for _, ex := range cfg.Exchanges {
			status := "disabled"
			if ex.Enabled {
				status = "enabled"
			}
			fmt.Printf("│   %-12s %-8s %-21s│\n", ex.Name, "["+ex.Type+"]", status)
		}
	}
	fmt.Println("│                                               │")
	fmt.Println("│ Notifications                                 │")
	tgStatus := "disabled"
	if cfg.Notifications.Telegram.Enabled {
		tgStatus = "enabled"
	}
	dcStatus := "disabled"
	if cfg.Notifications.Discord.Enabled {
		dcStatus = "enabled"
	}
	fmt.Printf("│   telegram:            %-23s│\n", tgStatus)
	fmt.Printf("│   discord:             %-23s│\n", dcStatus)
	a := cfg.Notifications.Alerts
	alerts := []string{}
	if a.TradeExecuted {
		alerts = append(alerts, "trades")
	}
	if a.RiskAlert {
		alerts = append(alerts, "risk")
	}
	if a.PnlUpdate {
		alerts = append(alerts, "pnl")
	}
	if a.SystemAlert {
		alerts = append(alerts, "system")
	}
	fmt.Printf("│   alerts:              %-23s│\n", strings.Join(alerts, ", "))
	fmt.Println("└──────────────────────────────────────────────┘")
	fmt.Printf("\nConfig file: %s\n", configPath)
	return nil
}

func configSet(key, value string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if err := cfg.SetField(key, value); err != nil {
		return err
	}

	if err := config.Save(cfg, configPath); err != nil {
		return err
	}

	fmt.Printf("✓ Set %s = %s\n", key, value)
	return nil
}

func configReset() error {
	cfg, _ := config.Load("")
	if err := config.Save(cfg, configPath); err != nil {
		return err
	}
	fmt.Println("Configuration reset to defaults.")
	return nil
}

// ─── exchange ────────────────────────────────────────────────────────

var exchangeFields = map[string][]struct {
	key      string
	label    string
	secret   bool
	required bool
}{
	"binance": {
		{key: "api_key", label: "API Key", required: true},
		{key: "api_secret", label: "API Secret", secret: true, required: true},
		{key: "signature", label: "Signature Algorithm (hmac/rsa/ed25519)", required: true},
		{key: "environment", label: "Environment (production/testnet)"},
	},
	"bybit": {
		{key: "api_key", label: "API Key", required: true},
		{key: "api_secret", label: "API Secret", secret: true, required: true},
		{key: "key_type", label: "Key Type (hmac/rsa)"},
		{key: "environment", label: "Environment (production/testnet/demo)"},
	},
	"okx": {
		{key: "api_key", label: "API Key", required: true},
		{key: "api_secret", label: "API Secret", secret: true, required: true},
		{key: "passphrase", label: "Passphrase", secret: true, required: true},
		{key: "environment", label: "Environment (production/demo)"},
	},
	"mt5": {
		{key: "login", label: "Login (Account Number)", required: true},
		{key: "password", label: "Password", secret: true, required: true},
		{key: "server", label: "Server (e.g. Forex4you-Demo)", required: true},
		{key: "account_type", label: "Account Type (demo/live)"},
	},
	"ibkr": {
		{key: "username", label: "Username", required: true},
		{key: "account_id", label: "Account ID", required: true},
		{key: "connection", label: "Connection (gateway/tws)"},
		{key: "account_type", label: "Account Type (paper/live)"},
		{key: "host", label: "Host (default: 127.0.0.1)"},
		{key: "port", label: "Port (default: auto by connection+type)"},
		{key: "client_id", label: "Client ID (default: 1)"},
	},
	"hyperliquid": {
		{key: "wallet_address", label: "Wallet Address", required: true},
		{key: "private_key", label: "Private Key (or API Wallet Key)", secret: true, required: true},
		{key: "environment", label: "Environment (mainnet/testnet)"},
		{key: "use_api_wallet", label: "Use API Wallet (true/false)"},
		{key: "api_wallet_address", label: "API Wallet Address (if using API wallet)"},
	},
	"uniswap": {
		{key: "wallet_address", label: "Wallet Address", required: true},
		{key: "private_key", label: "Private Key", secret: true, required: true},
		{key: "chain", label: "Chain (ethereum/arbitrum/polygon/base/optimism)"},
		{key: "rpc_endpoint", label: "RPC Endpoint"},
		{key: "slippage", label: "Slippage Tolerance % (default: 0.5)"},
		{key: "gas_mode", label: "Gas Mode (standard/fast/aggressive)"},
	},
}

func handleExchange(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: clawtrade exchange <list|add|remove|test>")
		return nil
	}

	switch args[0] {
	case "list":
		return exchangeList()
	case "add":
		if len(args) < 2 {
			fmt.Println("Usage: clawtrade exchange add <name>")
			fmt.Println()
			fmt.Println("Supported exchanges:")
			fmt.Println("  binance       Binance (CEX)")
			fmt.Println("  bybit         Bybit (CEX)")
			fmt.Println("  okx           OKX (CEX)")
			fmt.Println("  mt5           MetaTrader 5 (Forex/CFD)")
			fmt.Println("  ibkr          Interactive Brokers (Stocks/Futures)")
			fmt.Println("  hyperliquid   Hyperliquid (DEX)")
			fmt.Println("  uniswap       Uniswap (DEX)")
			return nil
		}
		return exchangeAdd(strings.ToLower(args[1]))
	case "remove":
		if len(args) < 2 {
			fmt.Println("Usage: clawtrade exchange remove <name>")
			return nil
		}
		return exchangeRemove(strings.ToLower(args[1]))
	case "test":
		if len(args) < 2 {
			fmt.Println("Usage: clawtrade exchange test <name>")
			return nil
		}
		return exchangeTest(strings.ToLower(args[1]))
	default:
		fmt.Println("Usage: clawtrade exchange <list|add|remove|test>")
		return nil
	}
}

func exchangeList() error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if len(cfg.Exchanges) == 0 {
		fmt.Println("No exchanges configured.")
		fmt.Println()
		fmt.Println("Add one with: clawtrade exchange add <name>")
		fmt.Println("Supported: binance, bybit, okx, mt5, ibkr, hyperliquid, uniswap")
		return nil
	}

	fmt.Println("Configured Exchanges:")
	fmt.Println()
	for i, ex := range cfg.Exchanges {
		status := "\033[31m●\033[0m disabled"
		if ex.Enabled {
			status = "\033[32m●\033[0m enabled"
		}
		fmt.Printf("  %d. %-14s %-8s  %s\n", i+1, ex.Name, "["+ex.Type+"]", status)

		for k, v := range ex.Fields {
			if k == "api_secret" || k == "private_key" || k == "password" || k == "passphrase" {
				fmt.Printf("     %-20s %s\n", k+":", "••••••••")
			} else {
				fmt.Printf("     %-20s %s\n", k+":", v)
			}
		}
		fmt.Println()
	}

	return nil
}

func exchangeAdd(name string) error {
	fields, ok := exchangeFields[name]
	if !ok {
		return fmt.Errorf("unsupported exchange: %s\nSupported: binance, bybit, okx, mt5, ibkr, hyperliquid, uniswap", name)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	for _, ex := range cfg.Exchanges {
		if ex.Name == name {
			return fmt.Errorf("exchange %q already configured. Remove first: clawtrade exchange remove %s", name, name)
		}
	}

	exchangeType := getExchangeType(name)

	fmt.Printf("Adding %s (%s)\n", name, exchangeType)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()

	fieldValues := make(map[string]string)

	for _, f := range fields {
		label := fmt.Sprintf("  %s", f.label)
		if f.required {
			label += " *"
		}
		val := prompt(label + ": ")
		if val == "" && f.required {
			return fmt.Errorf("field %q is required", f.key)
		}
		if val != "" {
			fieldValues[f.key] = val
		}
	}

	entry := config.ExchangeEntry{
		Name:    name,
		Type:    exchangeType,
		Enabled: true,
		Fields:  fieldValues,
	}

	cfg.Exchanges = append(cfg.Exchanges, entry)

	if err := config.Save(cfg, configPath); err != nil {
		return err
	}

	if err := storeInVault(cfg.Vault.Path, name, fieldValues); err != nil {
		fmt.Printf("Warning: Could not store credentials in vault: %v\n", err)
		fmt.Println("Credentials are saved in config file (less secure)")
	}

	fmt.Println()
	fmt.Printf("✓ Exchange %s added and enabled\n", name)
	fmt.Printf("  Test connection: clawtrade exchange test %s\n", name)

	return nil
}

func exchangeRemove(name string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	found := false
	var updated []config.ExchangeEntry
	for _, ex := range cfg.Exchanges {
		if ex.Name == name {
			found = true
			continue
		}
		updated = append(updated, ex)
	}

	if !found {
		return fmt.Errorf("exchange %q not found", name)
	}

	cfg.Exchanges = updated

	if err := config.Save(cfg, configPath); err != nil {
		return err
	}

	fmt.Printf("✓ Exchange %s removed\n", name)
	return nil
}

func exchangeTest(name string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	var found *config.ExchangeEntry
	for _, ex := range cfg.Exchanges {
		if ex.Name == name {
			found = &ex
			break
		}
	}

	if found == nil {
		return fmt.Errorf("exchange %q not configured. Add it first: clawtrade exchange add %s", name, name)
	}

	fmt.Printf("Testing connection to %s...\n", name)
	fmt.Println()

	fields, ok := exchangeFields[name]
	if !ok {
		fmt.Println("  ⚠ Unknown exchange type, skipping field validation")
	} else {
		allOk := true
		for _, f := range fields {
			if f.required {
				val, exists := found.Fields[f.key]
				if !exists || val == "" {
					fmt.Printf("  ✗ Missing required field: %s\n", f.label)
					allOk = false
				} else {
					if f.secret {
						fmt.Printf("  ✓ %s: ••••••••\n", f.label)
					} else {
						fmt.Printf("  ✓ %s: %s\n", f.label, val)
					}
				}
			}
		}
		if !allOk {
			fmt.Println()
			fmt.Println("  ✗ Configuration incomplete")
			return nil
		}
	}

	fmt.Println()
	fmt.Printf("  ✓ Configuration valid for %s\n", name)
	fmt.Println("  ⚠ Live connection test requires running server (clawtrade serve)")

	return nil
}

// ─── risk ────────────────────────────────────────────────────────────

func handleRisk(args []string) error {
	if len(args) == 0 {
		return riskShow()
	}

	switch args[0] {
	case "show":
		return riskShow()
	case "set":
		if len(args) < 3 {
			fmt.Println("Usage: clawtrade risk set <key> <value>")
			fmt.Println()
			fmt.Println("Keys:")
			fmt.Println("  max_risk_per_trade  Max risk per trade (0.0-1.0)")
			fmt.Println("  max_daily_loss      Max daily loss (0.0-1.0)")
			fmt.Println("  max_positions       Max concurrent positions")
			fmt.Println("  max_leverage        Max leverage multiplier")
			fmt.Println("  default_mode        Trading mode (paper|live)")
			return nil
		}
		return configSet("risk."+args[1], args[2])
	default:
		return riskShow()
	}
}

func riskShow() error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	r := cfg.Risk
	fmt.Println("Risk Management")
	fmt.Println(strings.Repeat("─", 35))
	fmt.Printf("  Trading Mode:        %s\n", r.DefaultMode)
	fmt.Printf("  Max Risk/Trade:      %.1f%%\n", r.MaxRiskPerTrade*100)
	fmt.Printf("  Max Daily Loss:      %.1f%%\n", r.MaxDailyLoss*100)
	fmt.Printf("  Max Positions:       %d\n", r.MaxPositions)
	fmt.Printf("  Max Leverage:        %.0fx\n", r.MaxLeverage)
	fmt.Println()

	if r.DefaultMode == "live" {
		fmt.Println("  ⚠ LIVE TRADING ENABLED - Real money at risk")
	} else {
		fmt.Println("  ✓ Paper trading mode - No real money at risk")
	}

	return nil
}

// ─── agent ───────────────────────────────────────────────────────────

func handleAgent(args []string) error {
	if len(args) == 0 {
		return agentShow()
	}

	switch args[0] {
	case "show":
		return agentShow()
	case "set":
		if len(args) < 3 {
			fmt.Println("Usage: clawtrade agent set <key> <value>")
			fmt.Println()
			fmt.Println("Keys:")
			fmt.Println("  enabled         Enable AI agent (true|false)")
			fmt.Println("  auto_trade      Auto-execute trades (true|false)")
			fmt.Println("  confirmation    Require confirmation (true|false)")
			fmt.Println("  min_confidence  Min confidence threshold (0.0-1.0)")
			fmt.Println("  scan_interval   Market scan interval in seconds")
			return nil
		}
		return configSet("agent."+args[1], args[2])
	case "watchlist":
		return handleWatchlist(args[1:])
	default:
		return agentShow()
	}
}

func agentShow() error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	a := cfg.Agent
	fmt.Println("AI Agent Configuration")
	fmt.Println(strings.Repeat("─", 35))

	enabledStr := "\033[31mdisabled\033[0m"
	if a.Enabled {
		enabledStr = "\033[32menabled\033[0m"
	}
	fmt.Printf("  Status:              %s\n", enabledStr)
	fmt.Printf("  Auto Trade:          %v\n", a.AutoTrade)
	fmt.Printf("  Order Confirmation:  %v\n", a.Confirmation)
	fmt.Printf("  Min Confidence:      %.0f%%\n", a.MinConfidence*100)
	fmt.Printf("  Scan Interval:       %ds\n", a.ScanInterval)
	fmt.Println()

	// Model
	fmt.Println()
	if a.Model.Primary != "" {
		fmt.Printf("  Model:               %s\n", a.Model.Primary)
		fmt.Printf("  Max Tokens:          %d\n", a.Model.MaxTokens)
		fmt.Printf("  Temperature:         %.1f\n", a.Model.Temperature)
	} else {
		fmt.Println("  Model:               \033[33mnot configured\033[0m")
		fmt.Println("  Run: clawtrade models setup")
	}
	fmt.Println()

	fmt.Println("  Sub-Agents:")
	for _, sa := range a.SubAgents {
		fmt.Printf("    • %s\n", sa)
	}
	fmt.Println()

	fmt.Println("  Watchlist:")
	if len(a.Watchlist) == 0 {
		fmt.Println("    (empty)")
	} else {
		for _, sym := range a.Watchlist {
			fmt.Printf("    • %s\n", sym)
		}
	}

	return nil
}

func handleWatchlist(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: clawtrade agent watchlist <add|remove> <SYMBOL>")
		return nil
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	switch args[0] {
	case "add":
		if len(args) < 2 {
			return fmt.Errorf("specify symbol: clawtrade agent watchlist add BTC/USDT")
		}
		symbol := strings.ToUpper(args[1])
		for _, s := range cfg.Agent.Watchlist {
			if s == symbol {
				fmt.Printf("%s already in watchlist\n", symbol)
				return nil
			}
		}
		cfg.Agent.Watchlist = append(cfg.Agent.Watchlist, symbol)
		if err := config.Save(cfg, configPath); err != nil {
			return err
		}
		fmt.Printf("✓ Added %s to watchlist\n", symbol)

	case "remove", "rm":
		if len(args) < 2 {
			return fmt.Errorf("specify symbol: clawtrade agent watchlist remove BTC/USDT")
		}
		symbol := strings.ToUpper(args[1])
		var updated []string
		found := false
		for _, s := range cfg.Agent.Watchlist {
			if s == symbol {
				found = true
				continue
			}
			updated = append(updated, s)
		}
		if !found {
			return fmt.Errorf("%s not in watchlist", symbol)
		}
		cfg.Agent.Watchlist = updated
		if err := config.Save(cfg, configPath); err != nil {
			return err
		}
		fmt.Printf("✓ Removed %s from watchlist\n", symbol)

	default:
		fmt.Println("Usage: clawtrade agent watchlist <add|remove> <SYMBOL>")
	}

	return nil
}

// ─── models (LLM provider) ───────────────────────────────────────────

type providerInfo struct {
	name     string
	label    string
	envKey   string
	models   []string
	isLocal  bool
	baseURL  string
}

var providers = []providerInfo{
	{name: "anthropic", label: "Anthropic (Claude)", envKey: "ANTHROPIC_API_KEY", models: []string{
		"anthropic/claude-opus-4-6", "anthropic/claude-sonnet-4-6", "anthropic/claude-haiku-4-5",
	}},
	{name: "openai", label: "OpenAI (GPT)", envKey: "OPENAI_API_KEY", models: []string{
		"openai/gpt-4o", "openai/gpt-4o-mini", "openai/o3-mini",
	}},
	{name: "openrouter", label: "OpenRouter (multi-model)", envKey: "OPENROUTER_API_KEY", models: []string{
		"openrouter/anthropic/claude-sonnet-4-6", "openrouter/openai/gpt-4o", "openrouter/meta-llama/llama-4-scout",
	}},
	{name: "deepseek", label: "DeepSeek", envKey: "DEEPSEEK_API_KEY", models: []string{
		"deepseek/deepseek-chat", "deepseek/deepseek-reasoner",
	}},
	{name: "google", label: "Google AI (Gemini)", envKey: "GOOGLE_AI_API_KEY", models: []string{
		"google/gemini-2.5-pro", "google/gemini-2.5-flash",
	}},
	{name: "ollama", label: "Ollama (local)", envKey: "OLLAMA_API_KEY", isLocal: true, baseURL: "http://localhost:11434", models: []string{
		"ollama/llama4", "ollama/qwen3", "ollama/deepseek-v3", "ollama/codellama",
	}},
}

func handleModels(args []string) error {
	if len(args) == 0 {
		return modelsStatus()
	}

	switch args[0] {
	case "setup":
		return modelsSetup()
	case "set":
		if len(args) < 2 {
			fmt.Println("Usage: clawtrade models set <provider/model>")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  clawtrade models set anthropic/claude-sonnet-4-6")
			fmt.Println("  clawtrade models set openai/gpt-4o")
			fmt.Println("  clawtrade models set ollama/llama4")
			return nil
		}
		return modelsSet(args[1])
	case "list":
		return modelsList()
	case "status":
		return modelsStatus()
	default:
		// Treat as shorthand: clawtrade models anthropic/claude-sonnet-4-6
		if strings.Contains(args[0], "/") {
			return modelsSet(args[0])
		}
		fmt.Println("Usage: clawtrade models <setup|set|list|status>")
		return nil
	}
}

func modelsSetup() error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("  ╔═══════════════════════════════════════════╗")
	fmt.Println("  ║       LLM Provider Setup                  ║")
	fmt.Println("  ╚═══════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("  Select your AI provider:")
	fmt.Println()

	for i, p := range providers {
		envStatus := ""
		if val := os.Getenv(p.envKey); val != "" {
			envStatus = " \033[32m(key found in env)\033[0m"
		}
		localTag := ""
		if p.isLocal {
			localTag = " [local/free]"
		}
		fmt.Printf("    %d. %-30s%s%s\n", i+1, p.label+localTag, envStatus, "")
	}

	fmt.Println()
	choice := prompt("  Select provider (number or name): ")

	var selected *providerInfo
	// Try number first
	for i, p := range providers {
		if choice == fmt.Sprintf("%d", i+1) || strings.EqualFold(choice, p.name) {
			selected = &providers[i]
			break
		}
	}

	if selected == nil {
		return fmt.Errorf("unknown provider: %s", choice)
	}

	fmt.Printf("\n  Configuring %s\n", selected.label)
	fmt.Println("  " + strings.Repeat("─", 35))

	// API key
	if !selected.isLocal {
		existingKey := os.Getenv(selected.envKey)
		if existingKey != "" {
			masked := existingKey[:8] + "..." + existingKey[len(existingKey)-4:]
			fmt.Printf("  Found %s in environment: %s\n", selected.envKey, masked)
			if !promptYN("  Use this key?", true) {
				existingKey = ""
			}
		}

		if existingKey == "" {
			fmt.Printf("\n  Enter your %s API key\n", selected.label)
			fmt.Printf("  (env: %s)\n", selected.envKey)
			apiKey := prompt("  API Key: ")
			if apiKey == "" {
				return fmt.Errorf("API key is required for %s", selected.label)
			}
			// Store in vault
			if err := storeInVault(cfg.Vault.Path, selected.name, map[string]string{"api_key": apiKey}); err != nil {
				fmt.Printf("  ⚠ Vault: %v\n", err)
			} else {
				fmt.Println("  ✓ API key stored in encrypted vault")
			}
		}
	} else {
		fmt.Println("  No API key needed for local models.")
		fmt.Printf("  Make sure %s is running at %s\n", selected.name, selected.baseURL)
	}

	// Select model
	fmt.Println()
	fmt.Println("  Available models:")
	for i, m := range selected.models {
		fmt.Printf("    %d. %s\n", i+1, m)
	}
	fmt.Println()

	modelChoice := promptDefault("  Select model (number or full name)", "1")
	var modelStr string

	for i, m := range selected.models {
		if modelChoice == fmt.Sprintf("%d", i+1) {
			modelStr = m
			break
		}
	}
	if modelStr == "" {
		if strings.Contains(modelChoice, "/") {
			modelStr = modelChoice
		} else {
			modelStr = selected.name + "/" + modelChoice
		}
	}

	cfg.Agent.Model.Primary = modelStr
	if err := config.Save(cfg, configPath); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  ✓ Model set: %s\n", modelStr)
	fmt.Println()
	fmt.Println("  Your AI agent is ready! Start with:")
	fmt.Println("    clawtrade serve")
	fmt.Println()

	return nil
}

func modelsSet(model string) error {
	if !strings.Contains(model, "/") {
		return fmt.Errorf("model must be in provider/model format (e.g. anthropic/claude-sonnet-4-6)")
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	cfg.Agent.Model.Primary = model
	if err := config.Save(cfg, configPath); err != nil {
		return err
	}

	fmt.Printf("✓ Model set: %s\n", model)
	return nil
}

func modelsList() error {
	fmt.Println("Available LLM Providers")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()

	for _, p := range providers {
		envStatus := "\033[31m✗ no key\033[0m"
		if p.isLocal {
			envStatus = "\033[33m● local\033[0m"
		} else if val := os.Getenv(p.envKey); val != "" {
			envStatus = "\033[32m✓ key set\033[0m"
		}
		fmt.Printf("  %-25s %s\n", p.label, envStatus)
		if !p.isLocal {
			fmt.Printf("    env: %s\n", p.envKey)
		}
		for _, m := range p.models {
			fmt.Printf("    • %s\n", m)
		}
		fmt.Println()
	}

	return nil
}

func modelsStatus() error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	m := cfg.Agent.Model
	fmt.Println("Model Configuration")
	fmt.Println(strings.Repeat("─", 40))

	if m.Primary == "" {
		fmt.Println("  ⚠ No model configured")
		fmt.Println()
		fmt.Println("  Quick setup:")
		fmt.Println("    clawtrade models setup")
		fmt.Println()
		fmt.Println("  Or set directly:")
		fmt.Println("    clawtrade models set anthropic/claude-sonnet-4-6")
		return nil
	}

	fmt.Printf("  Primary:     %s\n", m.Primary)
	fmt.Printf("  Provider:    %s\n", m.Provider())
	fmt.Printf("  Model:       %s\n", m.ModelName())
	fmt.Printf("  Max Tokens:  %d\n", m.MaxTokens)
	fmt.Printf("  Temperature: %.1f\n", m.Temperature)
	fmt.Println()

	// Check API key availability
	provider := m.Provider()
	for _, p := range providers {
		if p.name == provider {
			if p.isLocal {
				fmt.Printf("  API Key:     not required (local)\n")
			} else if val := os.Getenv(p.envKey); val != "" {
				fmt.Printf("  API Key:     \033[32m✓ found in %s\033[0m\n", p.envKey)
			} else {
				fmt.Printf("  API Key:     \033[33m⚠ not in env (%s)\033[0m\n", p.envKey)
				fmt.Println("               May be in vault — check with: clawtrade config show")
			}
			break
		}
	}

	if len(m.Fallbacks) > 0 {
		fmt.Println()
		fmt.Println("  Fallbacks:")
		for _, f := range m.Fallbacks {
			fmt.Printf("    • %s\n", f)
		}
	}

	return nil
}

// ─── telegram ────────────────────────────────────────────────────────

func handleTelegram(args []string) error {
	if len(args) == 0 {
		return telegramShow()
	}

	switch args[0] {
	case "setup":
		return telegramSetup()
	case "test":
		return telegramTest()
	case "show":
		return telegramShow()
	case "disable":
		return configSet("telegram.enabled", "false")
	case "enable":
		return configSet("telegram.enabled", "true")
	default:
		fmt.Println("Usage: clawtrade telegram <setup|test|show|enable|disable>")
		return nil
	}
}

func telegramSetup() error {
	fmt.Println("Telegram Bot Setup")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()
	fmt.Println("Steps to create your Telegram bot:")
	fmt.Println()
	fmt.Println("  1. Open Telegram and search for @BotFather")
	fmt.Println("  2. Send /newbot and follow the instructions")
	fmt.Println("  3. BotFather will give you a bot token like:")
	fmt.Println("     123456789:ABCdefGHIjklMNOpqrSTUvwxYZ")
	fmt.Println()
	fmt.Println("  4. Start a chat with your new bot")
	fmt.Println("  5. To get your Chat ID, message @userinfobot")
	fmt.Println("     or @RawDataBot, it will reply with your ID")
	fmt.Println()

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	token := prompt("  Bot Token: ")
	if token == "" {
		return fmt.Errorf("bot token is required")
	}

	chatID := prompt("  Chat ID: ")
	if chatID == "" {
		return fmt.Errorf("chat ID is required")
	}

	cfg.Notifications.Telegram.Enabled = true
	cfg.Notifications.Telegram.Token = token
	cfg.Notifications.Telegram.ChatID = chatID

	if err := config.Save(cfg, configPath); err != nil {
		return err
	}

	// Store token in vault
	if err := storeInVault(cfg.Vault.Path, "telegram", map[string]string{"token": token}); err != nil {
		fmt.Printf("  ⚠ Vault: %v\n", err)
	}

	fmt.Println()
	fmt.Println("✓ Telegram bot configured and enabled")
	fmt.Println()
	fmt.Println("  Test it: clawtrade telegram test")
	fmt.Println()
	fmt.Println("  Bot commands available after 'clawtrade serve':")
	fmt.Println("    /portfolio  - View portfolio")
	fmt.Println("    /positions  - View open positions")
	fmt.Println("    /price BTC  - Get price")
	fmt.Println("    /analyze    - AI analysis")
	fmt.Println("    /risk       - Risk status")
	fmt.Println("    /briefing   - Daily briefing")

	return nil
}

func telegramTest() error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if cfg.Notifications.Telegram.Token == "" {
		return fmt.Errorf("telegram not configured. Run: clawtrade telegram setup")
	}

	fmt.Println("Sending test message to Telegram...")

	// Use the bot package to send a test message
	token := cfg.Notifications.Telegram.Token
	chatID := cfg.Notifications.Telegram.ChatID

	// Quick HTTP call to Telegram API
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	payload := fmt.Sprintf(`{"chat_id":%s,"text":"Clawtrade connected!\n\nYour trading bot is ready.\nVersion: %s\nMode: %s"}`,
		chatID, internal.Version, cfg.Risk.DefaultMode)

	resp, httpErr := httpPost(apiURL, payload)
	if httpErr != nil {
		return fmt.Errorf("failed to send: %v", httpErr)
	}

	if resp {
		fmt.Println("✓ Test message sent! Check your Telegram.")
	} else {
		fmt.Println("✗ Failed to send message. Check token and chat ID.")
	}

	return nil
}

func telegramShow() error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	tg := cfg.Notifications.Telegram
	fmt.Println("Telegram Configuration")
	fmt.Println(strings.Repeat("─", 35))

	if !tg.Enabled {
		fmt.Println("  Status:  \033[31mdisabled\033[0m")
	} else {
		fmt.Println("  Status:  \033[32menabled\033[0m")
	}

	if tg.Token != "" {
		masked := tg.Token[:8] + "••••••••"
		fmt.Printf("  Token:   %s\n", masked)
	} else {
		fmt.Println("  Token:   (not set)")
	}

	if tg.ChatID != "" {
		fmt.Printf("  Chat ID: %s\n", tg.ChatID)
	} else {
		fmt.Println("  Chat ID: (not set)")
	}

	fmt.Println()

	if tg.Token == "" {
		fmt.Println("  Setup: clawtrade telegram setup")
	} else {
		fmt.Println("  Test:  clawtrade telegram test")
	}

	return nil
}

// ─── notify ──────────────────────────────────────────────────────────

func handleNotify(args []string) error {
	if len(args) == 0 {
		return notifyShow()
	}

	switch args[0] {
	case "show":
		return notifyShow()
	case "set":
		if len(args) < 3 {
			fmt.Println("Usage: clawtrade notify set <key> <value>")
			fmt.Println()
			fmt.Println("Keys:")
			fmt.Println("  telegram.enabled       Enable Telegram (true|false)")
			fmt.Println("  telegram.token         Bot token")
			fmt.Println("  telegram.chat_id       Chat ID")
			fmt.Println("  discord.enabled        Enable Discord (true|false)")
			fmt.Println("  discord.webhook_url    Webhook URL")
			fmt.Println("  alerts.trade_executed   Alert on trades (true|false)")
			fmt.Println("  alerts.risk_alert       Alert on risk events (true|false)")
			fmt.Println("  alerts.pnl_update       Alert on P&L changes (true|false)")
			fmt.Println("  alerts.system_alert     Alert on system events (true|false)")
			return nil
		}
		return configSet("notifications."+args[1], args[2])
	default:
		return notifyShow()
	}
}

func notifyShow() error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	n := cfg.Notifications

	fmt.Println("Notification Settings")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()

	// Telegram
	fmt.Println("  Telegram:")
	if n.Telegram.Enabled {
		fmt.Println("    Status:   \033[32menabled\033[0m")
		if n.Telegram.Token != "" {
			fmt.Printf("    Token:    %s••••\n", n.Telegram.Token[:8])
		}
		fmt.Printf("    Chat ID:  %s\n", n.Telegram.ChatID)
	} else {
		fmt.Println("    Status:   \033[31mdisabled\033[0m")
		fmt.Println("    Setup:    clawtrade telegram setup")
	}
	fmt.Println()

	// Discord
	fmt.Println("  Discord:")
	if n.Discord.Enabled {
		fmt.Println("    Status:   \033[32menabled\033[0m")
		if n.Discord.WebhookURL != "" {
			url := n.Discord.WebhookURL
			if len(url) > 30 {
				url = url[:30] + "..."
			}
			fmt.Printf("    Webhook:  %s\n", url)
		}
	} else {
		fmt.Println("    Status:   \033[31mdisabled\033[0m")
	}
	fmt.Println()

	// Alerts
	fmt.Println("  Alert Types:")
	printAlertStatus("    Trade Executions:", n.Alerts.TradeExecuted)
	printAlertStatus("    Risk Alerts:", n.Alerts.RiskAlert)
	printAlertStatus("    P&L Updates:", n.Alerts.PnlUpdate)
	printAlertStatus("    System Alerts:", n.Alerts.SystemAlert)

	return nil
}

func printAlertStatus(label string, enabled bool) {
	if enabled {
		fmt.Printf("  %-22s \033[32mon\033[0m\n", label)
	} else {
		fmt.Printf("  %-22s \033[31moff\033[0m\n", label)
	}
}

// ─── status ──────────────────────────────────────────────────────────

func handleStatus() error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	fmt.Printf("Clawtrade %s\n", internal.Version)
	fmt.Println(strings.Repeat("─", 40))

	fmt.Printf("  Config:    %s\n", configPath)
	fmt.Printf("  Database:  %s", cfg.Database.Path)
	if _, err := os.Stat(cfg.Database.Path); err == nil {
		fmt.Print(" ✓")
	} else {
		fmt.Print(" ✗ (not found)")
	}
	fmt.Println()

	fmt.Printf("  Vault:     %s", cfg.Vault.Path)
	if _, err := os.Stat(cfg.Vault.Path); err == nil {
		fmt.Print(" ✓")
	} else {
		fmt.Print(" (not initialized)")
	}
	fmt.Println()

	fmt.Printf("  Server:    %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("  Mode:      %s\n", cfg.Risk.DefaultMode)

	fmt.Printf("  Exchanges: %d configured\n", len(cfg.Exchanges))
	for _, ex := range cfg.Exchanges {
		status := "off"
		if ex.Enabled {
			status = "on"
		}
		fmt.Printf("             • %-12s [%s]\n", ex.Name, status)
	}

	agentStatus := "disabled"
	if cfg.Agent.Enabled {
		agentStatus = "enabled"
	}
	fmt.Printf("  Agent:     %s\n", agentStatus)
	fmt.Printf("  Watchlist: %s\n", strings.Join(cfg.Agent.Watchlist, ", "))

	// Notifications
	channels := []string{}
	if cfg.Notifications.Telegram.Enabled {
		channels = append(channels, "Telegram")
	}
	if cfg.Notifications.Discord.Enabled {
		channels = append(channels, "Discord")
	}
	if len(channels) > 0 {
		fmt.Printf("  Notify:    %s\n", strings.Join(channels, ", "))
	} else {
		fmt.Println("  Notify:    (none)")
	}

	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────

func getExchangeType(name string) string {
	switch name {
	case "binance", "bybit", "okx":
		return "CEX"
	case "mt5":
		return "Forex"
	case "ibkr":
		return "Broker"
	case "hyperliquid", "uniswap":
		return "DEX"
	default:
		return "Unknown"
	}
}

func storeInVault(vaultPath, namespace string, fields map[string]string) error {
	secretFields := map[string]bool{
		"api_secret":  true,
		"private_key": true,
		"password":    true,
		"passphrase":  true,
		"token":       true,
	}

	var vault *security.Vault
	var err error

	if _, statErr := os.Stat(vaultPath); statErr == nil {
		vault, err = security.OpenVault(vaultPath, "clawtrade")
		if err != nil {
			return err
		}
	} else {
		vault, err = security.NewVault(vaultPath, "clawtrade")
		if err != nil {
			return err
		}
	}

	for k, v := range fields {
		if secretFields[k] {
			if err := vault.Set(namespace, k, v); err != nil {
				return err
			}
		}
	}

	return vault.Save()
}

func httpPost(url, jsonPayload string) (bool, error) {
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(jsonPayload))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200, nil
}
