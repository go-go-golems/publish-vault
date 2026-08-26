// Central type definitions for the vault system.
// vaultApi.ts imports from here; components import from here too.

import type { NoteDates } from "../search/noteDate";

export type { NoteDates };

export interface SiteConfig {
  vaultName: string;
  pageTitle: string;
  notes: number;
}

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
  dates?: NoteDates;
  rawMarkdown?: string;
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
  path: string;
  score: number;
  date?: SearchResultDate;
}

export interface SearchResultDate {
  value: string;
  kind: string;
  precision: string;
}

export type TagMode = "all" | "any";

export type DateField = "display" | "created" | "updated";

export type SearchSort = "relevance" | "newest" | "oldest";

export interface DateOnly {
  year: number;
  month: number;
  day: number;
}

export interface FieldError {
  field: string;
  code: string;
  message: string;
}

export interface SearchRequest {
  query: string;
  tags: string[];
  tagMode: TagMode | "";
  pathPrefixes: string[];
  dateField: DateField | "";
  dateFrom?: DateOnly;
  dateTo?: DateOnly;
  sort: SearchSort | "";
  limit: number;
  offset: number;
}

export interface SearchResponse {
  results: SearchResult[];
  total: number;
  limit: number;
  offset: number;
  sort: SearchSort;
}

export interface TagCount {
  tag: string;
  count: number;
}

