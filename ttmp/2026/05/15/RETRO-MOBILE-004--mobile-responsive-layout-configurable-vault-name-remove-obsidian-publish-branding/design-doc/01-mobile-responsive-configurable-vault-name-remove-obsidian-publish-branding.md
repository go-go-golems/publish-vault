---
title: Mobile Responsive, Configurable Vault Name, Remove Obsidian Publish Branding
doc_type: design-doc
intent: long-term
status: active
topics:
  - mobile
  - config
  - vault-name
  - glazed
  - responsive
  - css
ticket: RETRO-MOBILE-004
created: 2026-05-15
---

# Mobile Responsive, Configurable Vault Name, Remove Obsidian Publish Branding

## Executive Summary

This design covers three related changes to retro-obsidian-publish:

1. **Mobile responsive layout** — The current UI is desktop-only; on viewports narrower than ~768px the sidebar steals 30-40% of screen space and elements overlap. We need an off-canvas sidebar drawer, full-width content, and a simplified header for mobile.
2. **Configurable vault name** — The vault display name ("Demo Vault") is currently hardcoded as a default prop in `VaultLayout.tsx`. We will add a `--vault-name` CLI flag via Glazed, expose it through a new `/api/config` endpoint, and have the frontend fetch and use it dynamically.
3. **Remove "Obsidian Publish" branding** — The sidebar footer contains an "Obsidian Publish" tag. This will be removed entirely.

## Problem Statement

### Mobile

On viewports under ~768px:
- The sidebar takes 30-40% of the viewport, leaving cramped content
- The resizable panel handle is useless on touch
- The menubar items (vault name, search button, panel toggle, clock) overlap
- Two scroll areas (sidebar + content) confuse mobile users
- No safe-area padding for notched devices

### Vault Name

The vault name is hardcoded in two places:
- `VaultLayout.tsx`: `vaultName = "Demo Vault"` (default prop)
- `Sidebar.tsx`: `vaultName = "Vault"` (default prop)

There is no way for a publisher to set their vault name without rebuilding the frontend.

### Obsidian Publish Tag

The sidebar footer reads "Obsidian Publish" with a book icon. This is branding that should not appear in a self-hosted publisher tool.

## Proposed Solution

### 1. Backend: Add `--vault-name` CLI Flag and `/api/config` Endpoint

#### Glazed Flag Addition

In `backend/cmd/retro-obsidian-publish/commands/serve/serve.go`, add a new flag:

```go
fields.New("vault-name", fields.TypeString,
    fields.WithDefault(""),
    fields.WithHelp("Display name for the vault in the web UI. Defaults to the vault directory basename."),
),
```

Add to `Settings` struct:
```go
VaultName string `glazed:"vault-name"`
```

#### Config Propagation

Pass `VaultName` through `appserver.Config` to the server. If empty, derive from the vault directory basename:

```go
if settings.VaultName == "" {
    settings.VaultName = filepath.Base(settings.Vault)
}
```

#### New `/api/config` Endpoint

Add a `GET /api/config` endpoint that returns:

```json
{
  "vaultName": "My Custom Vault",
  "notes": 42
}
```

This is a public, unauthenticated endpoint (no sensitive data).

#### Config File Support

Glazed's `CommandSettings` already include a `config-file` field. Users can create a YAML config:

```yaml
# retro-obsidian-publish.yaml
command-settings:
  vault-name: "My Research Vault"
  vault: "/path/to/vault"
  port: "8080"
```

And run:
```bash
retro-obsidian-publish serve --config-file retro-obsidian-publish.yaml
```

This gives us config-file support for free via Glazed's built-in `--config-file` flag.

### 2. Frontend: Dynamic Config Fetching

#### RTK Query API Slice

Add a `useGetConfigQuery` hook to `web/src/store/vaultApi.ts`:

```typescript
getConfig: builder.query<SiteConfig, void>({
  query: () => "/config",
}),
```

#### VaultLayout Integration

In `App.tsx` or `VaultLayout.tsx`, fetch the config and pass `vaultName` down:

```typescript
const { data: config } = useGetConfigQuery();
// ...
<VaultLayout vaultName={config?.vaultName}>{children}</VaultLayout>
```

### 3. Remove "Obsidian Publish" Footer

Delete the footer div from `Sidebar.tsx`:

```tsx
// DELETE THIS BLOCK:
<div className="border-t border-[var(--color-ink)] px-2 py-1 flex items-center gap-1 text-[10px] text-[var(--color-muted-foreground)]">
  <Icon name="book" size={10} />
  <span>Obsidian Publish</span>
</div>
```

### 4. Mobile Responsive Layout

#### Breakpoint Strategy

| Breakpoint | Behavior |
|---|---|
| ≥768px (desktop) | Current layout: resizable sidebar + content |
| <768px (mobile) | Sidebar hidden by default, hamburger toggle, full-width content |

#### CSS Changes (in `index.css`)

Add a mobile section at the bottom of `@layer components`:

```css
/* ── Mobile Responsive ── */
@media (max-width: 767px) {
  /* Hide the resizable panel group; sidebar becomes fixed overlay */
  .retro-resize-handle {
    display: none !important;
  }
}
```

#### VaultLayout Changes

The key change is making the sidebar an off-canvas drawer on mobile:

```tsx
// Mobile: sidebar is an overlay triggered by hamburger button
// Desktop: sidebar is inline in the resizable panel group
```

Strategy:
- Use CSS/Tailwind responsive classes
- On mobile (`<768px`): hide `ResizablePanelGroup`, show sidebar as a `fixed` overlay with `z-50`, full height, width ~80vw, with a backdrop
- The hamburger button in the menubar toggles `sidebarOpen` (already in Redux `uiSlice`)
- Tap a note or the backdrop closes the sidebar
- Content area gets `w-full` on mobile

#### Menubar Mobile Adaptation

On mobile, the menubar should:
- Show hamburger icon (left)
- Show truncated vault name (center)
- Show search icon that navigates to /search (right)
- Hide the clock and panel-toggle button

```tsx
// Desktop: full menubar with all items
// Mobile: hamburger | vault name | search icon only
```

#### Touch-Friendly Targets

- All interactive elements minimum 44px tap target
- Tree items get taller padding on mobile
- Search input gets larger padding

## Design Decisions

### D1: Server-side config via `/api/config` vs. build-time env vars

**Decision:** Server-side config endpoint.

**Rationale:** 
- Same binary serves multiple vaults without rebuilding
- Config can change on reload (git-sync)
- Glazed already provides `--config-file` for free
- Build-time env vars would require separate frontend builds per vault

### D2: Mobile breakpoint at 768px

**Decision:** 768px breakpoint (standard tablet/mobile boundary).

**Rationale:**
- Aligns with Tailwind's `md:` breakpoint
- Below 768px, the sidebar + content side-by-side is cramped
- Above 768px (iPad portrait), the current layout works acceptably

### D3: Off-canvas drawer vs. always-visible collapsible sidebar

**Decision:** Off-canvas drawer (fixed overlay).

**Rationale:**
- Maximizes content area on mobile
- Familiar pattern (Material Design, Bootstrap offcanvas)
- The existing `sidebarOpen` Redux state already provides the toggle mechanism
- A collapsible sidebar would still steal horizontal space

### D4: Config endpoint is public (no auth)

**Decision:** The `/api/config` endpoint returns only `vaultName` and `notes` count — no sensitive data.

**Rationale:** The vault name is visible in the UI anyway. No security exposure.

## Alternatives Considered

### A1: CSS-only responsive sidebar (no JS state changes)

Considered using CSS `@media` to position the sidebar off-screen. Rejected because the sidebar toggle state is already in Redux (`uiSlice.sidebarOpen`), and we need consistent behavior between desktop and mobile toggles.

### A2: Separate mobile route/components

Considered `/m/note/:slug` routes. Rejected because it duplicates code and routing logic. A single responsive layout is simpler.

### A3: Vite env variables for vault name

Considered `VITE_VAULT_NAME` at build time. Rejected because it requires rebuilding the frontend per vault, while the Go binary already supports config files.

## Implementation Plan

### Phase 1: Backend Config Infrastructure
1. Add `--vault-name` flag to `serve.go` Settings
2. Add `VaultName` to `appserver.Config`
3. Derive default from vault directory basename
4. Add `GET /api/config` endpoint in `api.go`
5. Register route in `server.go`
6. Test with `curl http://localhost:8080/api/config`

### Phase 2: Frontend Config Integration
1. Add `useGetConfigQuery` to `vaultApi.ts`
2. Update `App.tsx` to fetch config and pass `vaultName`
3. Remove hardcoded "Demo Vault" default

### Phase 3: Remove Obsidian Publish Branding
1. Delete the sidebar footer div from `Sidebar.tsx`

### Phase 4: Mobile Responsive Layout
1. Update `VaultLayout.tsx` — mobile sidebar drawer
2. Add mobile CSS media queries to `index.css`
3. Add touch-friendly sizing
4. Test at 375×812 (iPhone) and 768×1024 (iPad)

### Phase 5: Validation
1. Screenshot comparison at desktop and mobile sizes
2. Verify config endpoint returns correct vault name
3. Verify `--config-file` loads vault-name from YAML

## Affected Files

| File | Change |
|---|---|
| `backend/cmd/.../serve/serve.go` | Add `--vault-name` Glazed flag |
| `backend/internal/server/server.go` | Pass VaultName, add route |
| `backend/internal/api/api.go` | Add `/api/config` handler |
| `backend/internal/server/runtime.go` | Store VaultName in RuntimeState |
| `web/src/store/vaultApi.ts` | Add `useGetConfigQuery` hook |
| `web/src/App.tsx` | Fetch config, pass vaultName |
| `web/src/components/organisms/Sidebar/Sidebar.tsx` | Remove footer |
| `web/src/components/pages/VaultLayout/VaultLayout.tsx` | Mobile drawer |
| `web/src/index.css` | Mobile responsive CSS |

## Config File Example

```yaml
# retro-config.yaml
command-settings:
  vault-name: "My Research Institute"
  vault: "/data/vaults/research"
  port: "8080"
  serve-web: true
  watch: false
```

Usage:
```bash
retro-obsidian-publish serve --config-file retro-config.yaml
```

Or via CLI flags only:
```bash
retro-obsidian-publish serve \
  --vault /data/vaults/research \
  --vault-name "My Research Institute" \
  --port 8080
```
