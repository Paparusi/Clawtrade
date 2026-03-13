// Skill Loader - Discovery, loading, sandboxing, and resource limits for skills.

import { readdir } from "node:fs/promises";
import { join } from "node:path";
import { defineSkill, type Skill, type SkillDefinition } from "./sdk";

// ---------------------------------------------------------------------------
// Manifest interface
// ---------------------------------------------------------------------------

export interface SkillManifest {
  name: string;
  version: string;
  description: string;
  author?: string;
  entry: string; // e.g. "index.ts"
  permissions: string[]; // e.g. ["market-data", "trade-read"]
  resources?: {
    maxMemoryMB?: number;
    timeoutMs?: number;
    maxConcurrent?: number;
  };
}

// ---------------------------------------------------------------------------
// LoadedSkill - a skill plus its metadata
// ---------------------------------------------------------------------------

export interface LoadedSkill {
  skill: Skill;
  manifest: SkillManifest;
  path: string;
  loadedAt: number;
}

// ---------------------------------------------------------------------------
// SkillLoader
// ---------------------------------------------------------------------------

export class SkillLoader {
  private skills: Map<string, LoadedSkill> = new Map();
  private skillsDir: string;

  constructor(skillsDir: string) {
    this.skillsDir = skillsDir;
  }

  /**
   * Discover all skills in the skills directory.
   * Scans for subdirectories containing a valid skill.json manifest.
   */
  async discover(): Promise<SkillManifest[]> {
    const manifests: SkillManifest[] = [];

    let entries: string[];
    try {
      entries = await readdir(this.skillsDir);
    } catch {
      return manifests;
    }

    for (const entry of entries) {
      const skillDir = join(this.skillsDir, entry);
      const manifestPath = join(skillDir, "skill.json");

      try {
        const file = Bun.file(manifestPath);
        const exists = await file.exists();
        if (!exists) continue;

        const raw = await file.json();
        if (this.validateManifest(raw)) {
          manifests.push(raw);
        }
      } catch {
        // Skip directories that don't have a valid skill.json
        continue;
      }
    }

    return manifests;
  }

  /**
   * Load a skill by name.
   * Reads its manifest, dynamically imports the entry point module,
   * creates a Skill instance, and invokes the load() lifecycle hook.
   */
  async load(name: string): Promise<LoadedSkill> {
    // Don't reload an already-loaded skill
    const existing = this.skills.get(name);
    if (existing) return existing;

    const skillDir = join(this.skillsDir, name);
    const manifestPath = join(skillDir, "skill.json");

    // Read and validate manifest
    const file = Bun.file(manifestPath);
    const exists = await file.exists();
    if (!exists) {
      throw new Error(`Skill '${name}' not found: missing skill.json at ${manifestPath}`);
    }

    const raw = await file.json();
    if (!this.validateManifest(raw)) {
      throw new Error(`Skill '${name}' has an invalid manifest`);
    }

    const manifest: SkillManifest = raw;

    // Dynamically import the entry point
    const entryPath = join(skillDir, manifest.entry);
    let mod: Record<string, unknown>;
    try {
      mod = await import(entryPath);
    } catch (err) {
      throw new Error(
        `Failed to import skill '${name}' entry point at ${entryPath}: ${err instanceof Error ? err.message : String(err)}`,
      );
    }

    // The module should export a SkillDefinition as default
    const skillDef = (mod.default ?? mod) as SkillDefinition;
    if (!skillDef || typeof skillDef !== "object" || !skillDef.name) {
      throw new Error(
        `Skill '${name}' entry point must export a SkillDefinition as default export`,
      );
    }

    // Create the Skill instance
    const skill = defineSkill(skillDef);

    // Call load lifecycle hook
    await skill.load();

    const loaded: LoadedSkill = {
      skill,
      manifest,
      path: skillDir,
      loadedAt: Date.now(),
    };

    this.skills.set(name, loaded);
    return loaded;
  }

  /**
   * Unload a skill by name.
   * Calls the unload() lifecycle hook and removes from the loaded map.
   */
  async unload(name: string): Promise<void> {
    const loaded = this.skills.get(name);
    if (!loaded) {
      throw new Error(`Skill '${name}' is not loaded`);
    }

    await loaded.skill.unload();
    this.skills.delete(name);
  }

  /**
   * Get a loaded skill by name.
   */
  get(name: string): LoadedSkill | undefined {
    return this.skills.get(name);
  }

  /**
   * List all currently loaded skills.
   */
  listLoaded(): LoadedSkill[] {
    return Array.from(this.skills.values());
  }

  /**
   * Validate that an unknown value conforms to the SkillManifest shape.
   */
  validateManifest(manifest: unknown): manifest is SkillManifest {
    if (!manifest || typeof manifest !== "object") return false;

    const m = manifest as Record<string, unknown>;

    // Required string fields
    if (typeof m.name !== "string" || m.name.length === 0) return false;
    if (typeof m.version !== "string" || m.version.length === 0) return false;
    if (typeof m.description !== "string" || m.description.length === 0) return false;
    if (typeof m.entry !== "string" || m.entry.length === 0) return false;

    // Optional author must be string if present
    if (m.author !== undefined && typeof m.author !== "string") return false;

    // permissions must be an array of strings
    if (!Array.isArray(m.permissions)) return false;
    for (const p of m.permissions) {
      if (typeof p !== "string") return false;
    }

    // Optional resources object
    if (m.resources !== undefined) {
      if (typeof m.resources !== "object" || m.resources === null) return false;
      const r = m.resources as Record<string, unknown>;
      if (r.maxMemoryMB !== undefined && typeof r.maxMemoryMB !== "number") return false;
      if (r.timeoutMs !== undefined && typeof r.timeoutMs !== "number") return false;
      if (r.maxConcurrent !== undefined && typeof r.maxConcurrent !== "number") return false;
    }

    return true;
  }
}
