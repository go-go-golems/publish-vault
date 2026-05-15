# Mobile Layout Brainstorm — Retro Obsidian Publish

## Current Desktop Layout (3 columns)

```
┌──────────────────────────────────────────────────────┐
│  ☰  │  ◆ Retro Knowledge Base  │  Search  │  🕐    │  ← Menubar
├──────┼──────────────────────────┼────────────────────┤
│ Side │     Main Content         │  Linked Mentions   │
│ bar  │                          │  (backlinks)       │
│      │  Breadcrumb              │                    │
│ Tree │  Title                   │  [Card 1]          │
│      │  Frontmatter/Tags        │  [Card 2]          │
│ Sear │  Note body (scroll)      │  [Card 3]          │
│ ch   │                          │  (scroll)          │
│      │                          │                    │
│ 20%  │         55%              │       25%          │
└──────┴──────────────────────────┴────────────────────┘
```

## Problems on Mobile (375px)

1. Three columns on 375px = ~75px per column — unusable
2. Two resize handles waste pixels
3. Two separate scroll areas (sidebar tree, backlinks panel) confuse touch users
4. The menubar has too many items (search button, panel toggle, clock)
5. The right panel (backlinks) steals space from the note you're trying to read
6. The left sidebar steals space from the note you're trying to read
7. Backlinks are useful context but secondary — they shouldn't take permanent space

## Guiding Principles

1. **One thing at a time**: On mobile, the user should see ONE primary task — reading a note
2. **Full-width content**: The note body deserves 100% of the viewport
3. **Secondary content is accessible but hidden**: Backlinks and file tree are one tap away
4. **No resize handles on mobile**: They're a desktop affordance, useless on touch
5. **Single scroll**: Only one scrollable region — the note body
6. **Large tap targets**: Everything ≥ 44px height for touch

## Proposed Mobile Layout

### State A: Reading a note (default)

```
┌──────────────────────┐
│  ☰   Retro KB    🔍  │  ← Compact menubar
├──────────────────────┤
│                      │
│  Breadcrumb: Index   │
│                      │
│  # Index             │  ← Note title
│                      │
│  Welcome to the...   │  ← Note body
│  (full width text)   │
│                      │
│  ... continues ...   │
│                      │
│  ─── Backlinks ───   │  ← Inline section at bottom
│  • Epistemology      │
│  • Stoicism          │
│  • Zettelkasten      │
│                      │
└──────────────────────┘
     Single scroll
```

### State B: Sidebar drawer (toggled by ☰)

```
┌──────────────────────┐
│  ☰   Retro KB    🔍  │
├──────────┬───────────┤
│ RETRO KB │  ▓▓▓▓▓▓  │
│          │  ▓▓▓▓▓▓  │  ← Semi-transparent
│ Search…  │  ▓▓▓▓▓▓     backdrop
│          │  ▓▓▓▓▓▓  │
│ > Philo  │  ▓▓▓▓▓▓  │
│   - Epi  │  ▓▓▓▓▓▓  │
│   - Stoi │  ▓▓▓▓▓▓  │
│ Index    │  ▓▓▓▓▓▓  │
│          │  ▓▓▓▓▓▓  │
│ ~80vw    │           │
└──────────┴───────────┘
   Tapping backdrop or a note closes drawer
```

### State C: Search page

```
┌──────────────────────┐
│  ☰   Retro KB    🔍  │
├──────────────────────┤
│  Search: [________]  │
│                      │
│  Results:            │
│  • Note 1 — excerpt  │
│  • Note 2 — excerpt  │
│  • Note 3 — excerpt  │
│                      │
└──────────────────────┘
```

## Key Decisions

### D1: Backlinks move inline at the bottom of the note

**Why**: On mobile, the right panel steals 25-40% of the viewport. Backlinks are secondary content — users primarily want to read. By moving them inline (below the note body), they're always visible but don't reduce the reading area.

**How**: In `NotePage.tsx`, on mobile, render `BacklinksPanel` below `NoteRenderer` instead of in a separate panel. Use `md:hidden` / `hidden md:block` to toggle.

### D2: Right panel is desktop-only

**Why**: The resizable right panel with resize handle is a desktop affordance. On mobile it's clutter.

**How**: Hide the entire `ResizablePanelGroup` right panel on mobile. Show backlinks inline instead.

### D3: Sidebar is an off-canvas drawer

**Why**: Already partially implemented. The drawer slides in from the left with a backdrop. Tapping a note or the backdrop closes it.

### D4: Menubar simplified on mobile

**Why**: Clock, panel toggle, and "Search" text don't fit. Replace with icons.

**Current mobile menubar**: ☰ | "RETRO KNOWLEDGE BASE" | 🔍
**Remove on mobile**: separator, "Search" text, panel toggle button, clock

## Implementation Changes

1. **`NotePage.tsx`**: Add inline backlinks section below note body on mobile (`md:hidden`)
2. **`NotePage.tsx`**: Hide right panel `ResizablePanelGroup` on mobile (`hidden md:flex`)
3. **`VaultLayout.tsx`**: Already has mobile drawer — just needs z-index fix (done)
4. **`uiSlice.ts`**: Initialize `rightPanelOpen` to false on mobile (or just let it be, since we hide the panel anyway)
5. **`index.css`**: Add mobile overrides for note prose padding, touch targets

## What NOT to do

- Don't create separate mobile routes/components
- Don't remove desktop functionality
- Don't add a separate "mobile backlinks" button — inline is simpler
- Don't try to make the resize handle work on touch
