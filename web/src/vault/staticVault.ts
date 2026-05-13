/**
 * staticVault.ts
 * ─────────────────────────────────────────────────────────────────
 * Imports all demo Markdown files via Vite's ?raw loader, parses
 * frontmatter (js-yaml), renders HTML (marked), extracts wiki-links,
 * builds the backlink graph, file tree, and search index — all in-browser.
 *
 * This module is the "static backend" used when VITE_API_URL is not set.
 */

import { load as yamlLoad } from "js-yaml";
import { marked } from "marked";
import type {
  Note,
  NoteListItem,
  FileNode,
  GraphData,
  SearchResult,
  TagCount,
} from "../types";

// ── Marked wiki-link extension ───────────────────────────────────
// We register a custom inline token so marked never sees raw HTML —
// the renderer emits the <a> tag directly from the token.
//
// The extension is registered lazily (inside buildVault) so it has
// access to the allSlugs set.
function makeWikiLinkExtension(allSlugs: Set<string>) {
  return {
    name: "wikiLink",
    level: "inline" as const,
    start(src: string) {
      return src.indexOf("[[");
    },
    tokenizer(src: string) {
      const match = src.match(/^\[\[([^\]]+)\]\]/);
      if (!match) return undefined;
      return {
        type: "wikiLink",
        raw: match[0],
        inner: match[1],
      };
    },
    renderer(token: { inner: string }) {
      const inner = token.inner;
      const alias = inner.includes("|")
        ? inner.split("|")[1].trim()
        : inner.split("#")[0].split("|")[0].trim();
      const slug = resolveWikiTarget(inner, allSlugs);
      const isBroken = !allSlugs.has(slug);
      const cls = isBroken ? "wiki-link broken" : "wiki-link";
      const href = isBroken ? "#" : `/note/${slug}`;
      return `<a class="${cls}" data-target="${slug}" href="${href}">${alias}</a>`;
    },
  };
}

// ── Import all vault notes as raw strings ─────────────────────────
// Vite's import.meta.glob with { query: "?raw" } loads file contents as strings.
const rawFiles = import.meta.glob("./notes/**/*.md", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

// ── Frontmatter parser ────────────────────────────────────────────

/** Parse YAML frontmatter from a raw Markdown string. */
function parseFrontmatter(raw: string): {
  data: Record<string, unknown>;
  content: string;
} {
  const FENCE = /^---\r?\n([\s\S]*?)\r?\n---\r?\n?/;
  const m = raw.match(FENCE);
  if (!m) return { data: {}, content: raw };
  try {
    const data = (yamlLoad(m[1]) ?? {}) as Record<string, unknown>;
    return { data, content: raw.slice(m[0].length) };
  } catch {
    return { data: {}, content: raw.slice(m[0].length) };
  }
}

// ── Slug helpers ──────────────────────────────────────────────────

/** Convert a file path like "./notes/Philosophy/Stoicism.md" → "philosophy/stoicism" */
function pathToSlug(path: string): string {
  return path
    .replace(/^\.\/notes\//, "")
    .replace(/\.md$/, "")
    .toLowerCase()
    .replace(/\s+/g, "-");
}

/** Convert a note title to slug: "Zeno of Citium" → "zeno-of-citium" */
function titleToSlug(title: string): string {
  return title.toLowerCase().replace(/\s+/g, "-").replace(/[^a-z0-9-/]/g, "");
}

/** Resolve a wiki-link target to a slug.
 *  Handles: bare title, folder/title, title with alias, title with heading. */
function resolveWikiTarget(raw: string, allSlugs: Set<string>): string {
  // Strip alias: [[Target|Alias]] → "Target"
  const withoutAlias = raw.split("|")[0].trim();
  // Strip heading: [[Target#Heading]] → "Target"
  const withoutHeading = withoutAlias.split("#")[0].trim();

  // Try exact slug match
  const direct = titleToSlug(withoutHeading);
  if (allSlugs.has(direct)) return direct;

  // Try with folder prefix search
  for (const slug of Array.from(allSlugs)) {
    const parts = slug.split("/");
    const last = parts[parts.length - 1];
    if (last === direct) return slug;
  }

  return direct; // may be broken
}

// ── Wiki-link regex ───────────────────────────────────────────────
const WIKI_LINK_RE = /\[\[([^\]]+)\]\]/g;

/** Extract all wiki-link targets from raw Markdown content */
function extractWikiLinks(content: string): string[] {
  const targets: string[] = [];
  let m: RegExpExecArray | null;
  WIKI_LINK_RE.lastIndex = 0;
  while ((m = WIKI_LINK_RE.exec(content)) !== null) {
    targets.push(m[1]);
  }
  return targets;
}

/** Replace [[wiki links]] with <a> tags in Markdown before rendering.
 *  marked is configured with { gfm: true } which passes raw HTML through. */
function preprocessWikiLinks(content: string, allSlugs: Set<string>): string {
  return content.replace(/\[\[([^\]]+)\]\]/g, (_match, inner) => {
    const alias = inner.includes("|")
      ? inner.split("|")[1].trim()
      : inner.split("#")[0].split("|")[0].trim();
    const slug = resolveWikiTarget(inner, allSlugs);
    const isBroken = !allSlugs.has(slug);
    const cls = isBroken ? "wiki-link broken" : "wiki-link";
    const href = isBroken ? "#" : `/note/${slug}`;
    return `<a class="${cls}" data-target="${slug}" href="${href}">${alias}</a>`;
  });
}

// ── Serialization helper ─────────────────────────────────────────

/**
 * Recursively convert any Date objects in a frontmatter record to ISO strings.
 * This prevents Redux's "non-serializable value" warning.
 */
function serializeFrontmatter(obj: Record<string, unknown>): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(obj)) {
    if (v instanceof Date) {
      out[k] = v.toISOString().slice(0, 10); // "2024-01-15"
    } else if (Array.isArray(v)) {
      out[k] = v.map((item) => (item instanceof Date ? item.toISOString().slice(0, 10) : item));
    } else if (v !== null && typeof v === "object") {
      out[k] = serializeFrontmatter(v as Record<string, unknown>);
    } else {
      out[k] = v;
    }
  }
  return out;
}

// ── Build the vault ───────────────────────────────────────────────

interface RawNote {
  path: string;
  slug: string;
  title: string;
  tags: string[];
  frontmatter: Record<string, unknown>;
  content: string; // raw Markdown without frontmatter
  modTime: string;
}

function buildVault(): {
  notes: Map<string, Note>;
  list: NoteListItem[];
  tree: FileNode;
  graph: GraphData;
  tagCounts: TagCount[];
} {
  // ── Parse all files ──────────────────────────────────────────
  const rawNotes: RawNote[] = [];

  for (const [filePath, raw] of Object.entries(rawFiles)) {
    const parsed = parseFrontmatter(raw as string);
    const fm = parsed.data;
    const slug = pathToSlug(filePath);
    const rawTitle = (fm.title as string) || slug.split("/").pop()!.replace(/-/g, " ");
    // Capitalise first letter of each word for display
    const title = rawTitle.replace(/\b\w/g, (c) => c.toUpperCase());
    const tags = Array.isArray(fm.tags)
      ? (fm.tags as string[])
      : typeof fm.tags === "string"
      ? [fm.tags]
      : [];

    // Serialize frontmatter to remove non-serializable Date objects
    const safeFm = serializeFrontmatter(fm);
    // Extract modTime: if fm.created is a Date, use its ISO form; otherwise coerce
    let modTime: string;
    if (fm.created instanceof Date) {
      modTime = fm.created.toISOString().slice(0, 10);
    } else if (typeof fm.created === "string") {
      modTime = fm.created.slice(0, 10);
    } else {
      modTime = new Date().toISOString().slice(0, 10);
    }

    rawNotes.push({
      path: filePath.replace(/^\.\/notes\//, ""),
      slug,
      title,
      tags,
      frontmatter: safeFm,
      content: parsed.content,
      modTime,
    });
  }

  // Sort: index first, then alphabetical
  rawNotes.sort((a, b) => {
    if (a.slug === "index") return -1;
    if (b.slug === "index") return 1;
    return a.title.localeCompare(b.title);
  });

  const allSlugs = new Set(rawNotes.map((n) => n.slug));

  // Register the wiki-link extension with the current slug set
  marked.use({ gfm: true, breaks: false, extensions: [makeWikiLinkExtension(allSlugs)] });

  // ── First pass: collect wiki-link graph ──────────────────────
  const wikiLinkMap = new Map<string, string[]>(); // slug → [unique target slugs]
  for (const rn of rawNotes) {
    const rawTargets = extractWikiLinks(rn.content);
    const resolved = rawTargets.map((t) => resolveWikiTarget(t, allSlugs));
    // Deduplicate: one note with [[X]] three times still counts as one outgoing link
    const unique = Array.from(new Set(resolved.filter(Boolean)));
    wikiLinkMap.set(rn.slug, unique);
  }

  // Build backlinks: for each note, which notes link to it?
  // Because wikiLinkMap already deduplicates per source, each source appears at most once.
  const backlinkMap = new Map<string, string[]>();
  for (const [sourceSlug, targets] of Array.from(wikiLinkMap.entries())) {
    for (const target of targets) {
      if (!backlinkMap.has(target)) backlinkMap.set(target, []);
      backlinkMap.get(target)!.push(sourceSlug);
    }
  }

  // ── Second pass: render HTML ─────────────────────────────────
  const notes = new Map<string, Note>();

  for (const rn of rawNotes) {
    // No preprocessing needed — the marked extension handles [[wiki links]]
    const html = marked.parse(rn.content) as string;

    // Excerpt: first 200 chars of plain text
    const plainText = rn.content
      .replace(/#+\s/g, "")
      .replace(/\[\[([^\]|]+)(?:\|[^\]]+)?\]\]/g, "$1")
      .replace(/[*_`~]/g, "")
      .trim();
    const excerpt = plainText.slice(0, 200) + (plainText.length > 200 ? "…" : "");

    notes.set(rn.slug, {
      slug: rn.slug,
      title: rn.title,
      path: rn.path,
      frontmatter: rn.frontmatter,
      tags: rn.tags,
      excerpt,
      html,
      wikiLinks: (wikiLinkMap.get(rn.slug) ?? []).map((t) => ({ target: t })),
      backlinks: backlinkMap.get(rn.slug) ?? [],
      modTime: rn.modTime,
    });
  }

  // ── Build list ───────────────────────────────────────────────
  const list: NoteListItem[] = rawNotes.map((rn) => ({
    slug: rn.slug,
    title: rn.title,
    tags: rn.tags,
    excerpt: notes.get(rn.slug)!.excerpt,
    modTime: rn.modTime,
    path: rn.path,
  }));

  // ── Build file tree ──────────────────────────────────────────
  const root: FileNode = {
    name: "Vault",
    path: "",
    isFolder: true,
    children: [],
  };

  for (const rn of rawNotes) {
    const parts = rn.path.replace(/\.md$/, "").split("/");
    let node = root;

    if (parts.length === 1) {
      node.children!.push({
        name: rn.title,
        slug: rn.slug,
        path: rn.path,
        isFolder: false,
      });
    } else {
      for (let i = 0; i < parts.length - 1; i++) {
        const folderName = parts[i];
        let folder = node.children!.find(
          (c) => c.isFolder && c.name === folderName
        );
        if (!folder) {
          folder = {
            name: folderName,
            path: parts.slice(0, i + 1).join("/"),
            isFolder: true,
            children: [],
          };
          node.children!.push(folder);
        }
        node = folder;
      }
      node.children!.push({
        name: rn.title,
        slug: rn.slug,
        path: rn.path,
        isFolder: false,
      });
    }
  }

  // Sort tree: folders first, then files, alphabetically
  function sortTree(n: FileNode) {
    if (!n.children) return;
    n.children.sort((a, b) => {
      if (a.isFolder && !b.isFolder) return -1;
      if (!a.isFolder && b.isFolder) return 1;
      return a.name.localeCompare(b.name);
    });
    n.children.forEach(sortTree);
  }
  sortTree(root);

  // ── Build graph ──────────────────────────────────────────────
  const graphNodes = rawNotes.map((rn) => ({
    id: rn.slug,
    title: rn.title,
    tags: rn.tags,
  }));

  const edgeSet = new Set<string>();
  const graphEdges: { source: string; target: string }[] = [];
  for (const [source, targets] of Array.from(wikiLinkMap.entries())) {
    for (const target of targets) {
      if (!allSlugs.has(target)) continue;
      const key = [source, target].sort().join("||");
      if (!edgeSet.has(key)) {
        edgeSet.add(key);
        graphEdges.push({ source, target });
      }
    }
  }

  const graph: GraphData = { nodes: graphNodes, edges: graphEdges };

  // ── Build tag counts ─────────────────────────────────────────
  const tagMap = new Map<string, number>();
  for (const rn of rawNotes) {
    for (const tag of rn.tags) {
      tagMap.set(tag, (tagMap.get(tag) ?? 0) + 1);
    }
  }
  const tagCounts: TagCount[] = Array.from(tagMap.entries())
    .map(([tag, count]) => ({ tag, count }))
    .sort((a, b) => b.count - a.count);

  return { notes, list, tree: root, graph, tagCounts };
}

// ── Singleton vault instance ──────────────────────────────────────
let _vault: ReturnType<typeof buildVault> | null = null;

function getVault() {
  if (!_vault) _vault = buildVault();
  return _vault;
}

// ── Public API ────────────────────────────────────────────────────

export function staticListNotes(): NoteListItem[] {
  return getVault().list;
}

export function staticGetNote(slug: string): Note | null {
  const vault = getVault();
  const note = vault.notes.get(slug);
  console.log('[staticGetNote] slug:', JSON.stringify(slug), '| found:', !!note, '| available slugs:', Array.from(vault.notes.keys()).join(', '));
  return note ?? null;
}

export function staticGetTree(): FileNode {
  return getVault().tree;
}

export function staticGetGraph(): GraphData {
  return getVault().graph;
}

export function staticListTags(): TagCount[] {
  return getVault().tagCounts;
}

export function staticSearch(query: string): SearchResult[] {
  if (!query.trim()) return [];
  const q = query.toLowerCase();
  const vault = getVault();
  const results: SearchResult[] = [];

  for (const note of Array.from(vault.notes.values())) {
    const titleScore = note.title.toLowerCase().includes(q) ? 2 : 0;
    const tagScore = note.tags.some((t: string) => t.toLowerCase().includes(q))
      ? 1.5
      : 0;
    const contentScore = note.excerpt.toLowerCase().includes(q) ? 1 : 0;
    const score = titleScore + tagScore + contentScore;
    if (score > 0) {
      results.push({
        slug: note.slug,
        title: note.title,
        excerpt: note.excerpt,
        tags: note.tags,
        score,
      });
    }
  }

  return results.sort((a, b) => b.score - a.score);
}

/** Return the first note slug (used as default landing note) */
export function staticGetDefaultSlug(): string {
  const list = staticListNotes();
  const idx = list.find((n) => n.slug === "index");
  return idx?.slug ?? list[0]?.slug ?? "";
}
