import { describe, test, expect } from "bun:test";
import { IPCClient } from "./ipc";

describe("IPCClient", () => {
  test("registerMethod stores handler", () => {
    const client = new IPCClient();
    let called = false;
    client.registerMethod("test.echo", async (params) => {
      called = true;
      return params;
    });
    // Verify method was registered (internal check)
    expect((client as any).methods.has("test.echo")).toBe(true);
  });
});
