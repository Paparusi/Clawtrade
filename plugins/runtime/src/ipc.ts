// JSON-RPC 2.0 protocol handler for communication with Go core

export interface RPCRequest {
  jsonrpc: "2.0";
  id?: number;
  method: string;
  params?: unknown;
}

export interface RPCResponse {
  jsonrpc: "2.0";
  id?: number;
  result?: unknown;
  error?: RPCError;
}

export interface RPCError {
  code: number;
  message: string;
}

export type MethodHandler = (params: unknown) => Promise<unknown>;

export class IPCClient {
  private methods: Map<string, MethodHandler> = new Map();
  private pendingRequests: Map<number, {
    resolve: (value: RPCResponse) => void;
    reject: (reason: Error) => void;
  }> = new Map();
  private nextId = 1;
  private buffer = "";

  constructor() {
    this.setupStdinReader();
  }

  private setupStdinReader(): void {
    const decoder = new TextDecoder();
    const stdin = Bun.stdin.stream();
    const reader = stdin.getReader();

    const readLoop = async () => {
      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          this.buffer += decoder.decode(value, { stream: true });
          const lines = this.buffer.split("\n");
          this.buffer = lines.pop() || "";

          for (const line of lines) {
            if (line.trim()) {
              await this.handleMessage(line.trim());
            }
          }
        }
      } catch (err) {
        console.error("[runtime] stdin read error:", err);
      }
    };

    readLoop();
  }

  private async handleMessage(data: string): Promise<void> {
    try {
      const msg = JSON.parse(data);

      // Check if it's a response to our request
      if ("result" in msg || "error" in msg) {
        const pending = this.pendingRequests.get(msg.id);
        if (pending) {
          this.pendingRequests.delete(msg.id);
          pending.resolve(msg as RPCResponse);
        }
        return;
      }

      // It's an incoming request from Go core
      const req = msg as RPCRequest;
      const handler = this.methods.get(req.method);

      let response: RPCResponse;
      if (!handler) {
        response = {
          jsonrpc: "2.0",
          id: req.id,
          error: { code: -32601, message: `Method not found: ${req.method}` },
        };
      } else {
        try {
          const result = await handler(req.params);
          response = { jsonrpc: "2.0", id: req.id, result };
        } catch (err) {
          response = {
            jsonrpc: "2.0",
            id: req.id,
            error: { code: -32000, message: String(err) },
          };
        }
      }

      this.send(response);
    } catch {
      this.send({
        jsonrpc: "2.0",
        error: { code: -32700, message: "Parse error" },
      });
    }
  }

  registerMethod(name: string, handler: MethodHandler): void {
    this.methods.set(name, handler);
  }

  async call(method: string, params?: unknown): Promise<RPCResponse> {
    const id = this.nextId++;
    const request: RPCRequest = {
      jsonrpc: "2.0",
      id,
      method,
      params,
    };

    return new Promise((resolve, reject) => {
      this.pendingRequests.set(id, { resolve, reject });
      this.send(request);

      // Timeout after 30 seconds
      setTimeout(() => {
        if (this.pendingRequests.has(id)) {
          this.pendingRequests.delete(id);
          reject(new Error(`RPC timeout: ${method}`));
        }
      }, 30000);
    });
  }

  private send(msg: RPCRequest | RPCResponse): void {
    const data = JSON.stringify(msg) + "\n";
    Bun.write(Bun.stdout, data);
  }
}
