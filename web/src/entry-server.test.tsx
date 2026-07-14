import { describe, expect, it } from "vitest";
import { extractNoteSlug, renderApp } from "./entry-server";
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

describe("extractNoteSlug", () => {
  it("returns undefined for home route", () => {
    expect(extractNoteSlug("/")).toBeUndefined();
  });

  it("extracts note route slug", () => {
    expect(extractNoteSlug("/note/index")).toBe("index");
  });

  it("extracts nested note route slug", () => {
    expect(extractNoteSlug("/note/research/notes")).toBe("research/notes");
  });

  it("returns undefined for search route", () => {
    expect(extractNoteSlug("/search")).toBeUndefined();
  });

  it("strips hash and query before extracting", () => {
    expect(extractNoteSlug("/note/index?x=1#heading")).toBe("index");
  });

  it("returns undefined for unrecognized paths", () => {
    expect(extractNoteSlug("/something/else")).toBeUndefined();
  });
});

describe("renderApp", () => {
  it("renders the home page with note list", async () => {
    const { html, preloadedState } = await renderApp("/", {
      config,
      notes,
      tree,
      note,
    });

    expect(html).toContain("TestVault");
    expect(html).toContain("Index");
    expect(html).toContain("Welcome");
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
    expect(html).toContain("Research Notes");
    expect(JSON.stringify(preloadedState)).toContain("getNote");
  });

  it("does not require the full notes list to SSR a note page", async () => {
    const { html, preloadedState } = await renderApp("/note/index", {
      config,
      tree,
      note,
    });
    const state = preloadedState as {
      vaultApi: {
        queries: Record<string, { data?: unknown }>;
      };
    };

    expect(html).toContain("This is the vault index");
    expect(
      state.vaultApi.queries["listNotes(undefined)"]?.data
    ).toBeUndefined();
    expect(state.vaultApi.queries['getNote("index")']?.data).toEqual(note);
  });

  it("renders loading state if a note route is rendered without a preloaded note", async () => {
    const { html } = await renderApp("/note/nonexistent", {
      config,
      notes,
      tree,
      note: null,
    });

    expect(html).toContain("Loading note");
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
