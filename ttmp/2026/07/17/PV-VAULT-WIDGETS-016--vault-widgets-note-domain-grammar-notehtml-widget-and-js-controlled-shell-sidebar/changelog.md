# Changelog

## 2026-07-17

- Initial workspace created


## 2026-07-17

Ticket created from 2026-07-17 conversation decisions (sibling module vault.widgets, existing-component adapters, NoteHtml as only new component, shell/sidebar support, dogfood on go-go-parc). Design doc written; implementation not started.


## 2026-07-17

Implemented phases A-D: vault.widgets module (6 node builders, provider-backed backlinks), 6 frontend adapters incl. new NoteHtml organism (flag-gated enhancement pipeline, wiki-links as dispatched actions), VaultLayout sidebar slot + SidebarNav + ported resolvePageShell with sidebar-slot context, reader.js example with verified v3 shell grammar, deterministic reader golden. Fixed query-string forwarding in WidgetPage and 'Content' auto-section headings. Dogfooded on go-go-parc via content-adjacent .publish/pages (981 notes, tmux session live).

### Related Files

- /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/internal/vaultwidgets/vaultwidgets.go — vault.widgets module
- /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/web/src/components/organisms/NoteHtml/NoteHtml.tsx — New NoteHtml widget organism
- /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/web/src/widgets/shell.ts — Shell resolution


## 2026-07-17

Fixed disappearing-enhancements bug (conditional hydrate in entry-client + load-bearing memo on NoteBody; React 19 rewrites dangerouslySetInnerHTML every re-render). Consolidated NoteView onto NoteHtml (single note-rendering path, allSlugs prop removed) and extended vw.noteHtml to accept slug strings with loud unknown-slug errors. Full validation incl. smoke:ssr PASS and 3-round hash matrix on the production build.

