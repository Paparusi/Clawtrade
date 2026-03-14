#!/usr/bin/env bun
// Clawtrade CLI - Entry Point

import { startRepl } from "./repl";

const API_BASE = process.env.CLAWTRADE_API || "http://127.0.0.1:9090";

async function main() {
  const args = process.argv.slice(2);
  const command = args[0];

  switch (command) {
    case "chat":
      await startRepl(API_BASE);
      break;

    case "version":
      await showVersion(API_BASE);
      break;

    case "health":
      await checkHealth(API_BASE);
      break;

    case "config":
      await handleConfig(API_BASE, args.slice(1));
      break;

    case "exchange":
      await handleExchange(API_BASE, args.slice(1));
      break;

    case "risk":
      await handleRisk(API_BASE, args.slice(1));
      break;

    case "agent":
      await handleAgent(API_BASE, args.slice(1));
      break;

    case "status":
      await handleStatus(API_BASE);
      break;

    default:
      printUsage();
      break;
  }
}

async function showVersion(apiBase: string) {
  try {
    const resp = await fetch(`${apiBase}/api/v1/system/version`);
    const data = (await resp.json()) as { version: string };
    console.log(`Clawtrade ${data.version}`);
  } catch {
    console.error("Error: Cannot connect to Clawtrade server at", apiBase);
    process.exit(1);
  }
}

async function checkHealth(apiBase: string) {
  try {
    const resp = await fetch(`${apiBase}/api/v1/system/health`);
    const data = (await resp.json()) as { status: string; version: string };
    console.log(`Status: ${data.status}`);
    console.log(`Version: ${data.version}`);
  } catch {
    console.error("Error: Cannot connect to Clawtrade server at", apiBase);
    process.exit(1);
  }
}

// ─── config ──────────────────────────────────────────────────────────

async function handleConfig(apiBase: string, args: string[]) {
  const sub = args[0] || "show";

  switch (sub) {
    case "show":
      await apiGet(apiBase, "/api/v1/config", (data) => {
        console.log("Clawtrade Configuration");
        console.log("─".repeat(40));
        printObj(data, 0);
      });
      break;

    case "set":
      if (args.length < 3) {
        console.log("Usage: clawtrade-cli config set <key> <value>");
        console.log("");
        console.log("Examples:");
        console.log("  clawtrade-cli config set server.port 8080");
        console.log("  clawtrade-cli config set risk.default_mode paper");
        console.log("  clawtrade-cli config set agent.enabled true");
        return;
      }
      await apiPost(apiBase, "/api/v1/config", { key: args[1], value: args[2] });
      console.log(`✓ Set ${args[1]} = ${args[2]}`);
      break;

    default:
      console.log("Usage: clawtrade-cli config <show|set>");
  }
}

// ─── exchange ────────────────────────────────────────────────────────

async function handleExchange(apiBase: string, args: string[]) {
  const sub = args[0] || "list";

  switch (sub) {
    case "list":
      await apiGet(apiBase, "/api/v1/exchanges", (data: any) => {
        const exchanges = data.exchanges || [];
        if (exchanges.length === 0) {
          console.log("No exchanges configured.");
          console.log("Add one via web dashboard: Settings → Exchanges");
          return;
        }
        console.log("Configured Exchanges:");
        console.log("");
        for (const ex of exchanges) {
          const status = ex.enabled ? "●" : "○";
          console.log(`  ${status} ${ex.name.padEnd(14)} [${ex.type}]`);
        }
      });
      break;

    case "add":
      if (args.length < 2) {
        console.log("Usage: clawtrade-cli exchange add <name>");
        console.log("");
        console.log("Supported: binance, bybit, okx, mt5, ibkr, hyperliquid, uniswap");
        console.log("");
        console.log("Tip: For interactive setup, use the Go CLI:");
        console.log("  clawtrade exchange add <name>");
        console.log("");
        console.log("Or configure via web dashboard: Settings → Exchanges");
        return;
      }
      console.log(`To add ${args[1]}, use the Go CLI for interactive setup:`);
      console.log(`  clawtrade exchange add ${args[1]}`);
      console.log("");
      console.log("Or configure via web dashboard: Settings → Exchanges");
      break;

    default:
      console.log("Usage: clawtrade-cli exchange <list|add>");
  }
}

// ─── risk ────────────────────────────────────────────────────────────

async function handleRisk(apiBase: string, args: string[]) {
  const sub = args[0] || "show";

  switch (sub) {
    case "show":
      await apiGet(apiBase, "/api/v1/config", (data: any) => {
        const r = data.risk || {};
        console.log("Risk Management");
        console.log("─".repeat(35));
        console.log(`  Trading Mode:        ${r.default_mode || "paper"}`);
        console.log(`  Max Risk/Trade:      ${((r.max_risk_per_trade || 0.02) * 100).toFixed(1)}%`);
        console.log(`  Max Daily Loss:      ${((r.max_daily_loss || 0.05) * 100).toFixed(1)}%`);
        console.log(`  Max Positions:       ${r.max_positions || 5}`);
        console.log(`  Max Leverage:        ${r.max_leverage || 10}x`);
      });
      break;

    case "set":
      if (args.length < 3) {
        console.log("Usage: clawtrade-cli risk set <key> <value>");
        return;
      }
      await apiPost(apiBase, "/api/v1/config", { key: `risk.${args[1]}`, value: args[2] });
      console.log(`✓ Set risk.${args[1]} = ${args[2]}`);
      break;

    default:
      console.log("Usage: clawtrade-cli risk <show|set>");
  }
}

// ─── agent ───────────────────────────────────────────────────────────

async function handleAgent(apiBase: string, args: string[]) {
  const sub = args[0] || "show";

  switch (sub) {
    case "show":
      await apiGet(apiBase, "/api/v1/config", (data: any) => {
        const a = data.agent || {};
        console.log("AI Agent Configuration");
        console.log("─".repeat(35));
        console.log(`  Status:              ${a.enabled ? "enabled" : "disabled"}`);
        console.log(`  Auto Trade:          ${a.auto_trade || false}`);
        console.log(`  Order Confirmation:  ${a.confirmation ?? true}`);
        console.log(`  Min Confidence:      ${((a.min_confidence || 0.7) * 100).toFixed(0)}%`);
        console.log(`  Scan Interval:       ${a.scan_interval || 30}s`);

        if (a.watchlist?.length) {
          console.log("");
          console.log("  Watchlist:");
          for (const sym of a.watchlist) {
            console.log(`    • ${sym}`);
          }
        }
      });
      break;

    case "set":
      if (args.length < 3) {
        console.log("Usage: clawtrade-cli agent set <key> <value>");
        return;
      }
      await apiPost(apiBase, "/api/v1/config", { key: `agent.${args[1]}`, value: args[2] });
      console.log(`✓ Set agent.${args[1]} = ${args[2]}`);
      break;

    default:
      console.log("Usage: clawtrade-cli agent <show|set>");
  }
}

// ─── status ──────────────────────────────────────────────────────────

async function handleStatus(apiBase: string) {
  try {
    const healthResp = await fetch(`${apiBase}/api/v1/system/health`);
    const health = (await healthResp.json()) as any;

    console.log(`Clawtrade ${health.version || "?"}`);
    console.log("─".repeat(40));
    console.log(`  Server:    ${apiBase}`);
    console.log(`  Status:    ${health.status || "unknown"}`);

    // Try to get config
    try {
      const configResp = await fetch(`${apiBase}/api/v1/config`);
      const cfg = (await configResp.json()) as any;

      console.log(`  Mode:      ${cfg.risk?.default_mode || "paper"}`);
      console.log(`  Exchanges: ${cfg.exchanges?.length || 0} configured`);
      console.log(`  Agent:     ${cfg.agent?.enabled ? "enabled" : "disabled"}`);
    } catch {
      // Config endpoint might not exist yet
    }
  } catch {
    console.error("Error: Cannot connect to Clawtrade server at", apiBase);
    console.log("");
    console.log("Is the server running? Start it with:");
    console.log("  clawtrade serve");
    process.exit(1);
  }
}

// ─── helpers ─────────────────────────────────────────────────────────

async function apiGet(apiBase: string, path: string, handler: (data: any) => void) {
  try {
    const resp = await fetch(`${apiBase}${path}`);
    if (!resp.ok) {
      console.error(`Error: ${resp.status} ${resp.statusText}`);
      process.exit(1);
    }
    const data = await resp.json();
    handler(data);
  } catch {
    console.error("Error: Cannot connect to Clawtrade server at", apiBase);
    console.log("Start the server: clawtrade serve");
    process.exit(1);
  }
}

async function apiPost(apiBase: string, path: string, body: any) {
  try {
    const resp = await fetch(`${apiBase}${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!resp.ok) {
      const err = await resp.text();
      console.error(`Error: ${err}`);
      process.exit(1);
    }
  } catch {
    console.error("Error: Cannot connect to Clawtrade server at", apiBase);
    console.log("Start the server: clawtrade serve");
    process.exit(1);
  }
}

function printObj(obj: any, indent: number) {
  for (const [key, value] of Object.entries(obj)) {
    const pad = " ".repeat(indent * 2 + 2);
    if (typeof value === "object" && value !== null && !Array.isArray(value)) {
      console.log(`${pad}${key}:`);
      printObj(value, indent + 1);
    } else if (Array.isArray(value)) {
      console.log(`${pad}${key}: ${value.join(", ")}`);
    } else {
      console.log(`${pad}${key}: ${value}`);
    }
  }
}

function printUsage() {
  console.log("Clawtrade CLI");
  console.log("");
  console.log("Usage: clawtrade-cli <command>");
  console.log("");
  console.log("Commands:");
  console.log("  chat               Start interactive chat with AI trading agent");
  console.log("  version            Show server version");
  console.log("  health             Check server health");
  console.log("");
  console.log("  config show        Show current configuration");
  console.log("  config set K V     Set a config value");
  console.log("");
  console.log("  exchange list      List configured exchanges");
  console.log("  exchange add NAME  Add exchange (redirects to Go CLI)");
  console.log("");
  console.log("  risk show          Show risk parameters");
  console.log("  risk set K V       Set a risk parameter");
  console.log("");
  console.log("  agent show         Show agent configuration");
  console.log("  agent set K V      Set an agent parameter");
  console.log("");
  console.log("  status             Show system status");
  console.log("");
  console.log("Environment:");
  console.log("  CLAWTRADE_API      Server URL (default: http://127.0.0.1:9090)");
}

main();
