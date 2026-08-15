/**
 * Wiki-link resolver for the frontend renderer.
 * The Go backend already converts [[wiki links]] to <a class="wiki-link" data-target="slug">
 * anchors. This module post-processes the rendered HTML to:
 *   1. Verify each *cross-note* wiki-link target exists in the note index.
 *   2. Add class "broken" to links whose target is not found.
 *   3. Leave same-note heading links (`[[#Heading]]`, class `wiki-link-self`)
 *      untouched — the backend already resolved their `href` to `#<id>`.
 *   4. Resolve embed placeholders to inline note excerpts.
 */

export type SlugSet = Set<string>;

/**
 * Minimal shape of an anchor element that the validation policy reads. Keeping
 * it structural lets the policy be unit-tested in the node environment (which
 * has no `DOMParser`) with a tiny stub instead of a full DOM.
 */
export interface WikiLinkLike {
  classList: { contains: (c: string) => boolean };
  getAttribute: (n: string) => string | null;
}

/**
 * Returns the note slug a wiki-link anchor targets for the "target note not
 * found" check, or `null` when the anchor is not subject to that check.
 *
 * Same-note heading links — `[[#Heading]]`, rendered by the Go backend with
 * class `wiki-link-self` and *no* `data-target` — point at a heading in the
 * current note, not another note. The backend resolves their `href` to the
 * real heading id in `resolveSelfHeadingLinks` (see
 * `internal/parser/parser.go`). This function must return `null` for them so
 * `resolveWikiLinks` neither marks them `broken` nor overwrites their resolved
 * `href` with `"#"` — which is exactly the regression this guard prevents.
 *
 * Extracted as a pure function so the policy is named and testable without a
 * DOM (the web test runner runs in the node environment, which has no
 * `DOMParser`; see `vitest.config.ts`).
 */
export function wikiLinkTargetForValidation(el: WikiLinkLike): string | null {
  if (el.classList.contains("wiki-link-self")) return null;
  return el.getAttribute("data-target") ?? "";
}

/**
 * Post-process rendered HTML string:
 * - Mark broken cross-note wiki-links whose target is not in the note index.
 * - Preserve same-note heading links (`wiki-link-self`) as the backend
 *   rendered them.
 * - Return the processed HTML.
 */
export function resolveWikiLinks(html: string, slugSet: SlugSet): string {
  // Use DOMParser in browser context
  if (typeof document === "undefined") return html;

  const parser = new DOMParser();
  const doc = parser.parseFromString(html, "text/html");

  // Validate wiki-link anchors. Same-note heading links are skipped by
  // wikiLinkTargetForValidation so their server-resolved href survives.
  doc.querySelectorAll("a.wiki-link").forEach((el) => {
    const target = wikiLinkTargetForValidation(el);
    if (target === null) return;
    if (!slugSet.has(target)) {
      el.classList.add("broken");
      el.setAttribute("title", `Note not found: ${target}`);
      el.setAttribute("href", "#");
    }
  });

  return doc.body.innerHTML;
}

/**
 * Extract all wiki-link targets from rendered HTML.
 */
export function extractWikiLinkTargets(html: string): string[] {
  if (typeof document === "undefined") return [];
  const parser = new DOMParser();
  const doc = parser.parseFromString(html, "text/html");
  const targets: string[] = [];
  doc.querySelectorAll("a.wiki-link").forEach((el) => {
    const t = el.getAttribute("data-target");
    if (t) targets.push(t);
  });
  return targets;
}

/**
 * Build a slug set from a list of note slugs for O(1) lookup.
 */
export function buildSlugSet(slugs: string[]): SlugSet {
  return new Set(slugs);
}
