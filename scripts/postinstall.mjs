#!/usr/bin/env node

// Postinstall script: tries to build the Go binary if Go is available,
// otherwise provides instructions for manual installation.

import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { platform, arch } from "node:os";

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = join(__dirname, "..");
const binDir = join(root, "bin");

function getBinaryName() {
  const os = platform();
  const ext = os === "win32" ? ".exe" : "";
  return `clawtrade${ext}`;
}

function main() {
  const binaryName = getBinaryName();
  const binaryPath = join(binDir, binaryName);

  // Already have a binary? Skip.
  if (existsSync(binaryPath)) {
    console.log("  Clawtrade server binary found.");
    return;
  }

  // Ensure bin directory exists
  if (!existsSync(binDir)) {
    mkdirSync(binDir, { recursive: true });
  }

  // Try to build from source if Go is available
  let hasGo = false;
  try {
    execFileSync("go", ["version"], { stdio: "pipe" });
    hasGo = true;
  } catch {
    // Go not found
  }

  if (hasGo && existsSync(join(root, "go.mod"))) {
    console.log("  Building Clawtrade server from source...");
    try {
      execFileSync("go", ["build", "-o", binaryPath, "./cmd/clawtrade"], {
        stdio: "pipe",
        cwd: root,
        timeout: 120000,
      });
      console.log("  Server binary built successfully.");
      return;
    } catch (err) {
      console.warn("  Warning: Could not build from source.");
    }
  }

  // If we get here, no binary and no Go
  console.log("");
  console.log("  ┌─────────────────────────────────────────────┐");
  console.log("  │  Clawtrade installed (CLI ready)             │");
  console.log("  │                                              │");
  console.log("  │  Server binary not found.                    │");
  console.log("  │  Install Go and run:                         │");
  console.log("  │    clawtrade install-server                  │");
  console.log("  │                                              │");
  console.log("  │  Or download from:                           │");
  console.log("  │    github.com/clawtrade/clawtrade/releases   │");
  console.log("  └─────────────────────────────────────────────┘");
  console.log("");
}

main();
