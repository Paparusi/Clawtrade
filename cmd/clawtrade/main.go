package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/clawtrade/clawtrade/internal"
	"github.com/clawtrade/clawtrade/internal/adapter"
	"github.com/clawtrade/clawtrade/internal/api"
	"github.com/clawtrade/clawtrade/internal/config"
	"github.com/clawtrade/clawtrade/internal/database"
	"github.com/clawtrade/clawtrade/internal/engine"
	"github.com/clawtrade/clawtrade/internal/memory"
	"github.com/clawtrade/clawtrade/internal/security"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Printf("Clawtrade %s\n", internal.Version)
	case "serve":
		if err := serve(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "init":
		if err := initSetup(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Clawtrade - AI Trading Agent Platform")
	fmt.Printf("Version: %s\n\n", internal.Version)
	fmt.Println("Usage: clawtrade <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  serve     Start the Clawtrade server")
	fmt.Println("  init      First-time setup wizard")
	fmt.Println("  version   Show version")
}

func serve() error {
	// Load config
	configPath := "config/default.yaml"
	if envPath := os.Getenv("CLAWTRADE_CONFIG"); envPath != "" {
		configPath = envPath
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("Warning: Could not load config from %s, using defaults\n", configPath)
		cfg, _ = config.Load("")
	}

	// Ensure data directory exists
	dataDir := filepath.Dir(cfg.Database.Path)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	// Open database
	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// Initialize components
	bus := engine.NewEventBus()
	memStore := memory.NewStore(db)
	auditLog := security.NewAuditLog(db)
	adapters := make(map[string]adapter.TradingAdapter)

	// Create API server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := api.NewServer(bus, memStore, auditLog, adapters)

	// Graceful shutdown
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

	// Start server
	if err := srv.Start(ctx, addr); err != nil && err.Error() != "http: Server closed" {
		return fmt.Errorf("server error: %w", err)
	}

	fmt.Println("Goodbye!")
	return nil
}

func initSetup() error {
	fmt.Println("Clawtrade Setup Wizard")
	fmt.Println("======================")
	fmt.Println()

	// Ensure data directory
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	// Initialize database
	dbPath := filepath.Join(dataDir, "clawtrade.db")
	db, err := database.Open(dbPath)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	db.Close()

	fmt.Println("Database initialized at", dbPath)

	// Ensure config directory
	if err := os.MkdirAll("config", 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	fmt.Println("Configuration ready")
	fmt.Println()
	fmt.Println("Setup complete! Run 'clawtrade serve' to start.")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Add your LLM API key (Claude or OpenAI)")
	fmt.Println("  2. Add exchange API key (or use paper trading mode)")
	fmt.Println("  3. Start trading with 'clawtrade serve'")

	return nil
}
