/**
 * RTK Query API slice for the Obsidian vault.
 *
 * Strategy:
 *   - By default, fetch from the Go backend on the same origin via /api/*.
 *   - If VITE_API_URL is set, fetch from that explicit backend origin.
 *   - If VITE_STATIC_VAULT=true, serve data from the in-browser staticVault
 *     module for standalone demo/static deployments.
 */
import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";
import type {
  Note,
  NoteListItem,
  FileNode,
  SearchResult,
  TagCount,
  WikiLinkRef,
} from "../types";

// Re-export types so consumers can import from one place
export type { Note, NoteListItem, FileNode, SearchResult, TagCount, WikiLinkRef };

// ── Mode detection ────────────────────────────────────────────────
const API_BASE = (import.meta.env.VITE_API_URL as string | undefined) || "";
const IS_STATIC = import.meta.env.VITE_STATIC_VAULT === "true";

// Lazy-load the static vault only in demo mode (tree-shaken in backend mode)
async function getStatic() {
  const m = await import("../vault/staticVault");
  return m;
}

// ── RTK Query helpers ─────────────────────────────────────────────

type QR<T> = Promise<{ data: T } | { error: FetchBaseQueryError }>;

function ok<T>(data: T): { data: T } {
  return { data };
}

function notFound(): { error: FetchBaseQueryError } {
  return { error: { status: 404, data: "Not found" } as FetchBaseQueryError };
}

// ── API Slice ─────────────────────────────────────────────────────

export const vaultApi = createApi({
  reducerPath: "vaultApi",
  // baseQuery is used only in backend mode; static mode uses queryFn
  baseQuery: fetchBaseQuery({ baseUrl: API_BASE }),
  tagTypes: ["Note", "Notes", "Tree", "Tags"],
  endpoints: (builder) => ({

    // ── List all notes ──────────────────────────────────────────
    listNotes: builder.query<NoteListItem[], void>(
      IS_STATIC
        ? {
            queryFn: async (): Promise<{ data: NoteListItem[] }> => {
              const s = await getStatic();
              return ok(s.staticListNotes());
            },
            providesTags: ["Notes"] as const,
          }
        : {
            query: () => "/api/notes",
            providesTags: ["Notes"] as const,
          }
    ),

    // ── Get a single note ───────────────────────────────────────
    getNote: builder.query<Note, string>(
      IS_STATIC
        ? {
            queryFn: async (slug): Promise<{ data: Note } | { error: FetchBaseQueryError }> => {
              const s = await getStatic();
              const note = s.staticGetNote(slug);
              if (!note) return notFound();
              return ok(note);
            },
            providesTags: (_r: unknown, _e: unknown, slug: string) => [{ type: "Note" as const, id: slug }],
          }
        : {
            query: (slug: string) => `/api/notes/${slug}`,
            providesTags: (_r: unknown, _e: unknown, slug: string) => [{ type: "Note" as const, id: slug }],
          }
    ),

    // ── File tree ───────────────────────────────────────────────
    getTree: builder.query<FileNode, void>(
      IS_STATIC
        ? {
            queryFn: async (): Promise<{ data: FileNode }> => {
              const s = await getStatic();
              return ok(s.staticGetTree());
            },
            providesTags: ["Tree"] as const,
          }
        : {
            query: () => "/api/tree",
            providesTags: ["Tree"] as const,
          }
    ),

    // ── Full-text search ────────────────────────────────────────
    search: builder.query<SearchResult[], string>(
      IS_STATIC
        ? {
            queryFn: async (q: string): Promise<{ data: SearchResult[] }> => {
              const s = await getStatic();
              return ok(s.staticSearch(q));
            },
          }
        : { query: (q: string) => `/api/search?q=${encodeURIComponent(q)}` }
    ),

    // ── Tags ────────────────────────────────────────────────────
    listTags: builder.query<TagCount[], void>(
      IS_STATIC
        ? {
            queryFn: async (): Promise<{ data: TagCount[] }> => {
              const s = await getStatic();
              return ok(s.staticListTags());
            },
            providesTags: ["Tags"] as const,
          }
        : {
            query: () => "/api/tags",
            providesTags: ["Tags"] as const,
          }
    ),
  }),
});

export const {
  useListNotesQuery,
  useGetNoteQuery,
  useGetTreeQuery,
  useSearchQuery,
  useListTagsQuery,
} = vaultApi;
