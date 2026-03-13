import { describe, test, expect } from "bun:test";
import { MCPClient } from "./client";
import type { MCPServerConfig, MCPTool, MCPResource } from "./client";

function makeConfig(name: string): MCPServerConfig {
  return {
    name,
    url: "http://localhost:8080",
    transport: "sse",
  };
}

describe("MCPClient", () => {
  test("connect() adds server to list", async () => {
    const client = new MCPClient();
    await client.connect(makeConfig("server-a"));
    expect(client.listServers()).toEqual(["server-a"]);
  });

  test("disconnect() removes server", async () => {
    const client = new MCPClient();
    await client.connect(makeConfig("server-a"));
    await client.disconnect("server-a");
    expect(client.listServers()).toEqual([]);
  });

  test("listServers() returns connected server names", async () => {
    const client = new MCPClient();
    await client.connect(makeConfig("alpha"));
    await client.connect(makeConfig("beta"));
    expect(client.listServers()).toEqual(["alpha", "beta"]);
  });

  test("listTools() returns tools from a server", async () => {
    const client = new MCPClient();
    await client.connect(makeConfig("srv"));
    const tools: MCPTool[] = [
      { name: "read_file", description: "Read a file", inputSchema: { type: "object" } },
    ];
    client.getConnection("srv")!.registerTools(tools);
    const result = await client.listTools("srv");
    expect(result).toEqual(tools);
  });

  test("listTools() aggregates from all servers when no name specified", async () => {
    const client = new MCPClient();
    await client.connect(makeConfig("a"));
    await client.connect(makeConfig("b"));
    client.getConnection("a")!.registerTools([
      { name: "tool1", description: "T1", inputSchema: {} },
    ]);
    client.getConnection("b")!.registerTools([
      { name: "tool2", description: "T2", inputSchema: {} },
    ]);
    const result = await client.listTools();
    expect(result).toHaveLength(2);
    expect(result.map((t) => t.name)).toEqual(["tool1", "tool2"]);
  });

  test("callTool() calls the correct server", async () => {
    const client = new MCPClient();
    await client.connect(makeConfig("srv"));
    const result = await client.callTool("srv", "echo", { msg: "hi" });
    expect(result.content[0].type).toBe("text");
    expect(result.content[0].text).toContain("echo");
    expect(result.content[0].text).toContain("hi");
  });

  test("callTool() throws for unknown server", async () => {
    const client = new MCPClient();
    expect(client.callTool("nope", "tool", {})).rejects.toThrow(
      'Server "nope" not found'
    );
  });

  test("readResource() works", async () => {
    const client = new MCPClient();
    await client.connect(makeConfig("srv"));
    const result = await client.readResource("srv", "file:///tmp/data.json");
    expect(result.content[0].type).toBe("text");
    expect(result.content[0].text).toContain("file:///tmp/data.json");
  });

  test("disconnectAll() clears all", async () => {
    const client = new MCPClient();
    await client.connect(makeConfig("a"));
    await client.connect(makeConfig("b"));
    await client.disconnectAll();
    expect(client.listServers()).toEqual([]);
  });

  test("MCPServerConnection lifecycle", async () => {
    const client = new MCPClient();
    await client.connect(makeConfig("srv"));
    const conn = client.getConnection("srv")!;
    expect(conn.isConnected()).toBe(true);
    await conn.close();
    expect(conn.isConnected()).toBe(false);
    expect(() => conn.callTool("x", {})).toThrow("Not connected");
  });
});
