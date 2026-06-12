---
title: Style fixes analysis and implementation guide
doc_type: design-doc
status: active
intent: long-term
topics: [styling, frontend, ux]
ticket: RETRO-STYLE-009
created: 2026-06-12
---

# Style Fixes Analysis and Implementation Guide

## Executive Summary

Five UI/UX issues need fixing in the publish-vault frontend: (1) Chicago font reference that doesn't exist as a web font, (2) backlink sidebar icon color mismatch, (3) clock that doesn't update, (4) diamond symbol next to vault name, (5) search page doesn't sync with URL query parameter. All fixes are localized to specific React components and the main CSS file.

## Problem Statement

The retro System 1 theme has accumulated several visual and functional inconsistencies:

1. **Chicago font** — `font-family: "Chicago", ...` is specified in `index.css` but Chicago is not a web font available on any modern system. It silently falls back to `system-ui`, making the explicit reference misleading. The design philosophy header also references Chicago, which should be updated.

2. **Backlink sidebar icon color** — The `BacklinksPanel` component renders `<Icon name="link" />` inside a `.retro-window-title` bar (which has `background: var(--color-ink); color: var(--color-paper)`). The Lucide `<Link>` icon inherits `currentColor`, so it should appear in paper color. However, the top-bar panel-right icon in the menubar is also rendered with `<Icon>`, but the menubar button gets an inverted color treatment when active (`bg-[var(--color-paper)] text-[var(--color-ink)]`). The "mismatch" is that the backlink panel title bar icon appears as `var(--color-paper)` (white on dark) while the menubar panel-right toggle icon appears as `var(--color-ink)` (dark on dark) when the menubar is in its default state, creating a visual disconnect. The fix is to ensure the panel-right icon in the menubar uses a consistent color with the rest of the menubar items.

3. **Clock doesn't update** — `HydrationSafeClock` in `VaultLayout.tsx` renders `new Date().toLocaleTimeString()` once on mount and never re-renders. It needs a `setInterval` to tick every minute.

4. **Diamond symbol next to vault name** — In `VaultLayout.tsx`, the vault name button renders `&#9670; {vaultName}` (◆ character). This should be removed.

5. **Search doesn't sync with URL** — `SearchPage` reads query from Redux (`s.ui.searchQuery`) but never reads from or writes to the URL query string. Navigating to `/search?q=foo` doesn't populate the search. Typing in search doesn't update the URL. This means:
   - Direct links to search results don't work
   - Browser back/forward doesn't work for search
   - The search state is lost on page refresh

## Current-State Architecture

### Key files

| File | Role |
|------|------|
| `web/src/index.css` | Global CSS with design tokens, base styles, component utilities |
| `web/src/components/pages/VaultLayout/VaultLayout.tsx` | Main layout: menubar, sidebar, content area, clock |
| `web/src/components/organisms/BacklinksPanel/BacklinksPanel.tsx` | Right-panel backlinks with window chrome title bar |
| `web/src/components/pages/SearchPage/SearchPage.tsx` | Search page: input + results list |
| `web/src/components/molecules/SearchBar/SearchBar.tsx` | Reusable search input with debounce |
| `web/src/store/uiSlice.ts` | Redux slice holding `searchQuery`, `sidebarOpen`, etc. |
| `web/src/App.tsx` | Routes: `/`, `/note/*`, `/search` |

### Styling approach

The app uses Tailwind CSS v4 with custom design tokens defined in `@theme inline {}` and `:root {}` CSS variables. The retro aesthetic uses:
- Zero border-radius
- 1px hard borders everywhere
- Monochrome ink-on-paper palette with limited accent colors
- Custom utility classes (`.retro-window`, `.retro-menubar`, `.retro-search`, etc.)

### Routing approach

React Router v6 with `BrowserRouter`. Routes are defined in `App.tsx`:
- `/` → HomeRedirect
- `/note/*` → NotePage
- `/search` → SearchPage (no query param support currently)

## Gap Analysis

| Issue | Current behavior | Desired behavior |
|-------|-----------------|-----------------|
| Chicago font | `font-family: "Chicago", "Charcoal", system-ui, ...` — Chicago never loads | Remove "Chicago" from font stack; use `system-ui, -apple-system, sans-serif` as primary |
| Sidebar icon color | Panel-right icon in menubar matches menubar color (white on dark bar), but when toggled active, gets inverted; the backlink panel icon is consistent within its title bar | Ensure the panel-right toggle icon is always visible and consistent. The actual fix: the panel-right button icon in the menubar should have `currentColor` (white on dark menubar), which it already does — the mismatch is visual perception because the active state inverts colors. Make the active/hover state more subtle or use a different indicator. |
| Clock | Renders time once, never updates | Tick every minute via `setInterval` |
| Diamond | `&#9670;` prepended to vault name | Remove the `&#9670; ` prefix |
| Search URL | No URL query param support | Read `?q=` on mount, write `?q=` on search, sync with Redux |

## Proposed Solution

### Fix 1: Remove Chicago font

**File:** `web/src/index.css`

- Remove `"Chicago", "Charcoal", ` from `body` font-family
- Remove `"Chicago", "Charcoal", ` from headings font-family
- Update the design philosophy comment header to remove Chicago references
- Keep the fallback to `system-ui, -apple-system, sans-serif`

### Fix 2: Backlink sidebar icon color

**File:** `web/src/components/pages/VaultLayout/VaultLayout.tsx`

The panel-right toggle button in the menubar currently uses a conditional class that inverts colors when `rightPanelOpen` is true. This creates a visual mismatch with the backlink panel's title bar icon (which is always paper-on-ink). The fix is to make the toggle button's active state use a subtle indicator (like an underline or a different background shade) instead of fully inverting, so the icon color stays consistent with the menubar's white-on-dark scheme.

### Fix 3: Clock updates

**File:** `web/src/components/pages/VaultLayout/VaultLayout.tsx`

Add a `setInterval` in `HydrationSafeClock` that updates every 60 seconds. Clean up the interval on unmount.

### Fix 4: Remove diamond

**File:** `web/src/components/pages/VaultLayout/VaultLayout.tsx`

Remove `&#9670; ` from the vault name button text.

### Fix 5: Search URL sync

**Files:** `web/src/App.tsx`, `web/src/components/pages/SearchPage/SearchPage.tsx`, `web/src/components/molecules/SearchBar/SearchBar.tsx`, `web/src/components/pages/VaultLayout/VaultLayout.tsx`

Strategy:
1. Add `useSearchParams` from React Router to `SearchPage`
2. On mount, read `?q=` and dispatch `setSearchQuery`
3. On search input change, update the URL search params
4. In `VaultLayout.handleSearch`, update the URL to `/search?q=...` instead of just `/search`
5. Make `SearchBar` accept an optional `value` prop for controlled mode (currently it's uncontrolled with `initialValue`)

## Implementation Plan

1. **Step 1 — Remove Chicago font** (index.css only)
2. **Step 2 — Remove diamond from vault name** (VaultLayout.tsx)
3. **Step 3 — Fix backlink sidebar icon color** (VaultLayout.tsx — adjust panel-right toggle active state)
4. **Step 4 — Fix clock to update** (VaultLayout.tsx — HydrationSafeClock)
5. **Step 5 — Fix search URL sync** (App.tsx, SearchPage.tsx, SearchBar.tsx, VaultLayout.tsx)

Each step should be a separate commit with a clear message.

## Risks and Open Questions

- **Font removal impact:** The Chicago font was a core part of the design philosophy. Removing it means the retro feel relies entirely on the other styling tokens (zero radius, hard borders, monochrome palette). This is acceptable since Chicago was never actually rendering.
- **Search URL depth:** Should we also update the URL when navigating to a note from search results? Currently note navigation uses `/note/:slug`. This is out of scope for this ticket.
- **Clock SSR:** The `HydrationSafeClock` already handles SSR via the `mounted` state guard. The interval approach is client-only, which is correct.
