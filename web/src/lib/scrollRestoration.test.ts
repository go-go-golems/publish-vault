import { describe, expect, it } from "vitest";
import { pickScrollAction } from "./scrollRestoration";

/**
 * Regression tests for PV-SCROLL-024.
 *
 * pickScrollAction encodes the back/forward scroll-restore precedence
 * (hash > restore > top, none for non-note routes). It is pure so it can be
 * tested in the node environment (the web test runner is `environment: "node"`;
 * the DOM-dependent `useScrollRestoration` hook is exercised manually on the
 * real vault — see the design doc's test plan).
 */

const note = (key: string, hash = ""): { key: string; pathname: string; hash: string } => ({
  key,
  pathname: "/note/some-note",
  hash,
});

describe("pickScrollAction", () => {
  it("restores when a saved offset exists for the note", () => {
    const saved = new Map([["k1", 1234]]);
    expect(pickScrollAction(note("k1"), saved)).toBe("restore");
  });

  it("scrolls to top for a fresh note with no saved offset", () => {
    expect(pickScrollAction(note("k1"), new Map())).toBe("top");
  });

  it("defers to hash even when a saved offset exists (hash beats restore)", () => {
    const saved = new Map([["k1", 1234]]);
    expect(pickScrollAction(note("k1", "#heading"), saved)).toBe("hash");
  });

  it("defers to hash for a fresh note (hash beats top)", () => {
    expect(pickScrollAction(note("k1", "#heading"), new Map())).toBe("hash");
  });

  it("does nothing for a non-note route", () => {
    expect(
      pickScrollAction({ key: "k1", pathname: "/search", hash: "" }, new Map())
    ).toBe("none");
  });

  it("does nothing for a non-note route even with a hash", () => {
    expect(
      pickScrollAction({ key: "k1", pathname: "/search", hash: "#x" }, new Map())
    ).toBe("none");
  });

  it("treats /note/ exactly (the route prefix) as a note route", () => {
    expect(
      pickScrollAction({ key: "k1", pathname: "/note/", hash: "" }, new Map())
    ).toBe("top");
  });

  it("distinguishes notes by key: a different key with no saved offset is top", () => {
    const saved = new Map([["k1", 1234]]);
    expect(pickScrollAction(note("k2"), saved)).toBe("top");
    expect(pickScrollAction(note("k1"), saved)).toBe("restore");
  });
});
