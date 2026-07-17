/**
 * Registry completeness: every component type appearing in the captured
 * server IR fixtures must have an adapter in the default registry, so real
 * pages never hit the UnknownWidget fallback.
 */
import { describe, expect, it } from "vitest";
import { defaultWidgetRegistry } from "./defaultRegistry";
import type { WidgetNode } from "./ir";
import recentPageFixture from "./__fixtures__/recent-page.json";

function collectComponentTypes(node: unknown, out: Set<string>): void {
  if (Array.isArray(node)) {
    for (const child of node) collectComponentTypes(child, out);
    return;
  }
  if (!node || typeof node !== "object") return;
  const candidate = node as { kind?: unknown; type?: unknown; children?: unknown; props?: unknown };
  if (candidate.kind === "component" && typeof candidate.type === "string") {
    out.add(candidate.type);
  }
  collectComponentTypes(candidate.children, out);
  if (candidate.props && typeof candidate.props === "object") {
    for (const value of Object.values(candidate.props)) {
      collectComponentTypes(value, out);
    }
  }
}

describe("defaultWidgetRegistry", () => {
  it("covers every component type in the recent-page fixture", () => {
    const types = new Set<string>();
    collectComponentTypes((recentPageFixture as { root: WidgetNode }).root, types);
    expect(types.size).toBeGreaterThan(0);
    for (const type of types) {
      expect(defaultWidgetRegistry.has(type), `missing adapter for ${type}`).toBe(true);
    }
  });

  it("rejects duplicate adapter registrations", () => {
    const entries = defaultWidgetRegistry.entries();
    const types = entries.map(adapter => adapter.type);
    expect(new Set(types).size).toBe(types.length);
  });
});
