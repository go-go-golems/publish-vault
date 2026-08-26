import { describe, expect, it } from "vitest";
import {
  apiValue,
  display,
  resolveNoteDates,
  toUTCSecond,
  type NoteDate,
} from "./noteDate";
import fixture from "../../../testdata/search-date-cases.json";

interface ExpectedDate {
  value: string;
  precision: "date" | "timestamp";
  source_key: string;
}
interface ExpectedWarning {
  concept: string;
  reason: string;
}
interface DateCase {
  name: string;
  frontmatter: Record<string, unknown>;
  created: ExpectedDate | null;
  updated: ExpectedDate | null;
  display_kind: "" | "created" | "updated";
  display_value: string | null;
  display_precision: "" | "date" | "timestamp";
  warnings: ExpectedWarning[];
}

const cases = fixture.cases as DateCase[];

function checkDate(
  got: NoteDate | undefined,
  want: ExpectedDate | null,
  label: string,
) {
  if (want === null) {
    expect(got, `${label}: expected nil`).toBeUndefined();
    return;
  }
  expect(got, `${label}: expected non-nil`).toBeDefined();
  if (!got) return;
  expect(apiValue(got), `${label} value`).toBe(want.value);
  expect(got.precision, `${label} precision`).toBe(want.precision);
  expect(got.sourceKey, `${label} source_key`).toBe(want.source_key);
}

describe("resolveNoteDates (shared fixture)", () => {
  expect(cases.length, "fixture has cases").toBeGreaterThan(0);

  for (const c of cases) {
    it(c.name, () => {
      const { dates, warnings } = resolveNoteDates(c.frontmatter);
      checkDate(dates.created, c.created, "created");
      checkDate(dates.updated, c.updated, "updated");

      const shown = display(dates);
      expect(shown.kind, "display kind").toBe(c.display_kind);
      if (c.display_value === null) {
        expect(shown.date, "display date").toBeNull();
      } else {
        expect(shown.date, "display date").not.toBeNull();
        if (shown.date) {
          expect(apiValue(shown.date), "display value").toBe(c.display_value);
          expect(shown.date.precision, "display precision").toBe(
            c.display_precision,
          );
        }
      }

      const got = warnings
        .map((w) => `${w.concept}:${w.reason}`)
        .sort()
        .join(" ");
      const want = c.warnings
        .map((w) => `${w.concept}:${w.reason}`)
        .sort()
        .join(" ");
      expect(got, "warnings").toBe(want);
    });
  }
});

describe("toUTCSecond", () => {
  it("formats a timezone offset instant as UTC at second precision", () => {
    const d = new Date("2024-01-15T13:45:00-05:00");
    expect(toUTCSecond(d)).toBe("2024-01-15T18:45:00Z");
  });
  it("formats a Z instant at second precision (drops milliseconds)", () => {
    const d = new Date("2024-01-15T18:45:00.123Z");
    expect(toUTCSecond(d)).toBe("2024-01-15T18:45:00Z");
  });
});

describe("parseNoteDate edge cases", () => {
  it("rejects a timestamp without timezone as invalid_format", () => {
    const { dates } = resolveNoteDates({ created: "2024-01-15T13:45:00" });
    expect(dates.created).toBeUndefined();
  });
  it("rejects a space-separated datetime as invalid_format", () => {
    const { dates } = resolveNoteDates({ created: "2024-01-15 13:45:00Z" });
    expect(dates.created).toBeUndefined();
  });
});
