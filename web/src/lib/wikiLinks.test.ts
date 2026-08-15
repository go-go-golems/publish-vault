import { describe, expect, it } from "vitest";
import {
  wikiLinkTargetForValidation,
  resolveWikiLinks,
  type WikiLinkLike,
} from "./wikiLinks";

/**
 * Regression tests for PV-WIKILINK-022.
 *
 * The bug: the frontend `resolveWikiLinks` selected every `a.wiki-link`,
 * read `data-target`, and — when it was empty — marked the link `broken` and
 * overwrote `href` with `"#"`. Same-note heading links (`[[#Heading]]`) are
 * rendered by the Go backend with class `wiki-link-self` and *no* `data-target`,
 * their `href` already resolved to the real heading id by
 * `resolveSelfHeadingLinks`. The check therefore clobbered all of them.
 *
 * The fix lives in `wikiLinkTargetForValidation`: same-note links return `null`
 * and are skipped. These tests pin the policy without a DOM (the web test runner
 * uses the node environment; see `vitest.config.ts`). A full DOM-level test that
 * asserts the resolved `href` survives would need `happy-dom`/`jsdom`, which the
 * project does not currently depend on — see the design doc's open questions.
 */

/** Minimal anchor stub matching the WikiLinkLike interface. */
function el(opts: { classes?: string[]; target?: string | null }): WikiLinkLike {
  const classes = new Set(opts.classes ?? []);
  const attrs: Record<string, string | null> = {};
  if (opts.target !== undefined && opts.target !== null) {
    attrs["data-target"] = opts.target;
  }
  return {
    classList: { contains: (c: string) => classes.has(c) },
    getAttribute: (n: string) => (n in attrs ? attrs[n] : null),
  };
}

describe("wikiLinkTargetForValidation", () => {
  it("skips a same-note heading link (wiki-link-self, no data-target)", () => {
    // Exact shape the backend emits for [[#Heading]] after resolveSelfHeadingLinks.
    const self = el({ classes: ["wiki-link", "wiki-link-self"] });
    expect(wikiLinkTargetForValidation(self)).toBeNull();
  });

  it("skips a same-note link even if it carries a data-target", () => {
    // The class is the discriminator, not the attribute, so a self link can
    // never be mis-validated as a broken note link.
    const self = el({ classes: ["wiki-link", "wiki-link-self"], target: "anything" });
    expect(wikiLinkTargetForValidation(self)).toBeNull();
  });

  it("returns the data-target of a cross-note link for validation", () => {
    const cross = el({ classes: ["wiki-link"], target: "research/notes/foo" });
    expect(wikiLinkTargetForValidation(cross)).toBe("research/notes/foo");
  });

  it("returns '' for a cross-note link with no data-target (still subject to the check)", () => {
    // A degenerate cross-note link (e.g. [[!!!]] -> slug "") has no data-target
    // but is not a self link. It stays subject to the check; an empty target is
    // never in the slug set, so it is marked broken — preserving prior behavior.
    const cross = el({ classes: ["wiki-link"] });
    expect(wikiLinkTargetForValidation(cross)).toBe("");
  });
});

describe("resolveWikiLinks (node environment, no DOM)", () => {
  // vitest runs in the node environment, where `document` is undefined, so
  // resolveWikiLinks returns its input unchanged. The DOM path runs in the
  // browser/SSR; this guard keeps it from throwing under SSR (entry-server).
  it("returns html unchanged when no DOM is available", () => {
    const html = '<a class="wiki-link wiki-link-self" href="#heading-id">x</a>';
    expect(resolveWikiLinks(html, new Set(["a"]))).toBe(html);
  });
});
