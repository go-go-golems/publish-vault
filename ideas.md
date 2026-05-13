# Design Ideas — Retro Obsidian Publish

## Selected Design: Retro System 1 (Macintosh 1984)

<response>
<text>
**Design Movement**: Early Macintosh / Chicago / Susan Kare pixel aesthetic (System 1, 1984)

**Core Principles**:
1. Monochrome foundation — near-black ink (#1a1a1a) on warm aged paper (#f0ede8); no gradients, no shadows except hard 1px offsets
2. Colour accents reserved exclusively for interactive/functional UI elements: links = deep blue (#0000cc), tags = forest green (#005500), destructive = deep red (#cc0000)
3. Square corners everywhere — zero border-radius, hard 1px borders, inset box-shadows for sunken panels
4. Bitmap-inspired typography — system-ui/Chicago fallback stack, no web fonts, -webkit-font-smoothing: none for pixel-crisp rendering

**Color Philosophy**:
- Background: #f0ede8 (aged paper — warm, not clinical white)
- Ink: #1a1a1a (near-black, not pure black — softer on the eye)
- Accent blue: #0000cc (classic Mac link blue — functional only)
- Tag green: #005500 (forest green — semantic, not decorative)
- Muted: #888 (secondary text, borders, disabled)
- Selection: inverted (black bg / paper text)

**Layout Paradigm**: Asymmetric fixed chrome — top menubar (28px, full-width inverted), left sidebar (224px, collapsible), right content pane (flex-1), right panel (224px, backlinks + graph). No centered hero layouts.

**Signature Elements**:
1. Window chrome with title bar stripes (repeating-linear-gradient drag lines)
2. Inset box-shadows for sunken panels (retro-inset class)
3. Hard 1px borders with 2px offset box-shadows (retro-window class)

**Interaction Philosophy**: Immediate, no-nonsense. Button presses snap (1px translate on :active). Hover states invert (black bg / paper text). No smooth color transitions — state changes are instant like a physical button.

**Animation**: Minimal — only retro-fade-in (120ms ease-out, 4px translateY) for note content load. Cursor blink animation for loading states. No decorative motion.

**Typography System**: Chicago (system-ui) for UI chrome and headings, Monaco (monospace) for code. No web fonts. Font size 14px base, 12px for UI chrome, 11px for metadata.
</text>
<probability>0.08</probability>
</response>
