import { describe, test, expect, beforeEach, afterEach } from "bun:test";
import { mkdtemp, rm, readdir } from "node:fs/promises";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { CommunityRegistry, type RegistryEntry } from "./registry";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeEntry(overrides?: Partial<RegistryEntry>): RegistryEntry {
  return {
    name: "test-skill",
    version: "1.0.0",
    description: "A test skill for unit tests",
    author: "tester",
    repository: "https://github.com/test/test-skill.git",
    tags: ["trading", "test"],
    downloads: 0,
    rating: 0,
    publishedAt: 1000,
    updatedAt: 2000,
    ...overrides,
  };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("CommunityRegistry", () => {
  let dir: string;
  let registry: CommunityRegistry;

  beforeEach(async () => {
    dir = await mkdtemp(join(tmpdir(), "registry-test-"));
    registry = new CommunityRegistry(dir);
  });

  afterEach(async () => {
    await rm(dir, { recursive: true, force: true });
  });

  // ---- publish -----------------------------------------------------------

  test("publish adds entry to registry", () => {
    const entry = makeEntry();
    expect(registry.publish(entry)).toBe(true);
    expect(registry.get("test-skill")).toBeDefined();
  });

  test("publish validates required fields", () => {
    const bad = makeEntry({ name: "" });
    expect(registry.publish(bad)).toBe(false);
    expect(registry.count()).toBe(0);
  });

  test("publish rejects duplicate names", () => {
    const entry = makeEntry();
    expect(registry.publish(entry)).toBe(true);
    expect(registry.publish(entry)).toBe(false);
    expect(registry.count()).toBe(1);
  });

  // ---- get ---------------------------------------------------------------

  test("get returns entry by name", () => {
    registry.publish(makeEntry({ name: "alpha" }));
    const result = registry.get("alpha");
    expect(result).toBeDefined();
    expect(result!.name).toBe("alpha");
  });

  test("get returns undefined for unknown", () => {
    expect(registry.get("nonexistent")).toBeUndefined();
  });

  // ---- search ------------------------------------------------------------

  test("search by query matches name and description", () => {
    registry.publish(makeEntry({ name: "momentum", description: "Trend following strategy" }));
    registry.publish(makeEntry({ name: "mean-revert", description: "Mean reversion algo" }));

    const byName = registry.search({ query: "momentum" });
    expect(byName).toHaveLength(1);
    expect(byName[0].name).toBe("momentum");

    const byDesc = registry.search({ query: "reversion" });
    expect(byDesc).toHaveLength(1);
    expect(byDesc[0].name).toBe("mean-revert");
  });

  test("search by tags", () => {
    registry.publish(makeEntry({ name: "a", tags: ["trading", "crypto"] }));
    registry.publish(makeEntry({ name: "b", tags: ["trading", "stocks"] }));

    const results = registry.search({ tags: ["crypto"] });
    expect(results).toHaveLength(1);
    expect(results[0].name).toBe("a");
  });

  test("search by author", () => {
    registry.publish(makeEntry({ name: "x", author: "alice" }));
    registry.publish(makeEntry({ name: "y", author: "bob" }));

    const results = registry.search({ author: "alice" });
    expect(results).toHaveLength(1);
    expect(results[0].name).toBe("x");
  });

  test("search with sortBy downloads", () => {
    registry.publish(makeEntry({ name: "low", downloads: 10 }));
    registry.publish(makeEntry({ name: "high", downloads: 500 }));
    registry.publish(makeEntry({ name: "mid", downloads: 100 }));

    const results = registry.search({ sortBy: "downloads" });
    expect(results.map((r) => r.name)).toEqual(["high", "mid", "low"]);
  });

  test("search with sortBy rating", () => {
    registry.publish(makeEntry({ name: "ok", rating: 3 }));
    registry.publish(makeEntry({ name: "great", rating: 5 }));
    registry.publish(makeEntry({ name: "meh", rating: 1 }));

    const results = registry.search({ sortBy: "rating" });
    expect(results.map((r) => r.name)).toEqual(["great", "ok", "meh"]);
  });

  test("search with limit and offset pagination", () => {
    for (let i = 0; i < 10; i++) {
      registry.publish(makeEntry({ name: `skill-${i}`, downloads: i }));
    }

    const page1 = registry.search({ sortBy: "downloads", limit: 3, offset: 0 });
    expect(page1).toHaveLength(3);
    expect(page1[0].name).toBe("skill-9"); // highest downloads first

    const page2 = registry.search({ sortBy: "downloads", limit: 3, offset: 3 });
    expect(page2).toHaveLength(3);
    expect(page2[0].name).toBe("skill-6");
  });

  // ---- install -----------------------------------------------------------

  test("install creates result with success", async () => {
    registry.publish(makeEntry({ name: "installable" }));
    const result = await registry.install("installable");
    expect(result.success).toBe(true);
    expect(result.name).toBe("installable");
    expect(result.version).toBe("1.0.0");
    expect(result.path).toBe(join(dir, "installable"));

    // Verify directory and manifest exist
    const items = await readdir(join(dir, "installable"));
    expect(items).toContain("skill.json");
  });

  test("install unknown skill returns error", async () => {
    const result = await registry.install("ghost");
    expect(result.success).toBe(false);
    expect(result.error).toBeDefined();
  });

  // ---- uninstall ---------------------------------------------------------

  test("uninstall removes directory", async () => {
    registry.publish(makeEntry({ name: "removable" }));
    await registry.install("removable");
    const ok = await registry.uninstall("removable");
    expect(ok).toBe(true);

    const installed = await registry.listInstalled();
    expect(installed).not.toContain("removable");
  });

  // ---- rate --------------------------------------------------------------

  test("rate updates rating", () => {
    registry.publish(makeEntry({ name: "rateable", rating: 0 }));
    expect(registry.rate("rateable", 4)).toBe(true);
    expect(registry.get("rateable")!.rating).toBe(4);
  });

  test("rate rejects invalid rating (> 5)", () => {
    registry.publish(makeEntry({ name: "rateable2" }));
    expect(registry.rate("rateable2", 6)).toBe(false);
    expect(registry.rate("rateable2", -1)).toBe(false);
  });

  // ---- recordDownload ----------------------------------------------------

  test("recordDownload increments count", () => {
    registry.publish(makeEntry({ name: "popular", downloads: 10 }));
    registry.recordDownload("popular");
    registry.recordDownload("popular");
    expect(registry.get("popular")!.downloads).toBe(12);
  });

  // ---- getPopular / getRecent --------------------------------------------

  test("getPopular returns sorted by downloads", () => {
    registry.publish(makeEntry({ name: "a", downloads: 5 }));
    registry.publish(makeEntry({ name: "b", downloads: 100 }));
    registry.publish(makeEntry({ name: "c", downloads: 50 }));

    const popular = registry.getPopular(2);
    expect(popular).toHaveLength(2);
    expect(popular[0].name).toBe("b");
    expect(popular[1].name).toBe("c");
  });

  test("getRecent returns sorted by updatedAt", () => {
    registry.publish(makeEntry({ name: "old", updatedAt: 1000 }));
    registry.publish(makeEntry({ name: "new", updatedAt: 9000 }));
    registry.publish(makeEntry({ name: "mid", updatedAt: 5000 }));

    const recent = registry.getRecent(2);
    expect(recent).toHaveLength(2);
    expect(recent[0].name).toBe("new");
    expect(recent[1].name).toBe("mid");
  });

  // ---- validate ----------------------------------------------------------

  test("validate catches missing fields", () => {
    const errors = CommunityRegistry.validate({});
    expect(errors.length).toBeGreaterThan(0);
    expect(errors.some((e) => e.includes("name"))).toBe(true);
    expect(errors.some((e) => e.includes("version"))).toBe(true);
    expect(errors.some((e) => e.includes("description"))).toBe(true);
    expect(errors.some((e) => e.includes("author"))).toBe(true);
    expect(errors.some((e) => e.includes("repository"))).toBe(true);
  });

  // ---- count -------------------------------------------------------------

  test("count returns correct number", () => {
    expect(registry.count()).toBe(0);
    registry.publish(makeEntry({ name: "one" }));
    registry.publish(makeEntry({ name: "two" }));
    expect(registry.count()).toBe(2);
  });
});
