// Basic Chat Agent for trading conversations

import type { LLMMessage, LLMResponse, LLMAdapter } from "../llm/adapter";

export interface MemoryContext {
  episodes?: Array<{ summary: string; action: string; result: string }>;
  rules?: Array<{ condition: string; action: string }>;
  profile?: { riskTolerance: string; tradingStyle: string };
}

export interface ChatConfig {
  systemPrompt?: string;
  maxHistoryLength?: number;
}

const DEFAULT_SYSTEM_PROMPT = `You are Clawtrade, an AI trading assistant. You help users with:
- Market analysis and trading decisions
- Portfolio management and risk assessment
- Technical and fundamental analysis
- Trading strategy development

You are knowledgeable but cautious. Always remind users that trading involves risk.
Never provide financial advice - only analysis and information.

When given memory context, use it to personalize your responses based on the user's trading history, preferences, and past experiences.`;

export class ChatAgent {
  private history: LLMMessage[] = [];
  private llm: LLMAdapter;
  private systemPrompt: string;
  private maxHistoryLength: number;

  constructor(llm: LLMAdapter, config?: ChatConfig) {
    this.llm = llm;
    this.systemPrompt = config?.systemPrompt || DEFAULT_SYSTEM_PROMPT;
    this.maxHistoryLength = config?.maxHistoryLength || 20;
  }

  async chat(userMessage: string, memoryContext?: MemoryContext): Promise<string> {
    // Build messages array
    const messages: LLMMessage[] = [];

    // System prompt with memory context
    let system = this.systemPrompt;
    if (memoryContext) {
      system += "\n\n## Your Memory Context\n";
      if (memoryContext.profile) {
        system += `\nUser Profile: Risk tolerance: ${memoryContext.profile.riskTolerance}, Trading style: ${memoryContext.profile.tradingStyle}`;
      }
      if (memoryContext.rules && memoryContext.rules.length > 0) {
        system += "\n\nTrading Rules:";
        for (const rule of memoryContext.rules) {
          system += `\n- When ${rule.condition}, then ${rule.action}`;
        }
      }
      if (memoryContext.episodes && memoryContext.episodes.length > 0) {
        system += "\n\nRecent Trading Episodes:";
        for (const ep of memoryContext.episodes) {
          system += `\n- ${ep.summary}: ${ep.action} → ${ep.result}`;
        }
      }
    }

    messages.push({ role: "system", content: system });

    // Add conversation history
    messages.push(...this.history);

    // Add current message
    messages.push({ role: "user", content: userMessage });

    // Call LLM
    const response = await this.llm.chat(messages);

    // Update history
    this.history.push({ role: "user", content: userMessage });
    this.history.push({ role: "assistant", content: response.content });

    // Trim history if too long
    if (this.history.length > this.maxHistoryLength * 2) {
      this.history = this.history.slice(-this.maxHistoryLength * 2);
    }

    return response.content;
  }

  getHistory(): LLMMessage[] {
    return [...this.history];
  }

  clearHistory(): void {
    this.history = [];
  }
}
