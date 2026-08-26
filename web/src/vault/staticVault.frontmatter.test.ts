import { describe, expect, it } from "vitest";
import { parseFrontmatter, serializeFrontmatter } from "./staticVault";

/**
 * Regression coverage for the PV-SEARCH-027 review finding: the default
 * js-yaml schema resolved unquoted RFC3339 scalars to Date objects and
 * serializeFrontmatter truncated them to YYYY-MM-DD, losing the instant and
 * timestamp precision before authored-date resolution. parseFrontmatter now
 * uses JSON_SCHEMA so scalars stay strings through parse + serialize.
 */
describe("parseFrontmatter preserves date scalars as strings", () => {
  it("keeps an unquoted date-only value as a string", () => {
    const { data } = parseFrontmatter("---\ncreated: 2024-01-15\n---\n# x\n");
    expect(typeof data.created).toBe("string");
    expect(data.created).toBe("2024-01-15");
  });

  it("keeps a quoted date-only value as a string", () => {
    const { data } = parseFrontmatter(
      '---\ncreated: "2024-01-15"\n---\n# x\n',
    );
    expect(typeof data.created).toBe("string");
    expect(data.created).toBe("2024-01-15");
  });

  it("keeps an unquoted RFC3339 value as a full-precision string (not a Date)", () => {
    const { data } = parseFrontmatter(
      "---\nupdated: 2024-01-15T13:45:00-05:00\n---\n# x\n",
    );
    expect(typeof data.updated).toBe("string");
    expect(data.updated).toBe("2024-01-15T13:45:00-05:00");
    expect(data.updated instanceof Date).toBe(false);
  });

  it("keeps a Z RFC3339 value as a full-precision string", () => {
    const { data } = parseFrontmatter(
      "---\nupdated: 2024-02-20T09:00:00Z\n---\n# x\n",
    );
    expect(typeof data.updated).toBe("string");
    expect(data.updated).toBe("2024-02-20T09:00:00Z");
  });

  it("survives serializeFrontmatter without truncation", () => {
    const { data } = parseFrontmatter(
      '---\ncreated: "2024-01-15"\nupdated: 2024-01-15T13:45:00-05:00\n---\n# x\n',
    );
    const safe = serializeFrontmatter(data);
    expect(safe.created).toBe("2024-01-15");
    expect(safe.updated).toBe("2024-01-15T13:45:00-05:00");
    expect(safe.updated instanceof Date).toBe(false);
  });

  it("still parses ordinary JSON-compatible scalars", () => {
    const { data } = parseFrontmatter(
      '---\ntitle: Hello\ncount: 3\nflag: true\ntags: [a, b]\n---\n# x\n',
    );
    expect(data.title).toBe("Hello");
    expect(data.count).toBe(3);
    expect(data.flag).toBe(true);
    expect(data.tags).toEqual(["a", "b"]);
  });
});
