/**
 * Canonical authored note-date resolution for the frontend (shared with the Go
 * backend contract from PV-SEARCH-027). Search dates are authored frontmatter
 * metadata, never filesystem modification time and never a build-time fallback.
 *
 * The static vault parses frontmatter with js-yaml JSON_SCHEMA (see
 * staticVault.ts) so unquoted RFC3339 scalars reach this resolver as strings.
 *
 * Created aliases (precedence high → low): "created", "date".
 * Updated aliases (precedence high → low): "updated", "modified", "last_updated".
 * A higher-precedence invalid value does NOT fall through to a lower alias.
 */

export type DatePrecision = "date" | "timestamp";

export type NoteDateKind = "created" | "updated";

export type InvalidDateReason =
  | "wrong_type"
  | "invalid_format"
  | "invalid_calendar_date";

export interface NoteDate {
  /** Indexed instant (date-only values use midnight UTC). */
  value: Date;
  precision: DatePrecision;
  /** Canonical lower-case frontmatter key selected by precedence. */
  sourceKey: string;
  /** Original scalar string, retained for API projection. */
  original: string;
}

export interface NoteDates {
  created?: NoteDate;
  updated?: NoteDate;
}

export interface DateWarning {
  concept: NoteDateKind;
  reason: InvalidDateReason;
}

const CREATED_ALIASES = ["created", "date"];
const UPDATED_ALIASES = ["updated", "modified", "last_updated"];

const STRICT_DATE_ONLY = /^\d{4}-\d{2}-\d{2}$/;
const STRICT_RFC3339 =
  /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$/;

function isStrictDateOnly(value: string): boolean {
  return STRICT_DATE_ONLY.test(value);
}

function pad2(n: number): string {
  return String(n).padStart(2, "0");
}

/** Format a Date as a UTC RFC3339 instant at second precision (matches Go). */
export function toUTCSecond(date: Date): string {
  return `${date.getUTCFullYear()}-${pad2(date.getUTCMonth() + 1)}-${pad2(
    date.getUTCDate(),
  )}T${pad2(date.getUTCHours())}:${pad2(date.getUTCMinutes())}:${pad2(
    date.getUTCSeconds(),
  )}Z`;
}

/** API projection value: the original literal for date precision, or a
 * normalized UTC RFC3339 instant for timestamp precision. */
export function apiValue(date: NoteDate | undefined | null): string | null {
  if (!date) return null;
  if (date.precision === "date") return date.original;
  return toUTCSecond(date.value);
}

/** Display precedence: updated over created, otherwise absent. */
export function display(
  dates: NoteDates,
): { kind: NoteDateKind; date: NoteDate } | { kind: ""; date: null } {
  if (dates.updated) return { kind: "updated", date: dates.updated };
  if (dates.created) return { kind: "created", date: dates.created };
  return { kind: "", date: null };
}

function parseNoteDate(
  value: string,
  sourceKey: string,
): { date: NoteDate } | { reason: InvalidDateReason } {
  if (isStrictDateOnly(value)) {
    const [y, m, d] = value.split("-").map((s) => Number.parseInt(s, 10));
    const instant = new Date(Date.UTC(y, m - 1, d));
    if (
      Number.isNaN(instant.getTime()) ||
      instant.getUTCFullYear() !== y ||
      instant.getUTCMonth() + 1 !== m ||
      instant.getUTCDate() !== d
    ) {
      // JavaScript's Date normalizes nonexistent calendar dates (e.g.
      // 2024-02-30 becomes March 1). Reject them so static mode does not
      // silently accept authored dates the Go backend rejects.
      return { reason: "invalid_calendar_date" };
    }
    return {
      date: {
        value: instant,
        precision: "date",
        sourceKey,
        original: value,
      },
    };
  }
  if (STRICT_RFC3339.test(value)) {
    const instant = new Date(value);
    if (Number.isNaN(instant.getTime())) {
      return { reason: "invalid_calendar_date" };
    }
    return {
      date: {
        value: instant,
        precision: "timestamp",
        sourceKey,
        original: value,
      },
    };
  }
  return { reason: "invalid_format" };
}

function lookupAlias(
  fm: Record<string, unknown> | null | undefined,
  aliases: string[],
): { canonical: string; actualKey: string } | null {
  if (!fm) return null;
  const lower = new Map<string, string>();
  for (const k of Object.keys(fm)) lower.set(k.toLowerCase(), k);
  for (const want of aliases) {
    const actual = lower.get(want);
    if (actual !== undefined) return { canonical: want, actualKey: actual };
  }
  return null;
}

function resolveConcept(
  fm: Record<string, unknown>,
  concept: NoteDateKind,
  aliases: string[],
): { date?: NoteDate; warning?: DateWarning } {
  const found = lookupAlias(fm, aliases);
  if (!found) return {};
  const raw = fm[found.actualKey];
  if (typeof raw !== "string") {
    return { warning: { concept, reason: "wrong_type" } };
  }
  const parsed = parseNoteDate(raw, found.canonical);
  if ("date" in parsed) return { date: parsed.date };
  return { warning: { concept, reason: parsed.reason } };
}

/** Resolve authored created and updated dates from frontmatter. Missing or
 * invalid values are omitted and content-free warnings are returned. */
export function resolveNoteDates(
  frontmatter: Record<string, unknown>,
): { dates: NoteDates; warnings: DateWarning[] } {
  const warnings: DateWarning[] = [];
  const created = resolveConcept(frontmatter, "created", CREATED_ALIASES);
  if (created.warning) warnings.push(created.warning);
  const updated = resolveConcept(frontmatter, "updated", UPDATED_ALIASES);
  if (updated.warning) warnings.push(updated.warning);
  return {
    dates: {
      created: created.date,
      updated: updated.date,
    },
    warnings,
  };
}
