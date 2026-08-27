/**
 * Pure canonical URL codec and request normalization for advanced search,
 * mirroring the Go backend contract (pkg/search/request.go). The browser URL is
 * the committed search request state; these functions keep it canonical and
 * shareable so another browser reconstructs the same request.
 *
 * Round-trip invariant: decode(encode(canonicalize(request))) === canonicalize(request).
 */
import type {
  DateField,
  DateOnly,
  FieldError,
  SearchRequest,
  SearchSort,
  TagMode,
} from "../types";
import { resolveNoteDates, toUTCSecond } from "./noteDate";

export const MAX_QUERY_BYTES = 1024;
export const MAX_TAGS = 20;
export const MAX_TAG_BYTES = 128;
export const MAX_PATH_PREFIXES = 10;
export const MAX_PATH_BYTES = 512;
export const DEFAULT_LIMIT = 30;
export const MAX_LIMIT = 100;
export const MAX_OFFSET = 10000;

const PARAM_ORDER = [
  "q",
  "tag",
  "tag_mode",
  "path",
  "date_field",
  "date_from",
  "date_to",
  "sort",
  "limit",
  "offset",
] as const;

const STRICT_DATE_ONLY = /^\d{4}-\d{2}-\d{2}$/;

export function parseDateOnly(value: string): DateOnly | null {
  if (!STRICT_DATE_ONLY.test(value)) return null;
  const [y, m, d] = value.split("-").map((s) => Number.parseInt(s, 10));
  const instant = new Date(Date.UTC(y, m - 1, d));
  if (Number.isNaN(instant.getTime())) return null;
  // JavaScript's Date normalizes nonexistent calendar dates (e.g. 2024-02-30
  // becomes March 1). Reject them by round-tripping the components so an
  // invalid authored or URL date is surfaced rather than silently shifted.
  if (
    instant.getUTCFullYear() !== y ||
    instant.getUTCMonth() + 1 !== m ||
    instant.getUTCDate() !== d
  ) {
    return null;
  }
  return { year: y, month: m, day: d };
}

function dateOnlyToInstant(d: DateOnly): Date {
  return new Date(Date.UTC(d.year, d.month - 1, d.day));
}

function dateOnlyNextDayInstant(d: DateOnly): Date {
  return new Date(Date.UTC(d.year, d.month - 1, d.day + 1));
}

export function dateOnlyString(d: DateOnly): string {
  return `${String(d.year).padStart(4, "0")}-${String(d.month).padStart(
    2,
    "0",
  )}-${String(d.day).padStart(2, "0")}`;
}

function dateOnlyBefore(a: DateOnly, b: DateOnly): boolean {
  if (a.year !== b.year) return a.year < b.year;
  if (a.month !== b.month) return a.month < b.month;
  return a.day < b.day;
}

function hasControlChars(s: string): boolean {
  for (const ch of s) {
    const c = ch.codePointAt(0);
    if (c === undefined) continue;
    if (c < 0x20 || c === 0x7f) return true;
  }
  return false;
}

function normalizeTags(tags: string[]): { tags: string[]; errors: FieldError[] } {
  const errors: FieldError[] = [];
  if (tags.length > MAX_TAGS) {
    errors.push({ field: "tag", code: "too_many_tags", message: `At most ${MAX_TAGS} tags are allowed.` });
  }
  const seen = new Set<string>();
  const out: string[] = [];
  for (let t of tags) {
    t = t.trim().replace(/^#/, "").toLowerCase();
    if (t === "") {
      errors.push({ field: "tag", code: "tag_invalid", message: "Tags must not be empty." });
      continue;
    }
    if (hasControlChars(t)) {
      errors.push({ field: "tag", code: "tag_invalid", message: "Tags must not contain control characters." });
      continue;
    }
    if (t.length > MAX_TAG_BYTES) {
      errors.push({ field: "tag", code: "tag_too_long", message: `Tags must be at most ${MAX_TAG_BYTES} bytes.` });
      continue;
    }
    if (seen.has(t)) continue;
    seen.add(t);
    out.push(t);
  }
  return { tags: out, errors };
}

function collapseSlashes(p: string): string {
  return p.replace(/\/{2,}/g, "/");
}

function normalizePathPrefixes(paths: string[]): { paths: string[]; errors: FieldError[] } {
  const errors: FieldError[] = [];
  if (paths.length > MAX_PATH_PREFIXES) {
    errors.push({ field: "path", code: "too_many_paths", message: `At most ${MAX_PATH_PREFIXES} path prefixes are allowed.` });
  }
  const seen = new Set<string>();
  const out: string[] = [];
  for (let p of paths) {
    p = p.trim().toLowerCase().replace(/^\.\//, "").replace(/^\//, "");
    if (p.includes("..")) {
      errors.push({ field: "path", code: "path_invalid", message: "Path prefixes must not contain traversal segments." });
      continue;
    }
    p = collapseSlashes(p).replace(/\/$/, "");
    if (p === "") {
      errors.push({ field: "path", code: "path_invalid", message: "Path prefixes must not be empty." });
      continue;
    }
    if (p.length > MAX_PATH_BYTES) {
      errors.push({ field: "path", code: "path_too_long", message: `Path prefixes must be at most ${MAX_PATH_BYTES} bytes.` });
      continue;
    }
    p = p + "/";
    if (seen.has(p)) continue;
    seen.add(p);
    out.push(p);
  }
  return { paths: out, errors };
}

/** Normalize, validate, and apply defaults to an advanced-search request.
 *
 * The query is NOT trimmed here. Trimming the query would strip leading and
 * trailing spaces the user is actively typing into the controlled search
 * field: the field's value is derived from this normalized request, so a trim
 * here would round-trip through the URL and overwrite the user's space. The
 * search backends (the Go API tokenizer and the static-vault matcher) trim and
 * word-split on their own, so preserving spaces here does not change search
 * results. Whitespace-only queries are treated as empty by isEffective and
 * omitted from the URL by encodeSearchParams (both gate on query.trim()). */
export function normalizeSearchRequest(raw: SearchRequest): { request: SearchRequest; errors: FieldError[] } {
  const req: SearchRequest = { ...raw, tags: [...raw.tags], pathPrefixes: [...raw.pathPrefixes] };
  const errors: FieldError[] = [];

  if (req.query.length > 0 && req.query.length > MAX_QUERY_BYTES) {
    errors.push({ field: "q", code: "query_too_long", message: "Query is too long." });
  }

  const tagResult = normalizeTags(req.tags);
  req.tags = tagResult.tags;
  errors.push(...tagResult.errors);

  if (req.tagMode !== "all" && req.tagMode !== "any") {
    if (req.tagMode === "") {
      req.tagMode = "all";
    } else {
      errors.push({ field: "tag_mode", code: "tag_mode_invalid", message: "tag_mode must be all or any." });
      req.tagMode = "all";
    }
  }

  const pathResult = normalizePathPrefixes(req.pathPrefixes);
  req.pathPrefixes = pathResult.paths;
  errors.push(...pathResult.errors);

  const hasRange = req.dateFrom !== undefined || req.dateTo !== undefined;
  if (req.dateField === "") {
    if (hasRange) req.dateField = "display";
  }
  if (req.dateField && !["display", "created", "updated"].includes(req.dateField)) {
    errors.push({ field: "date_field", code: "date_field_invalid", message: "date_field must be display, created, or updated." });
  }
  if (req.dateField && !hasRange) {
    errors.push({ field: "date_field", code: "date_field_without_range", message: "date_field requires date_from or date_to." });
  }
  if (req.dateFrom && req.dateTo && dateOnlyBefore(req.dateTo, req.dateFrom)) {
    errors.push({ field: "date_to", code: "before_date_from", message: "date_to must be on or after date_from." });
  }

  if (req.sort && !["relevance", "newest", "oldest"].includes(req.sort)) {
    errors.push({ field: "sort", code: "sort_invalid", message: "sort must be relevance, newest, or oldest." });
    req.sort = "relevance";
  }
  if (!req.sort) {
    req.sort = req.query.trim() ? "relevance" : "newest";
  }

  if (req.limit === 0) req.limit = DEFAULT_LIMIT;
  if (req.limit < 1 || req.limit > MAX_LIMIT) {
    errors.push({ field: "limit", code: "limit_out_of_range", message: "limit must be between 1 and 100." });
  }
  if (req.offset < 0 || req.offset > MAX_OFFSET) {
    errors.push({ field: "offset", code: "offset_out_of_range", message: "offset must be between 0 and 10000." });
  }

  return { request: req, errors };
}

/** A request is effective when it has non-whitespace text or at least one
 * structured filter. A whitespace-only query is not effective so typing a
 * space into an empty field does not trigger a search. */
export function isEffective(req: SearchRequest): boolean {
  return (
    req.query.trim() !== "" ||
    req.tags.length > 0 ||
    req.pathPrefixes.length > 0 ||
    req.dateFrom !== undefined ||
    req.dateTo !== undefined
  );
}

/** Canonicalize a request so equivalent inputs produce identical URL state.
 *
 * The query is preserved verbatim (not trimmed) for the same reason as
 * normalizeSearchRequest: the canonical form feeds the controlled search
 * field's value via the URL, and trimming here would strip spaces the user is
 * typing. The search backends trim on their own. */
export function canonicalizeSearchRequest(req: SearchRequest): SearchRequest {
  return {
    query: req.query,
    tags: [...req.tags].sort(),
    tagMode: req.tagMode || "all",
    pathPrefixes: [...req.pathPrefixes].sort(),
    dateField: req.dateField,
    dateFrom: req.dateFrom,
    dateTo: req.dateTo,
    sort: req.sort,
    limit: req.limit === DEFAULT_LIMIT ? 0 : req.limit,
    offset: req.offset === 0 ? 0 : req.offset,
  };
}

/** Encode a canonical request as URL search params in a fixed key order. */
export function encodeSearchParams(req: SearchRequest): URLSearchParams {
  const c = canonicalizeSearchRequest(req);
  const params = new URLSearchParams();
  // Emit q only when the query has non-whitespace content, but preserve the
  // user's exact spacing (leading/trailing/internal) in the emitted value so
  // the controlled search field round-trips without stripping typed spaces.
  if (c.query.trim()) params.set("q", c.query);
  for (const t of c.tags) params.append("tag", t);
  if (c.tags.length > 0) params.set("tag_mode", c.tagMode);
  for (const p of c.pathPrefixes) params.append("path", p);
  if (c.dateFrom || c.dateTo) params.set("date_field", c.dateField || "display");
  if (c.dateFrom) params.set("date_from", dateOnlyString(c.dateFrom));
  if (c.dateTo) params.set("date_to", dateOnlyString(c.dateTo));
  if (c.sort) params.set("sort", c.sort);
  if (c.limit) params.set("limit", String(c.limit));
  if (c.offset) params.set("offset", String(c.offset));
  // Re-order to a stable key order (URLSearchParams preserves insertion order
  // but append() for repeated keys must stay grouped).
  const ordered = new URLSearchParams();
  for (const key of PARAM_ORDER) {
    const values = params.getAll(key);
    if (values.length === 0) continue;
    if (advancedParamRepeated(key)) {
      for (const v of values) ordered.append(key, v);
    } else {
      ordered.set(key, values[0]);
    }
  }
  return ordered;
}

const REPEATED_PARAMS = new Set(["tag", "path"]);

function advancedParamRepeated(key: string): boolean {
  return REPEATED_PARAMS.has(key);
}

/** Decode URL search params into a raw request plus parse-time field errors. */
export function decodeSearchParams(params: URLSearchParams): { request: SearchRequest; errors: FieldError[] } {
  const errors: FieldError[] = [];
  const known = new Set<string>(PARAM_ORDER);
  for (const k of Array.from(params.keys())) {
    if (!known.has(k)) {
      errors.push({ field: k, code: "unknown_parameter", message: "Unknown search parameter." });
    }
  }
  const singleton = (field: string): string | undefined => {
    const vs = params.getAll(field);
    if (vs.length > 1) {
      errors.push({ field, code: "repeated_parameter", message: "Parameter must appear at most once." });
      return undefined;
    }
    return vs[0];
  };
  const req: SearchRequest = {
    query: singleton("q") ?? "",
    tags: params.getAll("tag"),
    tagMode: (singleton("tag_mode") as TagMode | "") || "all",
    pathPrefixes: params.getAll("path"),
    dateField: (singleton("date_field") as DateField | "") || "",
    dateFrom: parseDateOnlyOpt(singleton("date_from"), "date_from", errors),
    dateTo: parseDateOnlyOpt(singleton("date_to"), "date_to", errors),
    sort: (singleton("sort") as SearchSort | "") || "relevance",
    limit: parseIntOpt(singleton("limit"), DEFAULT_LIMIT, "limit", errors, 1, MAX_LIMIT),
    offset: parseIntOpt(singleton("offset"), 0, "offset", errors, 0, MAX_OFFSET),
  };
  return { request: req, errors };
}

function parseDateOnlyOpt(
  value: string | undefined,
  field: string,
  errors: FieldError[],
): DateOnly | undefined {
  if (value === undefined) return undefined;
  const d = parseDateOnly(value);
  if (!d) {
    errors.push({ field, code: `${field}_invalid`, message: `${field} must be YYYY-MM-DD.` });
    return undefined;
  }
  return d;
}

function parseIntOpt(
  value: string | undefined,
  fallback: number,
  field: string,
  errors: FieldError[],
  min: number,
  max: number,
): number {
  if (value === undefined) return fallback;
  // Require the entire value to be an integer representation; Number.parseInt
  // accepts a numeric prefix ("10junk" -> 10, "2.5" -> 2, "1e2" -> 1), which
  // would silently execute a different canonical request than the URL states.
  if (!/^-?\d+$/.test(value)) {
    errors.push({ field, code: `${field}_out_of_range`, message: `${field} must be an integer.` });
    return fallback;
  }
  const n = Number.parseInt(value, 10);
  if (n < min || n > max) {
    errors.push({ field, code: `${field}_out_of_range`, message: `${field} must be between ${min} and ${max}.` });
    return fallback;
  }
  return n;
}

export { dateOnlyToInstant, dateOnlyNextDayInstant, toUTCSecond };
export { resolveNoteDates };
