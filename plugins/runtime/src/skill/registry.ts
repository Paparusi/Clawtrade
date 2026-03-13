// Community Registry – Git-based skill publishing, install, and search.

import { readdir, mkdir, rm } from "node:fs/promises";
import { join } from "node:path";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface RegistryEntry {
  name: string;
  version: string;
  description: string;
  author: string;
  repository: string; // git URL
  tags: string[];
  downloads: number;
  rating: number; // 0-5
  publishedAt: number; // timestamp
  updatedAt: number;
}

export interface SearchOptions {
  query?: string;
  tags?: string[];
  author?: string;
  sortBy?: "downloads" | "rating" | "recent" | "name";
  limit?: number;
  offset?: number;
}

export interface InstallResult {
  name: string;
  version: string;
  path: string;
  success: boolean;
  error?: string;
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

export class CommunityRegistry {
  private entries: Map<string, RegistryEntry> = new Map();
  private skillsDir: string;

  constructor(skillsDir: string) {
    this.skillsDir = skillsDir;
  }

  // ---- Search & Discovery ------------------------------------------------

  search(options: SearchOptions): RegistryEntry[] {
    let results = Array.from(this.entries.values());

    // Filter by query (matches name or description, case-insensitive)
    if (options.query) {
      const q = options.query.toLowerCase();
      results = results.filter(
        (e) =>
          e.name.toLowerCase().includes(q) ||
          e.description.toLowerCase().includes(q),
      );
    }

    // Filter by tags (entry must contain ALL specified tags)
    if (options.tags && options.tags.length > 0) {
      const wanted = new Set(options.tags.map((t) => t.toLowerCase()));
      results = results.filter((e) =>
        [...wanted].every((t) => e.tags.some((et) => et.toLowerCase() === t)),
      );
    }

    // Filter by author
    if (options.author) {
      const a = options.author.toLowerCase();
      results = results.filter((e) => e.author.toLowerCase() === a);
    }

    // Sort
    switch (options.sortBy) {
      case "downloads":
        results.sort((a, b) => b.downloads - a.downloads);
        break;
      case "rating":
        results.sort((a, b) => b.rating - a.rating);
        break;
      case "recent":
        results.sort((a, b) => b.updatedAt - a.updatedAt);
        break;
      case "name":
        results.sort((a, b) => a.name.localeCompare(b.name));
        break;
      default:
        // no-op, keep insertion order
        break;
    }

    // Pagination
    const offset = options.offset ?? 0;
    const limit = options.limit ?? results.length;
    return results.slice(offset, offset + limit);
  }

  get(name: string): RegistryEntry | undefined {
    return this.entries.get(name);
  }

  // ---- Install / Uninstall / Update --------------------------------------

  async install(name: string): Promise<InstallResult> {
    const entry = this.entries.get(name);
    if (!entry) {
      return { name, version: "", path: "", success: false, error: "Skill not found in registry" };
    }

    const destPath = join(this.skillsDir, name);

    try {
      await mkdir(destPath, { recursive: true });

      // Write a manifest file so the skill loader can discover it later.
      const manifest = {
        name: entry.name,
        version: entry.version,
        description: entry.description,
        author: entry.author,
        repository: entry.repository,
        tags: entry.tags,
      };
      await Bun.write(join(destPath, "skill.json"), JSON.stringify(manifest, null, 2));

      this.recordDownload(name);

      return { name, version: entry.version, path: destPath, success: true };
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      return { name, version: entry.version, path: destPath, success: false, error: message };
    }
  }

  async uninstall(name: string): Promise<boolean> {
    const destPath = join(this.skillsDir, name);
    try {
      await rm(destPath, { recursive: true, force: true });
      return true;
    } catch {
      return false;
    }
  }

  async update(name: string): Promise<InstallResult> {
    const entry = this.entries.get(name);
    if (!entry) {
      return { name, version: "", path: "", success: false, error: "Skill not found in registry" };
    }

    // Re-install to get the latest version
    await this.uninstall(name);
    return this.install(name);
  }

  // ---- Publishing --------------------------------------------------------

  publish(entry: RegistryEntry): boolean {
    const errors = CommunityRegistry.validate(entry);
    if (errors.length > 0) return false;

    // Reject duplicates
    if (this.entries.has(entry.name)) return false;

    this.entries.set(entry.name, { ...entry });
    return true;
  }

  // ---- Installed ---------------------------------------------------------

  async listInstalled(): Promise<string[]> {
    try {
      const items = await readdir(this.skillsDir, { withFileTypes: true });
      return items.filter((d) => d.isDirectory()).map((d) => d.name);
    } catch {
      return [];
    }
  }

  // ---- Ratings & Downloads -----------------------------------------------

  rate(name: string, rating: number): boolean {
    if (rating < 0 || rating > 5) return false;
    const entry = this.entries.get(name);
    if (!entry) return false;
    entry.rating = rating;
    return true;
  }

  recordDownload(name: string): void {
    const entry = this.entries.get(name);
    if (entry) {
      entry.downloads += 1;
    }
  }

  // ---- Convenience -------------------------------------------------------

  getPopular(limit = 10): RegistryEntry[] {
    return this.search({ sortBy: "downloads", limit });
  }

  getRecent(limit = 10): RegistryEntry[] {
    return this.search({ sortBy: "recent", limit });
  }

  count(): number {
    return this.entries.size;
  }

  // ---- Validation --------------------------------------------------------

  static validate(entry: unknown): string[] {
    const errors: string[] = [];
    if (typeof entry !== "object" || entry === null) {
      return ["entry must be a non-null object"];
    }

    const e = entry as Record<string, unknown>;

    const requiredStrings: (keyof RegistryEntry)[] = [
      "name",
      "version",
      "description",
      "author",
      "repository",
    ];

    for (const field of requiredStrings) {
      if (typeof e[field] !== "string" || (e[field] as string).length === 0) {
        errors.push(`${field} is required and must be a non-empty string`);
      }
    }

    if (!Array.isArray(e.tags)) {
      errors.push("tags must be an array");
    }

    if (typeof e.downloads !== "number") {
      errors.push("downloads must be a number");
    }

    if (typeof e.rating !== "number" || (e.rating as number) < 0 || (e.rating as number) > 5) {
      errors.push("rating must be a number between 0 and 5");
    }

    if (typeof e.publishedAt !== "number") {
      errors.push("publishedAt must be a number");
    }

    if (typeof e.updatedAt !== "number") {
      errors.push("updatedAt must be a number");
    }

    return errors;
  }
}
