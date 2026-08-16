import { describe, expect, it } from "vitest";
import { pickScrollAction, scrollKeyOf } from "./scrollRestoration";

/**
 * Regression tests for PV-SCROLL-024.
 *
 * `pickScrollAction` encodes the back/forward scroll-restore precedence
 * (hash > restore > top, none for non-note routes) and `scrollKeyOf` encodes the
 * scroll identity (`location.key + "|" + hash`) that the hook keys saved
 * offsets by. Both are pure so they can be tested in the node environment (the
 * web test runner is `environment: "node"`); the DOM-dependent
 * `useScrollRestoration` hook is exercised manually on the real vault — see
 * the design doc's test plan.
 */

const note = (
  key: string,
  hash = ""
): { key: string; pathname: string; hash: string } => ({
  key,
  pathname: "/note/some-note",
  hash,
});

describe("scrollKeyOf", () => {
  it("combines the location key and hash", () => {
    expect(scrollKeyOf(note("k1"))).toBe("k1|");
    expect(scrollKeyOf(note("k1", "#heading"))).toBe("k1|#heading");
  });

  it("distinguishes same-note fragment states that share a key", () => {
    // Heading permalinks advance the history stack without changing the
    // React Router key, so the hash is what tells the pre- and post-anchor
    // states apart.
    expect(scrollKeyOf(note("k1"))).not.toBe(
      scrollKeyOf(note("k1", "#heading"))
    );
  });

  it("distinguishes different notes by key", () => {
    expect(scrollKeyOf(note("k1"))).not.toBe(scrollKeyOf(note("k2")));
  });
});

describe("pickScrollAction", () => {
  it("restores when a saved offset exists for the note", () => {
    const saved = new Map([[scrollKeyOf(note("k1")), 1234]]);
    expect(pickScrollAction(note("k1"), saved)).toBe("restore");
  });

  it("scrolls to top for a fresh note with no saved offset", () => {
    expect(pickScrollAction(note("k1"), new Map())).toBe("top");
  });

  it("defers to hash even when a saved offset exists (hash beats restore)", () => {
    const saved = new Map([[scrollKeyOf(note("k1")), 1234]]);
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
      pickScrollAction(
        { key: "k1", pathname: "/search", hash: "#x" },
        new Map()
      )
    ).toBe("none");
  });

  it("treats /note/ exactly (the route prefix) as a note route", () => {
    expect(
      pickScrollAction({ key: "k1", pathname: "/note/", hash: "" }, new Map())
    ).toBe("top");
  });

  it("distinguishes notes by key: a different key with no saved offset is top", () => {
    const saved = new Map([[scrollKeyOf(note("k1")), 1234]]);
    expect(pickScrollAction(note("k2"), saved)).toBe("top");
    expect(pickScrollAction(note("k1"), saved)).toBe("restore");
  });

  it("does not restore a hashed state from an unhashed saved offset", () => {
    // A heading permalink writes its own offset under the hashed identity; the
    // unhashed offset must not leak into a hashed location (which defers to
    // the heading scroll anyway), so back/forward to the unhashed state is
    // what restores the pre-anchor position.
    const saved = new Map([[scrollKeyOf(note("k1")), 1234]]);
    expect(pickScrollAction(note("k1", "#heading"), saved)).toBe("hash");
    expect(pickScrollAction(note("k1"), saved)).toBe("restore");
  });
});
