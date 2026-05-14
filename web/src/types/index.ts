// Central type definitions for the vault system.
// vaultApi.ts imports from here; components import from here too.

export interface WikiLinkRef {
  target: string;
  alias?: string;
  isEmbed?: boolean;
  heading?: string;
}

export interface NoteListItem {
  slug: string;
  title: string;
  tags: string[];
  excerpt: string;
  modTime: string;
  path: string;
}

export interface Note {
  slug: string;
  title: string;
  path: string;
  frontmatter: Record<string, unknown>;
  tags: string[];
  excerpt: string;
  html: string;
  wikiLinks: WikiLinkRef[];
  backlinks: string[];
  modTime: string;
}

export interface FileNode {
  name: string;
  slug?: string;
  path: string;
  isFolder: boolean;
  children?: FileNode[];
}

export interface SearchResult {
  slug: string;
  title: string;
  excerpt: string;
  tags: string[];
  score: number;
}

export interface TagCount {
  tag: string;
  count: number;
}

