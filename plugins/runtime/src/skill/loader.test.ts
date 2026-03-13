import { describe, test, expect, beforeEach, afterEach } from "bun:test";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { SkillLoader, type SkillManifest } from "./loader";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function validManifest(overrides?: Partial<SkillManifest>): SkillManifest {
  return {
    name: "test-skill",
    version: "1.0.0",
    description: "A test skill",
    entry: "index.ts",
    permissions: ["market-data"],
    ...overrides,
  };
}

async function writeManifest(dir: string, name: string, manifest: unknown): Promise<string> {
  const skillDir = join(dir, name);
  await Bun.write(join(skillDir, "skill.json"), JSON.stringify(manifest));
  return skillDir;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("SkillLoader", () => {
  let tmpDir: string;

  beforeEach(async () => {
    tmpDir = await mkdtemp(join(tmpdir(), "skill-loader-test-"));
  });

  afterEach(async () => {
    await rm(tmpDir, { recursive: true, force: true });
  });

  // -----------------------------------------------------------------------
  // validateManifest
  // -----------------------------------------------------------------------

  describe("validateManifest", () => {
    const loader = new SkillLoader("/dummy");

    test("accepts a valid manifest", () => {
      expect(loader.validateManifest(validManifest())).toBe(true);
    });

    test("accepts manifest with optional author", () => {
      expect(loader.validateManifest(validManifest({ author: "tester" }))).toBe(true);
    });

    test("accepts manifest with resources", () => {
      const m = validManifest({
        resources: { maxMemoryMB: 128, timeoutMs: 5000, maxConcurrent: 3 },
      });
      expect(loader.validateManifest(m)).toBe(true);
    });

    test("accepts manifest with partial resources", () => {
      const m = validManifest({ resources: { timeoutMs: 1000 } });
      expect(loader.validateManifest(m)).toBe(true);
    });

    test("rejects null", () => {
      expect(loader.validateManifest(null)).toBe(false);
    });

    test("rejects non-object", () => {
      expect(loader.validateManifest("not an object")).toBe(false);
      expect(loader.validateManifest(42)).toBe(false);
    });

    test("rejects missing name", () => {
      const { name, ...rest } = validManifest();
      expect(loader.validateManifest(rest)).toBe(false);
    });

    test("rejects empty name", () => {
      expect(loader.validateManifest(validManifest({ name: "" }))).toBe(false);
    });

    test("rejects missing version", () => {
      const { version, ...rest } = validManifest();
      expect(loader.validateManifest(rest)).toBe(false);
    });

    test("rejects missing description", () => {
      const { description, ...rest } = validManifest();
      expect(loader.validateManifest(rest)).toBe(false);
    });

    test("rejects missing entry", () => {
      const { entry, ...rest } = validManifest();
      expect(loader.validateManifest(rest)).toBe(false);
    });

    test("rejects missing permissions", () => {
      const { permissions, ...rest } = validManifest();
      expect(loader.validateManifest(rest)).toBe(false);
    });

    test("rejects non-array permissions", () => {
      expect(loader.validateManifest({ ...validManifest(), permissions: "bad" })).toBe(false);
    });

    test("rejects permissions with non-string elements", () => {
      expect(loader.validateManifest({ ...validManifest(), permissions: [123] })).toBe(false);
    });

    test("rejects non-string author", () => {
      expect(loader.validateManifest({ ...validManifest(), author: 42 })).toBe(false);
    });

    test("rejects resources with wrong types", () => {
      expect(
        loader.validateManifest({ ...validManifest(), resources: { maxMemoryMB: "big" } }),
      ).toBe(false);
    });

    test("rejects resources as non-object", () => {
      expect(loader.validateManifest({ ...validManifest(), resources: "bad" })).toBe(false);
    });

    test("accepts empty permissions array", () => {
      expect(loader.validateManifest(validManifest({ permissions: [] }))).toBe(true);
    });
  });

  // -----------------------------------------------------------------------
  // discover
  // -----------------------------------------------------------------------

  describe("discover", () => {
    test("finds skills in directory", async () => {
      await writeManifest(tmpDir, "skill-a", validManifest({ name: "skill-a" }));
      await writeManifest(tmpDir, "skill-b", validManifest({ name: "skill-b", version: "2.0.0" }));

      const loader = new SkillLoader(tmpDir);
      const manifests = await loader.discover();

      expect(manifests).toHaveLength(2);
      const names = manifests.map((m) => m.name).sort();
      expect(names).toEqual(["skill-a", "skill-b"]);
    });

    test("returns empty array for non-existent directory", async () => {
      const loader = new SkillLoader(join(tmpDir, "nope"));
      const manifests = await loader.discover();
      expect(manifests).toEqual([]);
    });

    test("skips directories without skill.json", async () => {
      // Create a directory with no manifest
      await Bun.write(join(tmpDir, "no-manifest", "other.txt"), "hello");
      // Create one with a valid manifest
      await writeManifest(tmpDir, "valid-skill", validManifest({ name: "valid-skill" }));

      const loader = new SkillLoader(tmpDir);
      const manifests = await loader.discover();

      expect(manifests).toHaveLength(1);
      expect(manifests[0].name).toBe("valid-skill");
    });

    test("skips directories with invalid manifests", async () => {
      await writeManifest(tmpDir, "bad-skill", { name: "bad" }); // missing required fields
      await writeManifest(tmpDir, "good-skill", validManifest({ name: "good-skill" }));

      const loader = new SkillLoader(tmpDir);
      const manifests = await loader.discover();

      expect(manifests).toHaveLength(1);
      expect(manifests[0].name).toBe("good-skill");
    });
  });

  // -----------------------------------------------------------------------
  // get / listLoaded / load / unload state management
  // -----------------------------------------------------------------------

  describe("state management", () => {
    test("get returns undefined for unloaded skill", () => {
      const loader = new SkillLoader(tmpDir);
      expect(loader.get("nonexistent")).toBeUndefined();
    });

    test("listLoaded returns empty array initially", () => {
      const loader = new SkillLoader(tmpDir);
      expect(loader.listLoaded()).toEqual([]);
    });

    test("unload throws for skill that is not loaded", async () => {
      const loader = new SkillLoader(tmpDir);
      await expect(loader.unload("nonexistent")).rejects.toThrow("not loaded");
    });

    test("load throws for missing skill directory", async () => {
      const loader = new SkillLoader(tmpDir);
      await expect(loader.load("missing-skill")).rejects.toThrow("not found");
    });

    test("load throws for invalid manifest", async () => {
      await writeManifest(tmpDir, "bad-skill", { name: "bad" });
      const loader = new SkillLoader(tmpDir);
      await expect(loader.load("bad-skill")).rejects.toThrow("invalid manifest");
    });
  });

  // -----------------------------------------------------------------------
  // Full load/unload cycle using a real skill entry point
  // -----------------------------------------------------------------------

  describe("load/unload lifecycle", () => {
    test("loads and unloads a skill with a real entry point", async () => {
      const manifest = validManifest({ name: "lifecycle-skill", permissions: [] });
      await writeManifest(tmpDir, "lifecycle-skill", manifest);

      // Write a minimal skill entry point
      const entryContent = `
        import { defineSkill } from "${join(import.meta.dir, "sdk").replace(/\\/g, "/")}";
        export default {
          name: "lifecycle-skill",
          version: "1.0.0",
          description: "A test skill",
          permissions: [],
          tools: [],
        };
      `;
      await Bun.write(join(tmpDir, "lifecycle-skill", "index.ts"), entryContent);

      const loader = new SkillLoader(tmpDir);

      // Load
      const loaded = await loader.load("lifecycle-skill");
      expect(loaded.skill.name).toBe("lifecycle-skill");
      expect(loaded.manifest.name).toBe("lifecycle-skill");
      expect(loaded.path).toBe(join(tmpDir, "lifecycle-skill"));
      expect(loaded.loadedAt).toBeGreaterThan(0);

      // get
      expect(loader.get("lifecycle-skill")).toBe(loaded);

      // listLoaded
      expect(loader.listLoaded()).toHaveLength(1);
      expect(loader.listLoaded()[0]).toBe(loaded);

      // Unload
      await loader.unload("lifecycle-skill");
      expect(loader.get("lifecycle-skill")).toBeUndefined();
      expect(loader.listLoaded()).toHaveLength(0);
    });

    test("load returns existing skill if already loaded", async () => {
      const manifest = validManifest({ name: "dup-skill", permissions: [] });
      await writeManifest(tmpDir, "dup-skill", manifest);

      const entryContent = `
        export default {
          name: "dup-skill",
          version: "1.0.0",
          description: "A test skill",
          permissions: [],
          tools: [],
        };
      `;
      await Bun.write(join(tmpDir, "dup-skill", "index.ts"), entryContent);

      const loader = new SkillLoader(tmpDir);
      const first = await loader.load("dup-skill");
      const second = await loader.load("dup-skill");
      expect(first).toBe(second);
    });
  });
});
