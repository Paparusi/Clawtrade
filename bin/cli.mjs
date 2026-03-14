#!/usr/bin/env node

// Clawtrade CLI Launcher
// Routes commands to the Go binary (server, config, exchange, etc.)
// or to the Node.js CLI (chat, interactive REPL)

import { spawn, execFileSync } from "node:child_process";
import { existsSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { platform, arch } from "node:os";

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = join(__dirname, "..");

// ─── Find Go binary ─────────────────────────────────────────────────

function getBinaryName() {
  const os = platform();
  const cpu = arch();
  const ext = os === "win32" ? ".exe" : "";
  const osName = { darwin: "darwin", linux: "linux", win32: "windows" }[os] || os;
  const cpuName = { x64: "amd64", arm64: "arm64" }[cpu] || cpu;
  return { full: `clawtrade-${osName}-${cpuName}${ext}`, ext };
}

function findBinary() {
  const { full, ext } = getBinaryName();

  // Check multiple locations
  const candidates = [
    join(root, "bin", full),
    join(root, "bin", `clawtrade${ext}`),
    join(root, `clawtrade${ext}`),
  ];

  for (const path of candidates) {
    if (existsSync(path)) return path;
  }

  return null;
}

// ─── CLI Router ──────────────────────────────────────────────────────

const args = process.argv.slice(2);
const command = args[0];

// Commands that need the Go binary
const goCommands = new Set([
  "serve", "init", "version", "config", "exchange",
  "risk", "agent", "telegram", "notify", "status",
]);

// No arguments → show help
if (!command) {
  printHelp();
  process.exit(0);
}

if (command === "help" || command === "--help" || command === "-h") {
  printHelp();
  process.exit(0);
}

if (command === "--version" || command === "-v") {
  args[0] = "version";
}

if (goCommands.has(args[0])) {
  runGoBinary(args);
} else if (command === "chat") {
  runChat(args.slice(1));
} else if (command === "dashboard" || command === "web") {
  runDashboard();
} else if (command === "install-server") {
  installServer();
} else {
  // Try Go binary for unknown commands (might be valid)
  runGoBinary(args);
}

// ─── Go Binary Runner ────────────────────────────────────────────────

function runGoBinary(args) {
  const binary = findBinary();

  if (!binary) {
    console.error("Clawtrade server binary not found.");
    console.error("");
    console.error("Install it with:");
    console.error("  clawtrade install-server");
    console.error("");
    console.error("Or build from source:");
    console.error("  go build -o bin/clawtrade ./cmd/clawtrade");
    process.exit(1);
  }

  const child = spawn(binary, args, {
    stdio: "inherit",
    cwd: root,
  });

  child.on("error", (err) => {
    console.error(`Failed to start clawtrade: ${err.message}`);
    process.exit(1);
  });

  child.on("exit", (code) => {
    process.exit(code ?? 0);
  });
}

// ─── Chat (Node.js REPL) ────────────────────────────────────────────

async function runChat(chatArgs) {
  const apiBase = process.env.CLAWTRADE_API || "http://127.0.0.1:8899";

  // Check if server is running
  try {
    const resp = await fetch(`${apiBase}/api/v1/system/health`, {
      signal: AbortSignal.timeout(2000),
    });
    if (!resp.ok) throw new Error("not ok");
  } catch {
    console.error("Cannot connect to Clawtrade server at", apiBase);
    console.error("");
    console.error("Start the server first:");
    console.error("  clawtrade serve");
    process.exit(1);
  }

  // Dynamic import the chat REPL
  const replPath = join(root, "cli", "dist", "repl.mjs");
  if (existsSync(replPath)) {
    const { startRepl } = await import(replPath);
    await startRepl(apiBase);
  } else {
    // Fallback: basic readline REPL
    const readline = await import("node:readline");
    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
      prompt: "\x1b[35mclawtrade>\x1b[0m ",
    });

    console.log("Clawtrade AI Trading Agent");
    console.log('Type your message, or /quit to exit.\n');
    rl.prompt();

    rl.on("line", async (line) => {
      const input = line.trim();
      if (!input) { rl.prompt(); return; }
      if (input === "/quit" || input === "/exit") { rl.close(); return; }

      if (input === "/portfolio") {
        try {
          const resp = await fetch(`${apiBase}/api/v1/portfolio`);
          const data = await resp.json();
          console.log(JSON.stringify(data, null, 2));
        } catch {
          console.log("Could not fetch portfolio.");
        }
        rl.prompt();
        return;
      }

      try {
        const resp = await fetch(`${apiBase}/api/v1/chat`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ message: input }),
        });
        const data = await resp.json();
        console.log(`\n\x1b[36m${data.response || JSON.stringify(data)}\x1b[0m\n`);
      } catch {
        console.log("Error communicating with server.");
      }
      rl.prompt();
    });

    rl.on("close", () => {
      console.log("\nGoodbye!");
      process.exit(0);
    });
  }
}

// ─── Dashboard ───────────────────────────────────────────────────────

function runDashboard() {
  const webDir = join(root, "web");
  if (!existsSync(join(webDir, "package.json"))) {
    console.error("Web dashboard not found at", webDir);
    process.exit(1);
  }

  console.log("Starting Clawtrade Dashboard...");
  const child = spawn("npx", ["vite"], {
    stdio: "inherit",
    cwd: webDir,
    shell: true,
  });

  child.on("exit", (code) => process.exit(code ?? 0));
}

// ─── Install Server ──────────────────────────────────────────────────

function installServer() {
  console.log("Installing Clawtrade server binary...");
  console.log("");

  // Check if Go is available
  try {
    execFileSync("go", ["version"], { stdio: "pipe" });
  } catch {
    console.error("Go is not installed. Install Go from https://go.dev/dl/");
    console.error("");
    console.error("Or download pre-built binary from:");
    console.error("  https://github.com/clawtrade/clawtrade/releases");
    process.exit(1);
  }

  const { ext } = getBinaryName();
  const outPath = join(root, "bin", `clawtrade${ext}`);

  console.log("Building from source...");
  try {
    execFileSync("go", ["build", "-o", outPath, "./cmd/clawtrade"], {
      stdio: "inherit",
      cwd: root,
    });
    console.log("");
    console.log(`Server binary installed at: ${outPath}`);
    console.log("");
    console.log("Get started:");
    console.log("  clawtrade init     # Setup wizard");
    console.log("  clawtrade serve    # Start server");
  } catch (err) {
    console.error("Build failed:", err.message);
    process.exit(1);
  }
}

// ─── Help ────────────────────────────────────────────────────────────

function printHelp() {
  console.log(`
  Clawtrade - AI Trading Agent Platform

  Usage: clawtrade <command> [options]

  Getting Started:
    init                   Interactive setup wizard
    serve                  Start the trading server
    chat                   Chat with AI trading agent
    dashboard              Start web dashboard (dev mode)
    status                 Show system status

  Configuration:
    config show            Show all configuration
    config set K V         Set a config value
    exchange add NAME      Add exchange (binance/bybit/okx/mt5/ibkr/hyperliquid/uniswap)
    exchange list          List configured exchanges
    risk show              Show risk parameters
    agent show             Show agent configuration

  Notifications:
    telegram setup         Setup Telegram bot
    telegram test          Send test message
    notify show            Show notification settings

  Utilities:
    install-server         Build server from Go source
    version                Show version
    help                   Show this help

  Environment:
    CLAWTRADE_API          Server URL (default: http://127.0.0.1:8899)
    CLAWTRADE_CONFIG       Config file path

  Docs: https://github.com/clawtrade/clawtrade
`);
}
