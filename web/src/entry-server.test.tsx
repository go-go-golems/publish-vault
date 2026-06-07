import { describe, expect, it } from "vitest";
import { renderApp, parseRoute } from "./entry-server";
import type { Note, NoteListItem, SiteConfig, FileNode } from "./types/index";

const config: SiteConfig = {
  vaultName: "TestVault",
  pageTitle: "Test Vault",
  notes: 3,
};

const notes: NoteListItem[] = [
  {
    slug: "index",
    title: "Index",
    tags: ["home"],
    excerpt: "Welcome to the vault",
    modTime: "2026-01-01",
    path: "index.md",
  },
  {
    slug: "research/notes",
    title: "Research Notes",
    tags: ["research"],
    excerpt: "Some research notes",
    modTime: "2026-02-01",
    path: "research/notes.md",
  },
  {
    slug: "projects/my-project",
    title: "My Project",
    tags: ["project"],
    excerpt: "A cool project",
    modTime: "2026-03-01",
    path: "projects/my-project.md",
  },
];

const tree: FileNode = {
  name: "root",
  path: "",
  isFolder: true,
  children: [
    { name: "index", slug: "index", path: "index.md", isFolder: false },
    {
      name: "research",
      path: "research",
      isFolder: true,
      children: [
        {
          name: "notes",
          slug: "research/notes",
          path: "research/notes.md",
          isFolder: false,
        },
      ],
    },
  ],
};

const note: Note = {
  slug: "index",
  title: "Index",
  path: "index.md",
  frontmatter: {},
  tags: ["home"],
  excerpt: "Welcome to the vault",
  html: "<h2>Welcome</h2><p>This is the vault index.</p>",
  wikiLinks: [{ target: "research/notes" }],
  backlinks: ["research/notes"],
  modTime: "2026-01-01",
};

describe("parseRoute", () => {
  it("parses home route", () => {
    expect(parseRoute("/")).toEqual({ type: "home" });
  });

  it("parses note route with slug", () => {
    expect(parseRoute("/note/index")).toEqual({
      type: "note",
      slug: "index",
    });
  });

  it("parses note route with nested slug", () => {
    expect(parseRoute("/note/research/notes")).toEqual({
      type: "note",
      slug: "research/notes",
    });
  });

  it("parses search route", () => {
    expect(parseRoute("/search")).toEqual({ type: "search" });
  });

  it("strips hash and query before parsing", () => {
    expect(parseRoute("/note/index?x=1#heading")).toEqual({
      type: "note",
      slug: "index",
    });
  });

  it("returns unknown for unrecognized paths", () => {
    expect(parseRoute("/something/else")).toEqual({ type: "unknown" });
  });
});

describe("renderApp", () => {
  it("renders the home page with note list", async () => {
    const { html, preloadedState } = await renderApp("/", {
      config,
      notes,
      tree,
    });

    expect(html).toContain("Test Vault");
    expect(html).toContain("Index");
    expect(html).toContain("Research Notes");
    expect(JSON.stringify(preloadedState)).toContain("listNotes");
  });

  it("renders a note page with title and body", async () => {
    const { html, preloadedState } = await renderApp("/note/index", {
      config,
      notes,
      tree,
      note,
    });

    expect(html).toContain("Index");
    expect(html).toContain("Welcome");
    expect(html).toContain("This is the vault index");
    expect(html).toContain("research/notes");
    expect(JSON.stringify(preloadedState)).toContain("getNote");
  });

  it("renders note not found when note is null", async () => {
    const { html } = await renderApp("/note/nonexistent", {
      config,
      notes,
      tree,
      note: null,
    });

    expect(html).toContain("Note not found");
  });

  it("renders search page placeholder", async () => {
    const { html } = await renderApp("/search", {
      config,
      notes,
      tree,
    });

    expect(html).toContain("Search");
    expect(html).toContain("TestVault");
  });

  it("returns preloadedState with vaultApi cache", async () => {
    const { preloadedState } = await renderApp("/", {
      config,
      notes,
      tree,
    });

    const state = JSON.stringify(preloadedState);
    expect(state).toContain("getConfig");
    expect(state).toContain("listNotes");
    expect(state).toContain("getTree");
  });
});
