import { describe, test, expect } from "bun:test";
import { VoiceInput, MockSTTProvider } from "./voice";

describe("VoiceInput", () => {
  test("creates with default config", () => {
    const voice = new VoiceInput();
    const config = voice.getConfig();
    expect(config.provider).toBe("whisper-local");
    expect(config.language).toBe("en");
    expect(config.sampleRate).toBe(16000);
    expect(config.silenceThresholdMs).toBe(2000);
    expect(config.maxDurationMs).toBe(30000);
  });

  test("creates with custom config", () => {
    const voice = new VoiceInput({
      provider: "whisper-api",
      language: "de",
      sampleRate: 44100,
      silenceThresholdMs: 3000,
      maxDurationMs: 60000,
    });
    const config = voice.getConfig();
    expect(config.provider).toBe("whisper-api");
    expect(config.language).toBe("de");
    expect(config.sampleRate).toBe(44100);
    expect(config.silenceThresholdMs).toBe(3000);
    expect(config.maxDurationMs).toBe(60000);
  });

  test("setProvider sets the STT provider", () => {
    const voice = new VoiceInput();
    const provider = new MockSTTProvider();
    voice.setProvider(provider);
    // Verify by checking isAvailable resolves true
    expect(voice.isAvailable()).resolves.toBe(true);
  });

  test("isAvailable returns false with no provider", async () => {
    const voice = new VoiceInput();
    expect(await voice.isAvailable()).toBe(false);
  });

  test("isAvailable returns true with mock provider", async () => {
    const voice = new VoiceInput();
    voice.setProvider(new MockSTTProvider());
    expect(await voice.isAvailable()).toBe(true);
  });

  test("startRecording/stopRecording toggles state", () => {
    const voice = new VoiceInput();
    expect(voice.isRecording()).toBe(false);
    voice.startRecording();
    expect(voice.isRecording()).toBe(true);
    voice.stopRecording();
    expect(voice.isRecording()).toBe(false);
  });

  test("isRecording reflects current state", () => {
    const voice = new VoiceInput();
    expect(voice.isRecording()).toBe(false);
    voice.startRecording();
    expect(voice.isRecording()).toBe(true);
  });

  test("processAudio calls provider and returns result", async () => {
    const voice = new VoiceInput();
    voice.setProvider(new MockSTTProvider("hello world"));
    const result = await voice.processAudio(Buffer.from("fake-audio"));
    expect(result.text).toBe("hello world");
    expect(result.confidence).toBe(0.95);
    expect(result.language).toBe("en");
    expect(result.durationMs).toBeGreaterThan(0);
  });

  test("processAudio throws without provider", async () => {
    const voice = new VoiceInput();
    expect(voice.processAudio(Buffer.from("data"))).rejects.toThrow(
      "No STT provider configured"
    );
  });

  test("onResult callback fires after processAudio", async () => {
    const voice = new VoiceInput();
    voice.setProvider(new MockSTTProvider("callback test"));
    let received: string | null = null;
    voice.onResult((result) => {
      received = result.text;
    });
    await voice.processAudio(Buffer.from("audio"));
    expect(received).toBe("callback test");
  });

  test('parseCommand detects "trade" intent for "buy 0.1 BTC"', () => {
    const cmd = VoiceInput.parseCommand("buy 0.1 BTC");
    expect(cmd.intent).toBe("trade");
    expect(cmd.raw).toBe("buy 0.1 BTC");
  });

  test('parseCommand detects "query" intent for "what is the price of ETH"', () => {
    const cmd = VoiceInput.parseCommand("what is the price of ETH");
    expect(cmd.intent).toBe("query");
  });

  test('parseCommand detects "command" intent for "set stop loss"', () => {
    const cmd = VoiceInput.parseCommand("set stop loss");
    expect(cmd.intent).toBe("command");
  });

  test('parseCommand detects "chat" intent for general text', () => {
    const cmd = VoiceInput.parseCommand("hello how are you today");
    expect(cmd.intent).toBe("chat");
  });

  test("parseCommand extracts symbol entity", () => {
    const cmd = VoiceInput.parseCommand("what is the price of ETH");
    expect(cmd.entities.symbol).toBe("ETH");
  });

  test("parseCommand extracts amount entity", () => {
    const cmd = VoiceInput.parseCommand("buy 0.1 BTC");
    expect(cmd.entities.amount).toBe("0.1");
  });

  test("parseCommand extracts side entity", () => {
    const cmd = VoiceInput.parseCommand("sell 2 SOL");
    expect(cmd.entities.side).toBe("sell");
  });
});

describe("MockSTTProvider", () => {
  test("returns configured response", async () => {
    const provider = new MockSTTProvider("custom response");
    const result = await provider.transcribe(Buffer.from("audio-data"));
    expect(result.text).toBe("custom response");
    expect(result.confidence).toBe(0.95);
    expect(result.language).toBe("en");
    expect(result.durationMs).toBe(Buffer.from("audio-data").length * 10);
    expect(provider.name).toBe("mock");
    expect(await provider.isAvailable()).toBe(true);
  });
});

describe("VoiceInput config", () => {
  test("getConfig returns current config", () => {
    const voice = new VoiceInput({ provider: "browser", language: "fr" });
    const config = voice.getConfig();
    expect(config.provider).toBe("browser");
    expect(config.language).toBe("fr");
    // Ensure it returns a copy
    config.provider = "modified";
    expect(voice.getConfig().provider).toBe("browser");
  });
});
