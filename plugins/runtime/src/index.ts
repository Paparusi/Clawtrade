// Clawtrade Plugin Runtime - Entry Point
// Communicates with Go core via JSON-RPC over stdin/stdout

import { IPCClient } from "./ipc";

const ipc = new IPCClient();

// Register built-in methods that Go core can call
ipc.registerMethod("runtime.ping", async () => {
  return { status: "ok", timestamp: Date.now() };
});

ipc.registerMethod("runtime.info", async () => {
  return {
    name: "@clawtrade/runtime",
    version: "0.1.0",
    engine: "bun",
    engineVersion: Bun.version,
  };
});

console.error("[runtime] Clawtrade plugin runtime started");

export { ipc };
export type { IPCClient } from "./ipc";
