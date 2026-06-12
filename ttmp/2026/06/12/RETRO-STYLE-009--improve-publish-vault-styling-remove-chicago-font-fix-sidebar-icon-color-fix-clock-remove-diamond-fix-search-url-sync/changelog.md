# Changelog

## 2026-06-12

- Initial workspace created


## 2026-06-12

Step 1: Remove Chicago font from CSS (commit 43e4fbc)

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/web/src/index.css — Removed Chicago/Charcoal from font-family stacks and design philosophy comment


## 2026-06-12

Step 2: Remove diamond from vault name (commit 40987e1)

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/web/src/components/pages/VaultLayout/VaultLayout.tsx — Removed ◆ from vault name button


## 2026-06-12

Step 3: Fix backlink sidebar icon color mismatch (commit 2c023c3)

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/web/src/components/pages/VaultLayout/VaultLayout.tsx — Changed panel-right toggle active state from color inversion to dotted underline


## 2026-06-12

Step 4: Fix clock to update every minute (commit 54f74c8)

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/web/src/components/pages/VaultLayout/VaultLayout.tsx — Added setInterval to HydrationSafeClock


## 2026-06-12

Step 5: Fix search URL sync (commit c203b21)

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/web/src/components/molecules/SearchBar/SearchBar.tsx — Added controlled mode with value/onChange props
- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/web/src/components/pages/SearchPage/SearchPage.tsx — Added useSearchParams for URL sync
- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/web/src/components/pages/VaultLayout/VaultLayout.tsx — handleSearch navigates to /search?q=...

