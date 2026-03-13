// MCP Client - connects to external MCP servers and exposes their tools/resources

export interface MCPServerConfig {
  name: string;
  url: string; // stdio: command path, sse: http URL
  transport: "stdio" | "sse";
  command?: string; // for stdio transport
  args?: string[]; // for stdio transport
  env?: Record<string, string>;
  timeout?: number; // connection timeout ms
}

export interface MCPTool {
  name: string;
  description: string;
  inputSchema: Record<string, unknown>; // JSON Schema
}

export interface MCPResource {
  uri: string;
  name: string;
  description?: string;
  mimeType?: string;
}

export interface MCPCallResult {
  content: Array<{
    type: string;
    text?: string;
    data?: string;
    mimeType?: string;
  }>;
  isError?: boolean;
}

// Individual server connection
class MCPServerConnection {
  private config: MCPServerConfig;
  private tools: MCPTool[] = [];
  private resources: MCPResource[] = [];
  private connected: boolean = false;

  constructor(config: MCPServerConfig) {
    this.config = config;
  }

  async initialize(): Promise<void> {
    // For now, mark as connected (real transport implementation comes later)
    // In production, this would start stdio process or connect via SSE
    this.connected = true;
  }

  async close(): Promise<void> {
    this.connected = false;
  }

  isConnected(): boolean {
    return this.connected;
  }

  // Register tools that this server provides (for testing/mock purposes)
  registerTools(tools: MCPTool[]): void {
    this.tools = tools;
  }

  registerResources(resources: MCPResource[]): void {
    this.resources = resources;
  }

  getTools(): MCPTool[] {
    return this.tools;
  }

  getResources(): MCPResource[] {
    return this.resources;
  }

  async callTool(
    name: string,
    args: Record<string, unknown>
  ): Promise<MCPCallResult> {
    if (!this.connected) throw new Error("Not connected");
    // Placeholder: return mock result
    // Real implementation would send JSON-RPC to the server process
    return {
      content: [
        {
          type: "text",
          text: `Tool ${name} called with ${JSON.stringify(args)}`,
        },
      ],
    };
  }

  async readResource(uri: string): Promise<MCPCallResult> {
    if (!this.connected) throw new Error("Not connected");
    return { content: [{ type: "text", text: `Resource ${uri}` }] };
  }
}

// MCPClient manages connections to external MCP servers
export class MCPClient {
  private servers: Map<string, MCPServerConnection> = new Map();

  // Connect to an MCP server
  async connect(config: MCPServerConfig): Promise<void> {
    const conn = new MCPServerConnection(config);
    await conn.initialize();
    this.servers.set(config.name, conn);
  }

  // Disconnect from a server
  async disconnect(name: string): Promise<void> {
    const conn = this.servers.get(name);
    if (conn) {
      await conn.close();
      this.servers.delete(name);
    }
  }

  // List all connected servers
  listServers(): string[] {
    return Array.from(this.servers.keys());
  }

  // List tools from a specific server (or all servers)
  async listTools(serverName?: string): Promise<MCPTool[]> {
    if (serverName) {
      const conn = this.servers.get(serverName);
      if (!conn) throw new Error(`Server "${serverName}" not found`);
      return conn.getTools();
    }
    const allTools: MCPTool[] = [];
    for (const conn of this.servers.values()) {
      allTools.push(...conn.getTools());
    }
    return allTools;
  }

  // List resources from a specific server (or all servers)
  async listResources(serverName?: string): Promise<MCPResource[]> {
    if (serverName) {
      const conn = this.servers.get(serverName);
      if (!conn) throw new Error(`Server "${serverName}" not found`);
      return conn.getResources();
    }
    const allResources: MCPResource[] = [];
    for (const conn of this.servers.values()) {
      allResources.push(...conn.getResources());
    }
    return allResources;
  }

  // Call a tool on a specific server
  async callTool(
    serverName: string,
    toolName: string,
    args: Record<string, unknown>
  ): Promise<MCPCallResult> {
    const conn = this.servers.get(serverName);
    if (!conn) throw new Error(`Server "${serverName}" not found`);
    return conn.callTool(toolName, args);
  }

  // Read a resource
  async readResource(
    serverName: string,
    uri: string
  ): Promise<MCPCallResult> {
    const conn = this.servers.get(serverName);
    if (!conn) throw new Error(`Server "${serverName}" not found`);
    return conn.readResource(uri);
  }

  // Disconnect all
  async disconnectAll(): Promise<void> {
    for (const conn of this.servers.values()) {
      await conn.close();
    }
    this.servers.clear();
  }

  // Get underlying connection for testing/advanced use
  getConnection(name: string): MCPServerConnection | undefined {
    return this.servers.get(name);
  }
}
