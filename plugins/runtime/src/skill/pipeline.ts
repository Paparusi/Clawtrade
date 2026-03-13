// Skill Pipeline - chain skills together with pipeline definitions

import type { ToolRegistry } from './sdk';

// ---------------------------------------------------------------------------
// Interfaces
// ---------------------------------------------------------------------------

/** Pipeline step definition (from YAML/JSON) */
export interface PipelineStep {
  name: string;
  tool: string;           // qualified tool name: "skillName.toolName"
  params: Record<string, unknown>;
  condition?: string;     // "always" | "on_success" | "on_failure" | expression
  retries?: number;
  timeout?: number;       // timeout in ms
}

/** Pipeline definition */
export interface PipelineDefinition {
  name: string;
  description?: string;
  steps: PipelineStep[];
  onError?: 'stop' | 'continue' | 'skip';
}

/** Step execution result */
export interface StepResult {
  stepName: string;
  tool: string;
  output: unknown;
  error?: string;
  durationMs: number;
  status: 'success' | 'failed' | 'skipped';
}

/** Pipeline execution result */
export interface PipelineResult {
  pipelineName: string;
  steps: StepResult[];
  status: 'completed' | 'failed' | 'partial';
  totalDurationMs: number;
}

// ---------------------------------------------------------------------------
// PipelineExecutor
// ---------------------------------------------------------------------------

export class PipelineExecutor {
  private registry: ToolRegistry;

  constructor(registry: ToolRegistry) {
    this.registry = registry;
  }

  /** Execute a pipeline definition */
  async execute(pipeline: PipelineDefinition): Promise<PipelineResult> {
    const startTime = Date.now();
    const stepResults: StepResult[] = [];
    const outputMap = new Map<string, unknown>();
    const onError = pipeline.onError ?? 'stop';
    let lastStatus: 'success' | 'failed' = 'success';
    let hasFailed = false;

    for (const step of pipeline.steps) {
      // Evaluate condition
      if (step.condition) {
        const shouldRun = this.evaluateCondition(step.condition, lastStatus);
        if (!shouldRun) {
          stepResults.push({
            stepName: step.name,
            tool: step.tool,
            output: undefined,
            durationMs: 0,
            status: 'skipped',
          });
          continue;
        }
      }

      // Resolve parameter references
      const resolvedParams = this.resolveParams(step.params, outputMap);

      // Execute with retries
      const maxAttempts = (step.retries ?? 0) + 1;
      let stepResult: StepResult | null = null;

      for (let attempt = 0; attempt < maxAttempts; attempt++) {
        const stepStart = Date.now();
        try {
          let output: unknown;
          if (step.timeout !== undefined && step.timeout > 0) {
            output = await Promise.race([
              this.registry.callTool(step.tool, resolvedParams),
              new Promise<never>((_, reject) =>
                setTimeout(() => reject(new Error(`Step '${step.name}' timed out after ${step.timeout}ms`)), step.timeout),
              ),
            ]);
          } else {
            output = await this.registry.callTool(step.tool, resolvedParams);
          }
          stepResult = {
            stepName: step.name,
            tool: step.tool,
            output,
            durationMs: Date.now() - stepStart,
            status: 'success',
          };
          break; // success, no more retries
        } catch (err: unknown) {
          const errorMsg = err instanceof Error ? err.message : String(err);
          stepResult = {
            stepName: step.name,
            tool: step.tool,
            output: undefined,
            error: errorMsg,
            durationMs: Date.now() - stepStart,
            status: 'failed',
          };
          // Only retry if we have more attempts
        }
      }

      stepResults.push(stepResult!);
      lastStatus = stepResult!.status as 'success' | 'failed';

      if (stepResult!.status === 'success') {
        outputMap.set(step.name, stepResult!.output);
      } else {
        hasFailed = true;
        if (onError === 'stop') {
          return {
            pipelineName: pipeline.name,
            steps: stepResults,
            status: 'failed',
            totalDurationMs: Date.now() - startTime,
          };
        }
        // 'continue' or 'skip': keep going
      }
    }

    return {
      pipelineName: pipeline.name,
      steps: stepResults,
      status: hasFailed ? 'partial' : 'completed',
      totalDurationMs: Date.now() - startTime,
    };
  }

  /** Resolve $references in params using previous step results */
  resolveParams(
    params: Record<string, unknown>,
    stepResults: Map<string, unknown>,
  ): Record<string, unknown> {
    const resolved: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(params)) {
      resolved[key] = this.resolveValue(value, stepResults);
    }
    return resolved;
  }

  // ---- internal helpers ---------------------------------------------------

  private resolveValue(value: unknown, stepResults: Map<string, unknown>): unknown {
    if (typeof value === 'string') {
      return this.resolveStringRef(value, stepResults);
    }
    if (Array.isArray(value)) {
      return value.map((v) => this.resolveValue(v, stepResults));
    }
    if (value !== null && typeof value === 'object') {
      const resolved: Record<string, unknown> = {};
      for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
        resolved[k] = this.resolveValue(v, stepResults);
      }
      return resolved;
    }
    return value;
  }

  private resolveStringRef(value: string, stepResults: Map<string, unknown>): unknown {
    // Exact match: "$stepName.field" or "$stepName.field.nested"
    const exactMatch = /^\$([a-zA-Z_]\w*)\.(.+)$/.exec(value);
    if (exactMatch && value === exactMatch[0]) {
      const [, stepName, fieldPath] = exactMatch;
      const stepOutput = stepResults.get(stepName);
      if (stepOutput === undefined) return value;
      return this.getNestedValue(stepOutput, fieldPath);
    }

    // Exact reference to full step output: "$stepName"
    const fullRef = /^\$([a-zA-Z_]\w*)$/.exec(value);
    if (fullRef && value === fullRef[0]) {
      const stepOutput = stepResults.get(fullRef[1]);
      if (stepOutput !== undefined) return stepOutput;
      return value;
    }

    // Inline interpolation: "text $stepName.field more text"
    return value.replace(/\$([a-zA-Z_]\w*)\.([a-zA-Z_]\w*(?:\.[a-zA-Z_]\w*)*)/g, (_match, stepName, fieldPath) => {
      const stepOutput = stepResults.get(stepName);
      if (stepOutput === undefined) return _match;
      const resolved = this.getNestedValue(stepOutput, fieldPath);
      return String(resolved);
    });
  }

  private getNestedValue(obj: unknown, path: string): unknown {
    const parts = path.split('.');
    let current: unknown = obj;
    for (const part of parts) {
      if (current === null || current === undefined || typeof current !== 'object') {
        return undefined;
      }
      current = (current as Record<string, unknown>)[part];
    }
    return current;
  }

  private evaluateCondition(condition: string, lastStatus: 'success' | 'failed'): boolean {
    switch (condition) {
      case 'always':
        return true;
      case 'on_success':
        return lastStatus === 'success';
      case 'on_failure':
        return lastStatus === 'failed';
      default:
        // Unknown condition expressions default to true
        return true;
    }
  }

  // ---------------------------------------------------------------------------
  // Static helpers
  // ---------------------------------------------------------------------------

  /** Parse a pipeline definition from JSON or simple YAML string */
  static parse(input: string): PipelineDefinition {
    const trimmed = input.trim();

    // Try JSON first
    if (trimmed.startsWith('{')) {
      return JSON.parse(trimmed) as PipelineDefinition;
    }

    // Simple YAML-like parser
    return PipelineExecutor.parseSimpleYaml(trimmed);
  }

  /** Validate a pipeline definition, returning a list of errors (empty = valid) */
  static validate(pipeline: PipelineDefinition): string[] {
    const errors: string[] = [];

    if (!pipeline.name || typeof pipeline.name !== 'string') {
      errors.push('Pipeline name is required and must be a string');
    }

    if (!Array.isArray(pipeline.steps) || pipeline.steps.length === 0) {
      errors.push('Pipeline must have at least one step');
    } else {
      for (let i = 0; i < pipeline.steps.length; i++) {
        const step = pipeline.steps[i];
        if (!step.name || typeof step.name !== 'string') {
          errors.push(`Step ${i}: name is required`);
        }
        if (!step.tool || typeof step.tool !== 'string') {
          errors.push(`Step ${i}: tool is required`);
        } else if (!step.tool.includes('.')) {
          errors.push(`Step ${i}: tool must be in "skillName.toolName" format`);
        }
      }
    }

    if (
      pipeline.onError !== undefined &&
      !['stop', 'continue', 'skip'].includes(pipeline.onError)
    ) {
      errors.push('onError must be "stop", "continue", or "skip"');
    }

    return errors;
  }

  // ---------------------------------------------------------------------------
  // Simple YAML parser (no external dependency)
  // ---------------------------------------------------------------------------

  private static parseSimpleYaml(yaml: string): PipelineDefinition {
    const lines = yaml.split('\n');
    const result: Record<string, unknown> = {};
    const steps: Record<string, unknown>[] = [];
    let currentStep: Record<string, unknown> | null = null;
    let currentParams: Record<string, unknown> | null = null;
    let inSteps = false;
    let inParams = false;

    for (const rawLine of lines) {
      const line = rawLine.replace(/\r$/, '');
      // Skip empty lines and comments
      if (line.trim() === '' || line.trim().startsWith('#')) continue;

      const indent = line.length - line.trimStart().length;
      const trimmedLine = line.trim();

      // Top-level key-value
      if (indent === 0 && trimmedLine.includes(':')) {
        inParams = false;
        const colonIdx = trimmedLine.indexOf(':');
        const key = trimmedLine.slice(0, colonIdx).trim();
        const val = trimmedLine.slice(colonIdx + 1).trim();

        if (key === 'steps') {
          inSteps = true;
          continue;
        }
        inSteps = false;
        result[key] = PipelineExecutor.parseYamlValue(val);
        continue;
      }

      if (inSteps) {
        // New step item: "  - name: ..."
        if (trimmedLine.startsWith('- ')) {
          inParams = false;
          if (currentStep) {
            if (currentParams && Object.keys(currentParams).length > 0) {
              currentStep.params = currentParams;
            }
            steps.push(currentStep);
          }
          currentStep = {};
          currentParams = null;
          const content = trimmedLine.slice(2).trim();
          if (content.includes(':')) {
            const ci = content.indexOf(':');
            const k = content.slice(0, ci).trim();
            const v = content.slice(ci + 1).trim();
            currentStep[k] = PipelineExecutor.parseYamlValue(v);
          }
          continue;
        }

        // Step property
        if (currentStep && trimmedLine.includes(':')) {
          const ci = trimmedLine.indexOf(':');
          const k = trimmedLine.slice(0, ci).trim();
          const v = trimmedLine.slice(ci + 1).trim();

          if (k === 'params' && v === '') {
            inParams = true;
            currentParams = {};
            continue;
          }

          if (inParams && indent >= 6) {
            if (!currentParams) currentParams = {};
            currentParams[k] = PipelineExecutor.parseYamlValue(v);
            continue;
          }

          inParams = false;
          currentStep[k] = PipelineExecutor.parseYamlValue(v);
        }
      }
    }

    // Push last step
    if (currentStep) {
      if (currentParams && Object.keys(currentParams).length > 0) {
        currentStep.params = currentParams;
      }
      steps.push(currentStep);
    }

    return {
      name: result.name as string ?? '',
      description: result.description as string | undefined,
      steps: steps.map((s) => ({
        name: (s.name as string) ?? '',
        tool: (s.tool as string) ?? '',
        params: (s.params as Record<string, unknown>) ?? {},
        condition: s.condition as string | undefined,
        retries: s.retries as number | undefined,
        timeout: s.timeout as number | undefined,
      })),
      onError: result.onError as PipelineDefinition['onError'],
    };
  }

  private static parseYamlValue(val: string): unknown {
    if (val === '' || val === 'null' || val === '~') return undefined;
    if (val === 'true') return true;
    if (val === 'false') return false;
    // Remove quotes
    if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
      return val.slice(1, -1);
    }
    const num = Number(val);
    if (!isNaN(num) && val !== '') return num;
    return val;
  }
}
