# Changelog

## 2026-06-12

- Initial workspace created


## 2026-06-12

Step 1: Backend #-prefix tag search (commit 3952fa8)

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/internal/search/search.go — Added extractTagQuery() and searchByTag()
- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/internal/search/search_test.go — New test file for tag search


## 2026-06-12

Step 2: Frontend tag click wiring (commit ba74f31)

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/web/src/components/molecules/NoteCard/NoteCard.tsx — Added onTagClick prop
- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/web/src/components/pages/NotePage/NotePage.tsx — handleTagClick navigates to #tag search
- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/web/src/components/pages/SearchPage/SearchPage.tsx — Added handleTagClick for NoteCard tags


## 2026-06-12

Step 3: TagCloud molecule component (commit 3cdaae2)

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/web/src/components/molecules/TagCloud/TagCloud.tsx — New component


## 2026-06-12

Step 4: Tag cloud on empty search page (commit b3509e1)

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/web/src/components/pages/SearchPage/SearchPage.tsx — TagCloud replaces empty state placeholder

