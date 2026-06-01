---
doc_type: design-doc
title: Lightbox modal implementation guide
ticket: RETRO-LIGHTBOX-007
status: active
intent: long-term
topics: [ui, lightbox, image, mermaid, modal]
owners: []
created: 2026-05-31
---

# Lightbox modal implementation guide

## Executive summary

Add a full-screen lightbox modal that opens when the user clicks on an image or a mermaid diagram inside a rendered note. The modal displays the image or diagram at maximum resolution/screen size on a dark overlay, with Escape/click-outside to dismiss. This is a standard "lightbox" pattern used by image galleries, documentation sites, and note-taking apps.

## Problem statement

Currently, images in notes are constrained to `max-width: 100%` inside the note prose column (about 50rem). Mermaid diagrams are similarly constrained. Users cannot view the full resolution of images or explore the detail of complex diagrams. There is no way to zoom in or view content at screen-filling size.

### What the user sees today

- Images render inline at note-column width (≤ 50rem). Clicking an image does nothing.
- Mermaid SVG diagrams render inline, often cramped for complex flowcharts. Clicking does nothing.
- There is no way to view content at full resolution.

### What the user should see

- Clicking an image opens a full-screen modal with the image scaled to fit the viewport (maintaining aspect ratio) on a dark overlay.
- Clicking a mermaid diagram opens the same modal with the SVG scaled to fill the viewport.
- Pressing Escape or clicking the overlay dismisses the modal.
- A close button (✕) is visible in the top-right corner.

## System context: how content gets to the screen

The retro-obsidian-publish system renders note content through a pipeline:

1. **Go backend** (`backend/internal/parser/parser.go`) parses Markdown → HTML. Images become `<img src="/vault-assets/..." alt="...">`. Mermaid code blocks become `<pre><code class="language-mermaid">...</code></pre>`.

2. **API** (`backend/internal/api/api.go`) serves the note as JSON with an `html` field.

3. **NoteRenderer** (`web/src/components/organisms/NoteRenderer/NoteRenderer.tsx`) receives the HTML string and:
   - Resolves wiki links via `resolveWikiLinks()`
   - Renders mermaid blocks by replacing `<pre><code class="language-mermaid">` with `<div class="mermaid-svg">` containing the rendered SVG
   - Applies syntax highlighting to code blocks
   - Injects heading permalinks
   - Injects click handlers for wiki-link SPA navigation

4. The final HTML is injected via `dangerouslySetInnerHTML` into a `<div className="note-prose">`.

The lightbox must be added **after** the mermaid rendering step, because at that point the DOM contains the actual `<img>` and `<div class="mermaid-svg">` elements to attach click handlers to.

## Architecture decision: React Dialog vs. raw DOM overlay

### Option A: React Dialog component (chosen)

Use the existing `@radix-ui/react-dialog` (already in `dependencies`) wrapped in a new `LightboxModal` atom component. The NoteRenderer manages the open/close state and passes the content (img src or SVG HTML) as props.

**Pros:**
- Consistent with existing component patterns (ManusDialog already uses Radix Dialog)
- Built-in accessibility (focus trap, Escape key, aria attributes)
- Declarative React state management
- No manual DOM overlay creation/cleanup

**Cons:**
- Requires lifting click events from `dangerouslySetInnerHTML` content up to React state
- Slight overhead from React re-renders

### Option B: Raw DOM overlay created in useEffect

Create a full-screen `<div>` overlay directly in the DOM when a click happens, using `document.createElement` and manual CSS.

**Pros:**
- Simpler — no React state management needed
- Slightly faster (no React re-render)

**Cons:**
- No accessibility (focus trap, aria) out of the box
- Must manually handle keyboard events, body scroll lock, cleanup
- Inconsistent with the rest of the codebase

**Decision:** Option A (React Dialog). Accessibility and consistency outweigh the minor overhead of lifting click state.

## Component design

### LightboxModal atom

**File:** `web/src/components/atoms/LightboxModal/LightboxModal.tsx`

A dedicated atom component that wraps Radix Dialog with full-screen styling:

```tsx
interface LightboxModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  imageSrc?: string;    // for <img> lightbox
  imageAlt?: string;    // alt text for accessibility
  svgHtml?: string;     // for mermaid SVG lightbox
}
```

**Behavior:**
- When `open=true` and `imageSrc` is provided: render `<img src={imageSrc} alt={imageAlt}>` centered in the viewport at max scale (object-contain)
- When `open=true` and `svgHtml` is provided: render the SVG HTML centered in the viewport with scroll for overflow
- Dark overlay (`bg-black/90`)
- Close button (✕) in top-right corner
- Click overlay to dismiss
- Escape key to dismiss

**Pseudocode:**

```
<Dialog open={open} onOpenChange={onOpenChange}>
  <DialogContent className="full-screen overlay">
    {imageSrc && <img src={imageSrc} alt={imageAlt} />}
    {svgHtml && <div dangerouslySetInnerHTML={{ __html: svgHtml }} />}
    <CloseButton />
  </DialogContent>
</Dialog>
```

### NoteRenderer changes

The NoteRenderer already has a `handleClick` callback that intercepts wiki-link clicks. We extend it to also detect clicks on `<img>` elements and `.mermaid-svg` containers.

**New state:**

```tsx
const [lightbox, setLightbox] = useState<{
  type: "image" | "mermaid";
  src?: string;
  alt?: string;
  svgHtml?: string;
} | null>(null);
```

**Click handler extension:**

```
handleClick(e):
  // ... existing wiki-link handling ...
  
  // Image click
  img = target.closest("img")
  if img and img is inside .note-prose:
    setLightbox({ type: "image", src: img.src, alt: img.alt })
    return
  
  // Mermaid click
  mermaidEl = target.closest(".mermaid-svg")
  if mermaidEl:
    setLightbox({ type: "mermaid", svgHtml: mermaidEl.innerHTML })
    return
```

**Modal rendering:**

```
<LightboxModal
  open={lightbox !== null}
  onOpenChange={(open) => !open && setLightbox(null)}
  imageSrc={lightbox?.type === "image" ? lightbox.src : undefined}
  imageAlt={lightbox?.type === "image" ? lightbox.alt : undefined}
  svgHtml={lightbox?.type === "mermaid" ? lightbox.svgHtml : undefined}
/>
```

### CSS additions

**File:** `web/src/index.css`

```css
/* Clickable images and mermaid diagrams */
.note-prose img { cursor: zoom-in; }
.note-prose img:hover { outline: 2px solid var(--color-link); outline-offset: 2px; }
.mermaid-svg { cursor: zoom-in; }
.mermaid-svg:hover { outline: 2px solid var(--color-link); outline-offset: 2px; }

/* Lightbox modal */
.lightbox-content img {
  max-width: 100vw;
  max-height: 100vh;
  object-fit: contain;
}
.lightbox-content .mermaid-display {
  max-width: 95vw;
  max-height: 95vh;
  overflow: auto;
}
```

## Implementation plan (step-by-step tasks)

### Task 1: Create LightboxModal atom component

Create `web/src/components/atoms/LightboxModal/LightboxModal.tsx` and `LightboxModal.stories.tsx`.

Key details:
- Use `Dialog`, `DialogContent` from `@/components/ui/dialog`
- `DialogContent` with `showCloseButton={true}`, custom full-screen styling
- Support two modes: image display and SVG display
- Override the default Dialog content styling (which is a centered card) with full-screen overlay styling
- The close button should be prominent and easy to hit

### Task 2: Add click handlers to NoteRenderer

Extend the existing `handleClick` in NoteRenderer to detect:
- Clicks on `<img>` elements inside `.note-prose`
- Clicks on `.mermaid-svg` elements

Add `lightbox` state and render `<LightboxModal>`.

### Task 3: Add cursor and hover styles

Add CSS for `cursor: zoom-in` and hover outlines on images and mermaid containers.

### Task 4: Add Storybook stories

Add stories for the LightboxModal:
- Image lightbox
- Mermaid SVG lightbox

### Task 5: Smoke test

Start the server with a vault that has images and mermaid diagrams. Verify both types open in the lightbox and dismiss correctly.

### Task 6: Write implementation guide and upload to reMarkable

## Key files

| File | Role |
|------|------|
| `web/src/components/atoms/LightboxModal/LightboxModal.tsx` | New — the lightbox modal component |
| `web/src/components/atoms/LightboxModal/LightboxModal.stories.tsx` | New — Storybook stories |
| `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx` | Modify — add click handlers and lightbox state |
| `web/src/index.css` | Modify — add cursor/hover/lightbox styles |
| `web/src/components/ui/dialog.tsx` | Existing — Radix Dialog wrapper, used by LightboxModal |

## API reference

### Radix Dialog primitives used

- `Dialog` — root component, manages open/close state
- `DialogContent` — the modal panel, with overlay, close button, and focus trap
- `DialogOverlay` — the dark backdrop (already rendered inside DialogContent)

### CSS variables used

- `--color-ink` — dark text/border color
- `--color-link` — link color, used for hover outlines
- `--color-paper` — light background

## Risks and alternatives

- **Risk: SVG in lightbox might not scale well.** Mermaid SVGs use fixed pixel sizes. In the lightbox, we'll set `width: 100%; height: auto;` on the SVG, and the container will be scrollable if the diagram is very large.
- **Risk: Multiple images might want gallery navigation.** This is a future enhancement. For now, one-at-a-time viewing is sufficient.
- **Alternative: Use an existing lightbox library** (e.g., `yet-another-react-lightbox`). Rejected — we only need a simple modal, and the existing Radix Dialog handles all the accessibility.
