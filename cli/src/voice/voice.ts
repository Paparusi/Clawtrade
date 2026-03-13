// Voice input handler - manages audio recording and speech-to-text
// Designed as an abstraction layer that can work with different STT providers

export interface STTProvider {
  name: string;
  transcribe(audioData: Buffer): Promise<TranscriptionResult>;
  isAvailable(): Promise<boolean>;
}

export interface TranscriptionResult {
  text: string;
  confidence: number; // 0-1
  language?: string;
  durationMs: number;
}

export interface VoiceConfig {
  provider: string; // "whisper-local", "whisper-api", "browser"
  language?: string; // ISO 639-1 code
  sampleRate?: number; // default 16000
  silenceThresholdMs?: number; // auto-stop after silence, default 2000
  maxDurationMs?: number; // max recording duration, default 30000
}

// VoiceCommand represents a parsed voice command
export interface VoiceCommand {
  raw: string; // raw transcription
  intent: string; // detected intent: "trade", "query", "command", "chat"
  entities: Record<string, string>; // extracted entities
  confidence: number;
}

const TRADE_KEYWORDS = ["trade", "buy", "sell", "place order", "long", "short"];
const QUERY_KEYWORDS = ["what is", "how much", "show me", "price of"];
const COMMAND_KEYWORDS = ["set", "configure", "enable", "disable"];

const KNOWN_SYMBOLS = [
  "BTC",
  "ETH",
  "SOL",
  "DOGE",
  "XRP",
  "ADA",
  "DOT",
  "AVAX",
  "MATIC",
  "LINK",
  "UNI",
  "ATOM",
  "LTC",
  "BNB",
  "NEAR",
  "APT",
  "ARB",
  "OP",
  "SUI",
  "SEI",
];

const SIDE_KEYWORDS = ["buy", "sell", "long", "short"];

export class VoiceInput {
  private config: VoiceConfig;
  private provider: STTProvider | null = null;
  private recording: boolean = false;
  private onTranscription:
    | ((result: TranscriptionResult) => void)
    | null = null;

  constructor(config?: Partial<VoiceConfig>) {
    this.config = {
      provider: config?.provider || "whisper-local",
      language: config?.language || "en",
      sampleRate: config?.sampleRate || 16000,
      silenceThresholdMs: config?.silenceThresholdMs || 2000,
      maxDurationMs: config?.maxDurationMs || 30000,
    };
  }

  /** Set the STT provider */
  setProvider(provider: STTProvider): void {
    this.provider = provider;
  }

  /** Check if voice input is available */
  async isAvailable(): Promise<boolean> {
    if (!this.provider) return false;
    return this.provider.isAvailable();
  }

  /** Start recording (placeholder - actual mic capture depends on platform) */
  startRecording(): void {
    this.recording = true;
  }

  /** Stop recording */
  stopRecording(): void {
    this.recording = false;
  }

  /** Check if currently recording */
  isRecording(): boolean {
    return this.recording;
  }

  /** Register transcription callback */
  onResult(callback: (result: TranscriptionResult) => void): void {
    this.onTranscription = callback;
  }

  /** Process audio data through STT provider */
  async processAudio(audioData: Buffer): Promise<TranscriptionResult> {
    if (!this.provider) {
      throw new Error("No STT provider configured");
    }
    const result = await this.provider.transcribe(audioData);
    if (this.onTranscription) {
      this.onTranscription(result);
    }
    return result;
  }

  /** Parse a transcription into a voice command */
  static parseCommand(text: string): VoiceCommand {
    const lower = text.toLowerCase();
    const entities: Record<string, string> = {};

    // Intent detection
    let intent = "chat";
    let confidence = 0.5;

    if (TRADE_KEYWORDS.some((kw) => lower.includes(kw))) {
      intent = "trade";
      confidence = 0.9;
    } else if (QUERY_KEYWORDS.some((kw) => lower.includes(kw))) {
      intent = "query";
      confidence = 0.85;
    } else if (COMMAND_KEYWORDS.some((kw) => lower.includes(kw))) {
      intent = "command";
      confidence = 0.85;
    }

    // Entity extraction: symbol
    const upper = text.toUpperCase();
    for (const sym of KNOWN_SYMBOLS) {
      if (upper.includes(sym)) {
        entities.symbol = sym;
        break;
      }
    }

    // Entity extraction: amount (numbers, including decimals)
    const amountMatch = text.match(/(\d+\.?\d*)/);
    if (amountMatch) {
      entities.amount = amountMatch[1];
    }

    // Entity extraction: side
    for (const side of SIDE_KEYWORDS) {
      if (lower.includes(side)) {
        entities.side = side;
        break;
      }
    }

    return {
      raw: text,
      intent,
      entities,
      confidence,
    };
  }

  /** Get current config */
  getConfig(): VoiceConfig {
    return { ...this.config };
  }
}

/** Mock STT provider for testing */
export class MockSTTProvider implements STTProvider {
  name = "mock";
  private response: string;

  constructor(response: string = "test transcription") {
    this.response = response;
  }

  async transcribe(audioData: Buffer): Promise<TranscriptionResult> {
    return {
      text: this.response,
      confidence: 0.95,
      language: "en",
      durationMs: audioData.length * 10,
    };
  }

  async isAvailable(): Promise<boolean> {
    return true;
  }
}
