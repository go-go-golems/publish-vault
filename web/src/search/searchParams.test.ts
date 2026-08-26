import { describe, expect, it } from "vitest";
import {
  canonicalizeSearchRequest,
  decodeSearchParams,
  encodeSearchParams,
  normalizeSearchRequest,
  parseDateOnly,
  isEffective,
} from "./searchParams";
import type { SearchRequest } from "../types";

function emptyRequest(): SearchRequest {
  return {
    query: "",
    tags: [],
    tagMode: "",
    pathPrefixes: [],
    dateField: "",
    sort: "",
    limit: 0,
    offset: 0,
  };
}

describe("parseDateOnly", () => {
  it("accepts strict YYYY-MM-DD and rejects others", () => {
    expect(parseDateOnly("2024-01-15")).toEqual({ year: 2024, month: 1, day: 15 });
    expect(parseDateOnly("2024-1-5")).toBeNull();
    expect(parseDateOnly("01/15/2024")).toBeNull();
    expect(parseDateOnly("2024-13-40")).toBeNull();
  });
});

describe("normalizeSearchRequest", () => {
  it("applies defaults for an empty request", () => {
    const { request, errors } = normalizeSearchRequest(emptyRequest());
    expect(errors).toHaveLength(0);
    expect(request.limit).toBe(30);
    expect(request.sort).toBe("newest");
    expect(request.tagMode).toBe("all");
    expect(isEffective(request)).toBe(false);
  });

  it("defaults sort to relevance when a query is present", () => {
    const { request } = normalizeSearchRequest({ ...emptyRequest(), query: "memory" });
    expect(request.sort).toBe("relevance");
    expect(isEffective(request)).toBe(true);
  });

  it("normalizes, dedupes, and lowercases tags", () => {
    const { request, errors } = normalizeSearchRequest({
      ...emptyRequest(),
      tags: ["Go", "#rust", "go"],
    });
    expect(errors).toHaveLength(0);
    expect(request.tags).toEqual(["go", "rust"]);
  });

  it("rejects date_to before date_from", () => {
    const { errors } = normalizeSearchRequest({
      ...emptyRequest(),
      dateFrom: parseDateOnly("2024-02-01"),
      dateTo: parseDateOnly("2024-01-01"),
    });
    expect(errors.some((e) => e.field === "date_to" && e.code === "before_date_from")).toBe(true);
  });

  it("defaults date_field to display when a range exists", () => {
    const { request, errors } = normalizeSearchRequest({
      ...emptyRequest(),
      dateFrom: parseDateOnly("2024-01-01"),
      dateTo: parseDateOnly("2024-12-31"),
    });
    expect(errors).toHaveLength(0);
    expect(request.dateField).toBe("display");
  });
});

describe("URL codec round-trip", () => {
  it("encodes and decodes a full request preserving the contract", () => {
    const req: SearchRequest = {
      query: "memory",
      tags: ["go", "performance"],
      tagMode: "all",
      pathPrefixes: ["research/kb/"],
      dateField: "display",
      dateFrom: parseDateOnly("2024-01-01"),
      dateTo: parseDateOnly("2024-12-31"),
      sort: "newest",
      limit: 20,
      offset: 0,
    };
    const encoded = encodeSearchParams(req);
    const url = encoded.toString();
    // Fixed key order, sorted repeated values.
    expect(url).toBe(
      "q=memory&tag=go&tag=performance&tag_mode=all&path=research%2Fkb%2F&date_field=display&date_from=2024-01-01&date_to=2024-12-31&sort=newest&limit=20",
    );
    const { request: decoded, errors } = decodeSearchParams(encoded);
    expect(errors).toHaveLength(0);
    const { request: renorm } = normalizeSearchRequest(decoded);
    const canonical = canonicalizeSearchRequest(req);
    expect(canonicalizeSearchRequest(renorm)).toEqual(canonical);
  });

  it("omits defaults and empty values", () => {
    const { request } = normalizeSearchRequest({ ...emptyRequest(), query: "go" });
    const encoded = encodeSearchParams(request);
    expect(encoded.toString()).toBe("q=go&sort=relevance");
  });

  it("decodes unknown parameters as errors", () => {
    const { errors } = decodeSearchParams(new URLSearchParams("q=x&bogus=1"));
    expect(errors.some((e) => e.field === "bogus" && e.code === "unknown_parameter")).toBe(true);
  });

  it("decodes repeated singletons as errors", () => {
    const { errors } = decodeSearchParams(new URLSearchParams("q=a&q=b"));
    expect(errors.some((e) => e.field === "q" && e.code === "repeated_parameter")).toBe(true);
  });

  it("rejects an explicit limit of 0 instead of defaulting it", () => {
    const { errors } = decodeSearchParams(new URLSearchParams("q=x&limit=0"));
    expect(errors.some((e) => e.field === "limit" && e.code === "limit_out_of_range")).toBe(true);
  });

  it("rejects partially parsed numeric parameters", () => {
    for (const bad of ["10junk", "2.5", "1e2", "0x10"]) {
      const { errors } = decodeSearchParams(new URLSearchParams(`q=x&limit=${bad}`));
      expect(errors.some((e) => e.field === "limit" && e.code === "limit_out_of_range"), bad).toBe(true);
    }
  });
});

describe("parseDateOnly rejects invalid calendar dates", () => {
  it("rejects 2024-02-30 which JS would normalize to March 1", () => {
    expect(parseDateOnly("2024-02-30")).toBeNull();
  });
  it("rejects 2024-04-31 which JS would normalize to May 1", () => {
    expect(parseDateOnly("2024-04-31")).toBeNull();
  });
  it("accepts a valid leap day", () => {
    expect(parseDateOnly("2024-02-29")).toEqual({ year: 2024, month: 2, day: 29 });
  });
  it("rejects a non-leap February 29", () => {
    expect(parseDateOnly("2023-02-29")).toBeNull();
  });
});
