import { describe, test, expect } from "bun:test";
import { ChatAgent, type MemoryContext } from "./chat";
import type { LLMAdapter, LLMMessage, LLMResponse } from "../llm/adapter";

// Mock LLM adapter for testing
class MockLLM implements LLMAdapter {
  lastMessages: LLMMessage[] = [];

  name(): string {
    return "mock";
  }

  async chat(messages: LLMMessage[]): Promise<LLMResponse> {
    this.lastMessages = messages;
    const userMsg = messages.filter(m => m.role === "user").pop();
    return {
      content: `Mock response to: ${userMsg?.content || "unknown"}`,
      model: "mock-model",
      usage: { inputTokens: 10, outputTokens: 20 },
    };
  }
}

describe("ChatAgent", () => {
  test("sends message and gets response", async () => {
    const llm = new MockLLM();
    const agent = new ChatAgent(llm);

    const response = await agent.chat("What is BTC price?");
    expect(response).toContain("Mock response to: What is BTC price?");
  });

  test("maintains conversation history", async () => {
    const llm = new MockLLM();
    const agent = new ChatAgent(llm);

    await agent.chat("Hello");
    await agent.chat("How are you?");

    const history = agent.getHistory();
    expect(history.length).toBe(4); // 2 user + 2 assistant
    expect(history[0].role).toBe("user");
    expect(history[0].content).toBe("Hello");
  });

  test("includes memory context in system prompt", async () => {
    const llm = new MockLLM();
    const agent = new ChatAgent(llm);

    const memory: MemoryContext = {
      profile: { riskTolerance: "aggressive", tradingStyle: "scalping" },
      rules: [{ condition: "RSI > 70", action: "consider selling" }],
      episodes: [{ summary: "BTC pump", action: "bought", result: "profit 5%" }],
    };

    await agent.chat("Analyze BTC", memory);

    const systemMsg = llm.lastMessages.find(m => m.role === "system");
    expect(systemMsg?.content).toContain("aggressive");
    expect(systemMsg?.content).toContain("RSI > 70");
    expect(systemMsg?.content).toContain("BTC pump");
  });

  test("clears history", async () => {
    const llm = new MockLLM();
    const agent = new ChatAgent(llm);

    await agent.chat("Hello");
    expect(agent.getHistory().length).toBe(2);

    agent.clearHistory();
    expect(agent.getHistory().length).toBe(0);
  });

  test("trims history when too long", async () => {
    const llm = new MockLLM();
    const agent = new ChatAgent(llm, { maxHistoryLength: 3 });

    for (let i = 0; i < 10; i++) {
      await agent.chat(`Message ${i}`);
    }

    // Should be trimmed to maxHistoryLength * 2 = 6
    expect(agent.getHistory().length).toBeLessThanOrEqual(6);
  });
});
