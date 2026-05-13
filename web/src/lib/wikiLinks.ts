/**
 * Wiki-link resolver for the frontend renderer.
 * The Go backend already converts [[wiki links]] to <a class="wiki-link" data-target="slug">
 * anchors. This module post-processes the rendered HTML to:
 *   1. Verify each wiki-link target exists in the note index.
 *   2. Add class "broken" to links whose target is not found.
 *   3. Resolve embed placeholders to inline note excerpts.
 */

export type SlugSet = Set<string>;

/**
 * Post-process rendered HTML string:
 * - Mark broken wiki-links
 * - Return the processed HTML
 */
export function resolveWikiLinks(html: string, slugSet: SlugSet): string {
  // Use DOMParser in browser context
  if (typeof document === "undefined") return html;

  const parser = new DOMParser();
  const doc = parser.parseFromString(html, "text/html");

  // Resolve wiki-link anchors
  doc.querySelectorAll("a.wiki-link").forEach((el) => {
    const target = el.getAttribute("data-target") ?? "";
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
