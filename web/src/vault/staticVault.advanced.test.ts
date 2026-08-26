import { describe, expect, it } from "vitest";
import { buildVaultFromRaw, searchAdvancedInNotes } from "./staticVault";
import type { SearchRequest } from "../types";

const RAW_FILES: Record<string, string> = {
  "./notes/Research/KB/Alpha.md":
    "---\ntitle: Alpha\ntags: [go, performance]\ncreated: 2024-01-15\nupdated: 2024-02-20T09:00:00Z\n---\n# Alpha\n\ncommon content\n",
  "./notes/Research/KB/Beta.md":
    "---\ntitle: Beta\ntags: [go]\ncreated: 2024-03-01\n---\n# Beta\n\ncommon content\n",
  "./notes/Projects/Gamma.md":
    "---\ntitle: Gamma\ntags: [rust]\ncreated: 2024-01-10\n---\n# Gamma\n\ncommon content\n",
  "./notes/Notes/Plain.md": "# Plain\n\ncommon content\n",
};

function vault() {
  return buildVaultFromRaw(RAW_FILES).notes;
}

function slugs(resp: { results: { slug: string }[] }): string[] {
  return resp.results.map((r) => r.slug);
}

function req(partial: Partial<SearchRequest>): SearchRequest {
  return {
    query: "",
    tags: [],
    tagMode: "all",
    pathPrefixes: [],
    dateField: "",
    sort: "",
    limit: 0,
    offset: 0,
    ...partial,
  };
}

describe("staticSearchAdvanced (parity with backend)", () => {
  it("exact tag all matches only notes with every tag", () => {
    const resp = searchAdvancedInNotes(vault(), req({ tags: ["go", "performance"], tagMode: "all" }));
    expect(slugs(resp)).toEqual(["research/kb/alpha"]);
    expect(resp.total).toBe(1);
  });

  it("exact tag any matches notes with any tag", () => {
    const resp = searchAdvancedInNotes(vault(), req({ tags: ["go", "rust"], tagMode: "any" }));
    expect(resp.total).toBe(3);
  });

  it("path prefix filters by folder", () => {
    const resp = searchAdvancedInNotes(vault(), req({ pathPrefixes: ["research/kb/"] }));
    expect(slugs(resp).sort()).toEqual(["research/kb/alpha", "research/kb/beta"]);
    for (const r of resp.results) expect(r.path).toBeTruthy();
  });

  it("date range over display includes the right notes", () => {
    const resp = searchAdvancedInNotes(
      vault(),
      req({ dateField: "display", dateFrom: { year: 2024, month: 2, day: 1 }, dateTo: { year: 2024, month: 3, day: 1 } }),
    );
    expect(slugs(resp).sort()).toEqual(["research/kb/alpha", "research/kb/beta"]);
  });

  it("sort newest orders by display date descending with undated last", () => {
    const resp = searchAdvancedInNotes(vault(), req({ query: "content", sort: "newest" }));
    expect(slugs(resp)).toEqual(["research/kb/beta", "research/kb/alpha", "projects/gamma", "notes/plain"]);
  });

  it("sort oldest orders ascending with undated last", () => {
    const resp = searchAdvancedInNotes(vault(), req({ query: "content", sort: "oldest" }));
    expect(slugs(resp)).toEqual(["projects/gamma", "research/kb/alpha", "research/kb/beta", "notes/plain"]);
  });

  it("paginates and reports total", () => {
    const resp = searchAdvancedInNotes(vault(), req({ query: "content", sort: "newest", limit: 2, offset: 0 }));
    expect(resp.total).toBe(4);
    expect(resp.results).toHaveLength(2);
    const resp2 = searchAdvancedInNotes(vault(), req({ query: "content", sort: "newest", limit: 2, offset: 2 }));
    expect(resp2.results).toHaveLength(2);
  });

  it("reconstructs the display date in results", () => {
    const resp = searchAdvancedInNotes(vault(), req({ tags: ["performance"] }));
    expect(resp.results[0].date).toEqual({
      value: "2024-02-20T09:00:00Z",
      kind: "updated",
      precision: "timestamp",
    });
  });

  it("legacy #go discovery uses the pinned prefix contract", () => {
    const resp = searchAdvancedInNotes(vault(), req({ query: "#go", sort: "relevance" }));
    // #go (len 2) prefix-matches tags starting with "go": Alpha and Beta.
    expect(slugs(resp).sort()).toEqual(["research/kb/alpha", "research/kb/beta"]);
  });

  it("ineffective request returns empty", () => {
    const resp = searchAdvancedInNotes(vault(), req({}));
    expect(resp.total).toBe(0);
    expect(resp.results).toEqual([]);
  });

  it("compound query narrows to one note", () => {
    const resp = searchAdvancedInNotes(
      vault(),
      req({
        tags: ["go"],
        pathPrefixes: ["research/kb/"],
        dateField: "display",
        dateFrom: { year: 2024, month: 2, day: 1 },
        dateTo: { year: 2024, month: 2, day: 28 },
      }),
    );
    expect(slugs(resp)).toEqual(["research/kb/alpha"]);
  });
});
