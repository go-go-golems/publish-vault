# Changelog

## 2026-07-14

- Initial workspace created


## 2026-07-14

Created ticket: analysis of bundle size (2.03MB JS / 599KB gz, single chunk) and SSR HTML bloat (1.31MB, ~834KB inlined notes+tree). Wrote intern-grade design doc with 5 phased implementation steps including highlight.js per-language lazy loading via import.meta.glob.

### Related Files

- /home/manuel/code/wesen/go-go-golems/publish-vault/web/src/components/organisms/NoteRenderer/NoteRenderer.tsx — Static hljs+mermaid imports are the heaviest contributors


## 2026-07-14

Validated with docmgr doctor (clean). Uploaded bundle (design doc + diary) to reMarkable at /ai/2026/07/14/PERF-BUNDLE-014.

