/**
 * staticVault.ts
 * ─────────────────────────────────────────────────────────────────
 * Imports all demo Markdown files via Vite's ?raw loader, parses
 * frontmatter (js-yaml), renders HTML (marked), extracts wiki-links,
 * builds backlinks, file tree, and search index — all in-browser.
 *
 * This module is the "static backend" used when VITE_API_URL is not set.
 */

import { load as yamlLoad, JSON_SCHEMA } from "js-yaml";
import { marked } from "marked";
import type {
  Note,
  NoteListItem,
  FileNode,
  SearchResult,
  TagCount,
} from "../types";
import type { SiteConfig } from "../store/vaultApi";
import { resolveNoteDates, apiValue, display as displayDate } from "../search/noteDate";
import {
  normalizeSearchRequest,
  isEffective,
  dateOnlyToInstant,
  dateOnlyNextDayInstant,
} from "../search/searchParams";
import type {
  DateField,
  SearchRequest,
  SearchResponse,
  SearchResultDate,
  SearchSort,
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
      const alias = wikiLinkLabel(inner);
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

/** Parse YAML frontmatter from a raw Markdown string.
 *
 * Uses js-yaml JSON_SCHEMA so unquoted date and RFC3339 scalars are preserved
 * as strings rather than resolved to JavaScript Date objects. The default
 * YAML schema would turn `updated: 2024-01-15T13:45:00-05:00` into a Date,
 * and serializeFrontmatter would then truncate it to YYYY-MM-DD, losing the
 * instant and timestamp precision before authored-date resolution runs. */
export function parseFrontmatter(raw: string): {
  data: Record<string, unknown>;
  content: string;
} {
  const FENCE = /^---\r?\n([\s\S]*?)\r?\n---\r?\n?/;
  const m = raw.match(FENCE);
  if (!m) return { data: {}, content: raw };
  try {
    const data = (yamlLoad(m[1], { schema: JSON_SCHEMA }) ?? {}) as Record<
      string,
      unknown
    >;
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

/** Strip a trailing ".md" from a wiki-link target.
 *  Obsidian treats [[Note]] and [[Note.md]] as the same link, and pathToSlug
 *  builds every slug without the extension — while titleToSlug *deletes* the dot
 *  rather than keeping it, so an unstripped "foo.md" would look for "foomd".
 *  Only ".md" is stripped (the vault globs nothing else), never down to "". */
function stripNoteExtension(target: string): string {
  return target.length > 3 && target.slice(-3).toLowerCase() === ".md"
    ? target.slice(0, -3)
    : target;
}

/** Display text for a wiki link with no explicit alias: the target as written,
 *  minus the heading and the .md extension.
 *
 *  [[#Heading]] has no target at all, so it falls back to the heading — without
 *  it the anchor renders with empty text and is invisible on the page. It still
 *  renders as broken here: marked (v18) emits no heading ids, so the static
 *  build has nothing for a fragment to point at. The Go renderer resolves these
 *  properly; see resolveSelfHeadingLinks in internal/parser/parser.go. */
function wikiLinkLabel(inner: string): string {
  if (inner.includes("|")) return inner.split("|")[1].trim();
  const beforeHeading = inner.split("#")[0].split("|")[0].trim();
  if (beforeHeading === "") return inner.split("|")[0].replace(/^#/, "").trim();
  return stripNoteExtension(beforeHeading);
}

/** Resolve a wiki-link target to a slug.
 *  Handles: bare title, folder/title, title with alias, title with heading. */
function resolveWikiTarget(raw: string, allSlugs: Set<string>): string {
  // Strip alias: [[Target|Alias]] → "Target"
  const withoutAlias = raw.split("|")[0].trim();
  // Strip heading: [[Target#Heading]] → "Target"
  const withoutHeading = stripNoteExtension(withoutAlias.split("#")[0].trim());

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
  return content.replace(/\[\[([^\]]+)\]\]/g, (_match, inner: string) => {
    const alias = wikiLinkLabel(inner);
    const slug = resolveWikiTarget(inner, allSlugs);
    const isBroken = !allSlugs.has(slug);
    const cls = isBroken ? "wiki-link broken" : "wiki-link";
    const href = isBroken ? "#" : `/note/${slug}`;
    return `<a class="${cls}" data-target="${slug}" href="${href}">${alias}</a>`;
  });
}

// ── Serialization helper ─────────────────────────────────────────

/** Recursively convert any Date objects in a frontmatter record to ISO strings.
 * This prevents Redux's "non-serializable value" warning. */
export function serializeFrontmatter(
  obj: Record<string, unknown>,
): Record<string, unknown> {
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
  content: string; // Markdown body without frontmatter, used for rendering.
  rawMarkdown: string; // Original Markdown source, used by static-mode copy/download actions.
  modTime: string;
}

export function buildVaultFromRaw(rawFiles: Record<string, string>): {
  notes: Map<string, Note>;
  list: NoteListItem[];
  tree: FileNode;
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
      rawMarkdown: raw as string,
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
      dates: resolveNoteDates(rn.frontmatter).dates,
      rawMarkdown: rn.rawMarkdown,
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

  return { notes, list, tree: root, tagCounts };
}

// ── Singleton vault instance ──────────────────────────────────────
let _vault: ReturnType<typeof buildVaultFromRaw> | null = null;

function getVault() {
  if (!_vault) _vault = buildVaultFromRaw(rawFiles);
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

export function staticListTags(): TagCount[] {
  return getVault().tagCounts;
}

function extractStaticTagQuery(query: string): string | null {
  const trimmed = query.trim();
  if (trimmed.startsWith("#")) {
    return trimmed.slice(1).trim().toLowerCase();
  }
  if (trimmed.toLowerCase().startsWith("tag:")) {
    return trimmed.slice(4).trim().toLowerCase();
  }
  return null;
}

export function staticSearch(query: string): SearchResult[] {
  if (!query.trim()) return [];
  const q = query.toLowerCase();
  const tagQuery = extractStaticTagQuery(query);
  const vault = getVault();
  const results: SearchResult[] = [];

  for (const note of Array.from(vault.notes.values())) {
    const normalizedTags = note.tags.map((t: string) => t.toLowerCase());
    const titleScore = tagQuery ? 0 : note.title.toLowerCase().includes(q) ? 2 : 0;
    const tagScore = tagQuery
      ? normalizedTags.includes(tagQuery)
        ? 3
        : 0
      : normalizedTags.some((t: string) => t.includes(q))
        ? 1.5
        : 0;
    const contentScore = tagQuery ? 0 : note.excerpt.toLowerCase().includes(q) ? 1 : 0;
    const score = titleScore + tagScore + contentScore;
    if (score > 0) {
      results.push({
        slug: note.slug,
        title: note.title,
        excerpt: note.excerpt,
        tags: note.tags,
        path: note.path,
        score,
      });
    }
  }

  return results.sort((a, b) => b.score - a.score);
}

// ── Advanced static search (PV-SEARCH-028) ────────────────────────

function levenshtein(a: string, b: string): number {
  if (a === b) return 0;
  if (a.length === 0) return b.length;
  if (b.length === 0) return a.length;
  let prev = new Array(b.length + 1);
  let curr = new Array(b.length + 1);
  for (let j = 0; j <= b.length; j++) prev[j] = j;
  for (let i = 1; i <= a.length; i++) {
    curr[0] = i;
    for (let j = 1; j <= b.length; j++) {
      const cost = a[i - 1] === b[j - 1] ? 0 : 1;
      curr[j] = Math.min(prev[j] + 1, curr[j - 1] + 1, prev[j - 1] + cost);
    }
    [prev, curr] = [curr, prev];
  }
  return prev[b.length];
}

/** Pinned legacy #tag inclusion contract: prefix for queries of at most three
 * characters, otherwise exact or edit-distance-one over normalized tags. This
 * mirrors the backend's deployed dynamic behavior so static and dynamic modes
 * include the same notes. */
function legacyTagMatches(normalizedTags: string[], tagQuery: string): boolean {
  if (tagQuery.length <= 3) {
    return normalizedTags.some((t) => t.startsWith(tagQuery));
  }
  return normalizedTags.some((t) => t === tagQuery || levenshtein(t, tagQuery) <= 1);
}

function noteTextMatches(note: Note, q: string): boolean {
  return (
    note.title.toLowerCase().includes(q) ||
    note.tags.some((t) => t.toLowerCase().includes(q)) ||
    note.excerpt.toLowerCase().includes(q)
  );
}

function textScore(note: Note, q: string, tagQuery: string | null): number {
  if (tagQuery) {
    return legacyTagMatches(note.tags.map((t) => t.toLowerCase()), tagQuery) ? 3 : 0;
  }
  const title = note.title.toLowerCase().includes(q) ? 2 : 0;
  const tag = note.tags.some((t) => t.toLowerCase().includes(q)) ? 1.5 : 0;
  const content = note.excerpt.toLowerCase().includes(q) ? 1 : 0;
  return title + tag + content;
}

function dateInstantForField(note: Note, field: DateField | ""): Date | null {
  if (!note.dates) return null;
  if (field === "created") return note.dates.created?.value ?? null;
  if (field === "updated") return note.dates.updated?.value ?? null;
  const d = displayDate(note.dates).date;
  return d ? d.value : null;
}

function displayInstant(note: Note): Date | null {
  return dateInstantForField(note, "display");
}

function noteMatchesAdvanced(note: Note, req: SearchRequest, tagQuery: string | null, qLower: string): boolean {
  if (req.query) {
    if (tagQuery) {
      if (!legacyTagMatches(note.tags.map((t) => t.toLowerCase()), tagQuery)) return false;
    } else if (!noteTextMatches(note, qLower)) {
      return false;
    }
  }
  if (req.tags.length > 0) {
    const normTags = note.tags.map((t) => t.toLowerCase());
    const match = req.tagMode === "any"
      ? req.tags.some((t) => normTags.includes(t))
      : req.tags.every((t) => normTags.includes(t));
    if (!match) return false;
  }
  if (req.pathPrefixes.length > 0) {
    const normPath = note.path.toLowerCase().replace(/^\.\//, "").replace(/^\//, "");
    if (!req.pathPrefixes.some((p) => normPath.startsWith(p))) return false;
  }
  if (req.dateFrom || req.dateTo) {
    const instant = dateInstantForField(note, req.dateField);
    if (instant === null) return false;
    const t = instant.getTime();
    if (req.dateFrom && t < dateOnlyToInstant(req.dateFrom).getTime()) return false;
    if (req.dateTo && t >= dateOnlyNextDayInstant(req.dateTo).getTime()) return false;
  }
  return true;
}

/** staticSearchAdvanced reproduces the backend advanced-search inclusion and
 * ordering contract in the browser: exact tag all/any, path prefixes, date
 * ranges, deterministic sorts, pagination, and a total count. It does not
 * promise Bleve score parity. */
export function staticSearchAdvanced(req: SearchRequest): SearchResponse {
  return searchAdvancedInNotes(getVault().notes, req);
}

/** searchAdvancedInNotes runs the static advanced-search contract against an
 * arbitrary notes map, so tests can feed controlled fixtures without the
 * singleton demo vault. */
export function searchAdvancedInNotes(notes: Map<string, Note>, req: SearchRequest): SearchResponse {
  const { request } = normalizeSearchRequest(req);
  const all = Array.from(notes.values());
  if (!isEffective(request)) {
    return { results: [], total: 0, limit: request.limit, offset: request.offset, sort: request.sort as SearchSort };
  }
  const qLower = request.query.toLowerCase();
  const tagQuery = extractStaticTagQuery(request.query);
  const scored = all
    .filter((note) => noteMatchesAdvanced(note, request, tagQuery, qLower))
    .map((note) => ({ note, score: textScore(note, qLower, tagQuery) }));

  if (request.sort === "newest" || request.sort === "oldest") {
    scored.sort((a, b) => {
      const da = displayInstant(a.note);
      const db = displayInstant(b.note);
      if (da === null && db === null) return a.note.slug.localeCompare(b.note.slug);
      if (da === null) return 1;
      if (db === null) return -1;
      const cmp = da.getTime() - db.getTime();
      if (cmp !== 0) return request.sort === "newest" ? -cmp : cmp;
      return a.note.slug.localeCompare(b.note.slug);
    });
  } else {
    scored.sort((a, b) => {
      if (a.score !== b.score) return b.score - a.score;
      return a.note.slug.localeCompare(b.note.slug);
    });
  }

  const total = scored.length;
  const page = scored.slice(request.offset, request.offset + request.limit);
  const results: SearchResult[] = page.map(({ note, score }) => {
    const shown = note.dates ? displayDate(note.dates) : { kind: "" as const, date: null };
    const date: SearchResultDate | undefined =
      shown.date && shown.kind
        ? { value: apiValue(shown.date) ?? "", kind: shown.kind, precision: shown.date.precision }
        : undefined;
    return {
      slug: note.slug,
      title: note.title,
      excerpt: note.excerpt,
      tags: note.tags,
      path: note.path,
      score,
      date,
    };
  });
  return { results, total, limit: request.limit, offset: request.offset, sort: request.sort as SearchSort };
}

/** Return the first note slug (used as default landing note) */
export function staticGetDefaultSlug(): string {
  const list = staticListNotes();
  const idx = list.find((n) => n.slug === "index");
  return idx?.slug ?? list[0]?.slug ?? "";
}

/** Return site config for static mode */
export function staticGetConfig(): SiteConfig {
  return {
    vaultName: "Demo Vault",
    pageTitle: "Demo Vault",
    notes: getVault().list.length,
  };
}
