import { describe, expect, it } from "vitest";
import { resolvePageShell } from "./shell";
import readerPageFixture from "./__fixtures__/reader-page.json";

describe("resolvePageShell", () => {
  it("defaults to vault chrome when no shell is declared", () => {
    expect(resolvePageShell(undefined)).toEqual({ kind: "vault" });
    expect(resolvePageShell(null)).toEqual({ kind: "vault" });
    expect(resolvePageShell("bogus")).toEqual({ kind: "vault" });
    expect(resolvePageShell({})).toEqual({ kind: "vault" });
  });

  it("maps none and root-owned to none", () => {
    expect(resolvePageShell({ kind: "none" })).toEqual({ kind: "none" });
    expect(resolvePageShell({ kind: "root-owned" })).toEqual({ kind: "none" });
  });

  it("resolves an app shell with sidebar navigation", () => {
    const shell = resolvePageShell({
      kind: "app",
      navigation: {
        placement: "sidebar",
        activeItemId: "reader",
        sections: [
          {
            id: "pages",
            label: { kind: "text", text: "Pages" },
            items: [{ id: "reader", label: { kind: "text", text: "Reader" } }],
          },
        ],
      },
    });
    expect(shell.kind).toBe("app");
    if (shell.kind !== "app") return;
    expect(shell.navigation.placement).toBe("sidebar");
    expect(shell.navigation.activeItemId).toBe("reader");
    expect(shell.navigation.sections).toHaveLength(1);
  });

  it("treats an app shell without sidebar placement as top", () => {
    const shell = resolvePageShell({ kind: "app", navigation: { sections: [] } });
    expect(shell.kind).toBe("app");
    if (shell.kind !== "app") return;
    expect(shell.navigation.placement).toBe("top");
  });

  it("resolves the reader golden's real shell to a sidebar app shell", () => {
    const shell = resolvePageShell((readerPageFixture as { shell?: unknown }).shell);
    expect(shell.kind).toBe("app");
    if (shell.kind !== "app") return;
    expect(shell.navigation.placement).toBe("sidebar");
    expect(shell.navigation.sections[0]?.items[0]?.action).toMatchObject({
      kind: "navigate",
    });
  });
});
