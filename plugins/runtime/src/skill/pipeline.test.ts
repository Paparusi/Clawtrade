import { describe, test, expect } from 'bun:test';
import { PipelineExecutor } from './pipeline';
import type { PipelineDefinition } from './pipeline';

// ---------------------------------------------------------------------------
// Mock ToolRegistry
// ---------------------------------------------------------------------------

function createMockRegistry(
  handlers: Record<string, (params: Record<string, unknown>) => Promise<unknown>>,
) {
  return {
    callTool: async (qualifiedName: string, params: Record<string, unknown>) => {
      const handler = handlers[qualifiedName];
      if (!handler) throw new Error(`Tool not found in registry: ${qualifiedName}`);
      return handler(params);
    },
  } as any; // cast to ToolRegistry shape
}

// ---------------------------------------------------------------------------
// resolveParams tests
// ---------------------------------------------------------------------------

describe('PipelineExecutor.resolveParams', () => {
  test('replaces $step1.value with actual value', () => {
    const registry = createMockRegistry({});
    const executor = new PipelineExecutor(registry);

    const stepResults = new Map<string, unknown>();
    stepResults.set('step1', { value: 42 });

    const resolved = executor.resolveParams(
      { input: '$step1.value' },
      stepResults,
    );
    expect(resolved.input).toBe(42);
  });

  test('handles nested references', () => {
    const registry = createMockRegistry({});
    const executor = new PipelineExecutor(registry);

    const stepResults = new Map<string, unknown>();
    stepResults.set('fetch', { data: { price: 100.5 } });

    const resolved = executor.resolveParams(
      { amount: '$fetch.data.price' },
      stepResults,
    );
    expect(resolved.amount).toBe(100.5);
  });

  test('leaves non-reference values unchanged', () => {
    const registry = createMockRegistry({});
    const executor = new PipelineExecutor(registry);

    const stepResults = new Map<string, unknown>();
    const resolved = executor.resolveParams(
      { symbol: 'BTCUSD', count: 10, active: true },
      stepResults,
    );
    expect(resolved.symbol).toBe('BTCUSD');
    expect(resolved.count).toBe(10);
    expect(resolved.active).toBe(true);
  });

  test('resolves references inside nested objects', () => {
    const registry = createMockRegistry({});
    const executor = new PipelineExecutor(registry);

    const stepResults = new Map<string, unknown>();
    stepResults.set('step1', { id: 'abc123' });

    const resolved = executor.resolveParams(
      { config: { userId: '$step1.id', mode: 'live' } },
      stepResults,
    );
    expect(resolved.config).toEqual({ userId: 'abc123', mode: 'live' });
  });
});

// ---------------------------------------------------------------------------
// execute tests
// ---------------------------------------------------------------------------

describe('PipelineExecutor.execute', () => {
  test('runs steps in order', async () => {
    const callOrder: string[] = [];
    const registry = createMockRegistry({
      'data.fetch': async () => { callOrder.push('fetch'); return { price: 100 }; },
      'trade.execute': async () => { callOrder.push('execute'); return { orderId: '123' }; },
    });

    const executor = new PipelineExecutor(registry);
    const pipeline: PipelineDefinition = {
      name: 'test-pipeline',
      steps: [
        { name: 'step1', tool: 'data.fetch', params: { symbol: 'BTC' } },
        { name: 'step2', tool: 'trade.execute', params: { amount: 1 } },
      ],
    };

    const result = await executor.execute(pipeline);
    expect(result.status).toBe('completed');
    expect(callOrder).toEqual(['fetch', 'execute']);
    expect(result.steps).toHaveLength(2);
    expect(result.steps[0].status).toBe('success');
    expect(result.steps[1].status).toBe('success');
  });

  test('passes previous output to next step via $references', async () => {
    let capturedParams: Record<string, unknown> = {};
    const registry = createMockRegistry({
      'data.fetch': async () => ({ price: 42000 }),
      'trade.execute': async (params) => { capturedParams = params; return { done: true }; },
    });

    const executor = new PipelineExecutor(registry);
    const pipeline: PipelineDefinition = {
      name: 'ref-pipeline',
      steps: [
        { name: 'getPrice', tool: 'data.fetch', params: { symbol: 'BTC' } },
        { name: 'placeTrade', tool: 'trade.execute', params: { price: '$getPrice.price' } },
      ],
    };

    const result = await executor.execute(pipeline);
    expect(result.status).toBe('completed');
    expect(capturedParams.price).toBe(42000);
  });

  test('stops on error with onError=stop', async () => {
    const registry = createMockRegistry({
      'bad.tool': async () => { throw new Error('boom'); },
      'good.tool': async () => ({ ok: true }),
    });

    const executor = new PipelineExecutor(registry);
    const pipeline: PipelineDefinition = {
      name: 'stop-pipeline',
      onError: 'stop',
      steps: [
        { name: 'fail', tool: 'bad.tool', params: {} },
        { name: 'never', tool: 'good.tool', params: {} },
      ],
    };

    const result = await executor.execute(pipeline);
    expect(result.status).toBe('failed');
    expect(result.steps).toHaveLength(1);
    expect(result.steps[0].status).toBe('failed');
    expect(result.steps[0].error).toBe('boom');
  });

  test('continues on error with onError=continue', async () => {
    const registry = createMockRegistry({
      'bad.tool': async () => { throw new Error('boom'); },
      'good.tool': async () => ({ ok: true }),
    });

    const executor = new PipelineExecutor(registry);
    const pipeline: PipelineDefinition = {
      name: 'continue-pipeline',
      onError: 'continue',
      steps: [
        { name: 'fail', tool: 'bad.tool', params: {} },
        { name: 'succeed', tool: 'good.tool', params: {} },
      ],
    };

    const result = await executor.execute(pipeline);
    expect(result.status).toBe('partial');
    expect(result.steps).toHaveLength(2);
    expect(result.steps[0].status).toBe('failed');
    expect(result.steps[1].status).toBe('success');
  });

  test('skips step when condition=on_failure and previous succeeded', async () => {
    const registry = createMockRegistry({
      'data.fetch': async () => ({ price: 100 }),
      'alert.send': async () => ({ sent: true }),
    });

    const executor = new PipelineExecutor(registry);
    const pipeline: PipelineDefinition = {
      name: 'condition-pipeline',
      steps: [
        { name: 'fetchData', tool: 'data.fetch', params: {} },
        { name: 'alertOnFail', tool: 'alert.send', params: {}, condition: 'on_failure' },
      ],
    };

    const result = await executor.execute(pipeline);
    expect(result.status).toBe('completed');
    expect(result.steps).toHaveLength(2);
    expect(result.steps[0].status).toBe('success');
    expect(result.steps[1].status).toBe('skipped');
  });

  test('runs step when condition=on_failure and previous failed', async () => {
    const registry = createMockRegistry({
      'bad.tool': async () => { throw new Error('fail'); },
      'alert.send': async () => ({ sent: true }),
    });

    const executor = new PipelineExecutor(registry);
    const pipeline: PipelineDefinition = {
      name: 'condition-pipeline',
      onError: 'continue',
      steps: [
        { name: 'failStep', tool: 'bad.tool', params: {} },
        { name: 'alertOnFail', tool: 'alert.send', params: {}, condition: 'on_failure' },
      ],
    };

    const result = await executor.execute(pipeline);
    expect(result.steps[1].status).toBe('success');
  });

  test('retries on failure', async () => {
    let attempts = 0;
    const registry = createMockRegistry({
      'flaky.tool': async () => {
        attempts++;
        if (attempts < 3) throw new Error('transient');
        return { ok: true };
      },
    });

    const executor = new PipelineExecutor(registry);
    const pipeline: PipelineDefinition = {
      name: 'retry-pipeline',
      steps: [
        { name: 'retry', tool: 'flaky.tool', params: {}, retries: 3 },
      ],
    };

    const result = await executor.execute(pipeline);
    expect(result.status).toBe('completed');
    expect(result.steps[0].status).toBe('success');
    expect(attempts).toBe(3);
  });

  test('records duration for steps', async () => {
    const registry = createMockRegistry({
      'data.fetch': async () => ({ ok: true }),
    });

    const executor = new PipelineExecutor(registry);
    const pipeline: PipelineDefinition = {
      name: 'duration-pipeline',
      steps: [
        { name: 'fetch', tool: 'data.fetch', params: {} },
      ],
    };

    const result = await executor.execute(pipeline);
    expect(result.totalDurationMs).toBeGreaterThanOrEqual(0);
    expect(result.steps[0].durationMs).toBeGreaterThanOrEqual(0);
  });
});

// ---------------------------------------------------------------------------
// validate tests
// ---------------------------------------------------------------------------

describe('PipelineExecutor.validate', () => {
  test('catches missing name', () => {
    const errors = PipelineExecutor.validate({
      name: '',
      steps: [{ name: 'a', tool: 'x.y', params: {} }],
    });
    expect(errors.length).toBeGreaterThan(0);
    expect(errors.some((e) => e.toLowerCase().includes('name'))).toBe(true);
  });

  test('catches empty steps', () => {
    const errors = PipelineExecutor.validate({
      name: 'test',
      steps: [],
    });
    expect(errors.length).toBeGreaterThan(0);
    expect(errors.some((e) => e.toLowerCase().includes('step'))).toBe(true);
  });

  test('catches invalid tool format', () => {
    const errors = PipelineExecutor.validate({
      name: 'test',
      steps: [{ name: 'a', tool: 'no-dot-here', params: {} }],
    });
    expect(errors.length).toBeGreaterThan(0);
    expect(errors.some((e) => e.includes('skillName.toolName'))).toBe(true);
  });

  test('passes valid pipeline', () => {
    const errors = PipelineExecutor.validate({
      name: 'valid-pipeline',
      steps: [
        { name: 'step1', tool: 'skill.tool', params: {} },
        { name: 'step2', tool: 'another.action', params: { key: 'val' } },
      ],
    });
    expect(errors).toEqual([]);
  });
});

// ---------------------------------------------------------------------------
// parse tests
// ---------------------------------------------------------------------------

describe('PipelineExecutor.parse', () => {
  test('parses JSON format', () => {
    const json = JSON.stringify({
      name: 'json-pipeline',
      steps: [
        { name: 'step1', tool: 'data.fetch', params: { symbol: 'BTC' } },
      ],
    });

    const pipeline = PipelineExecutor.parse(json);
    expect(pipeline.name).toBe('json-pipeline');
    expect(pipeline.steps).toHaveLength(1);
    expect(pipeline.steps[0].tool).toBe('data.fetch');
  });

  test('parses simple YAML format', () => {
    const yaml = `name: yaml-pipeline
description: A test pipeline
onError: continue
steps:
  - name: fetchData
    tool: market.getData
    params:
      symbol: BTCUSD
      interval: 60`;

    const pipeline = PipelineExecutor.parse(yaml);
    expect(pipeline.name).toBe('yaml-pipeline');
    expect(pipeline.description).toBe('A test pipeline');
    expect(pipeline.onError).toBe('continue');
    expect(pipeline.steps).toHaveLength(1);
    expect(pipeline.steps[0].name).toBe('fetchData');
    expect(pipeline.steps[0].tool).toBe('market.getData');
    expect(pipeline.steps[0].params.symbol).toBe('BTCUSD');
    expect(pipeline.steps[0].params.interval).toBe(60);
  });
});
