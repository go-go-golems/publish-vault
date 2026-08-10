# Tasks

## TODO

- [ ] Phase 1 (mitigation): raise container memory limit 1536Mi -> 3Gi and set GOMEMLIMIT=2600MiB in the gitops deployment.yaml; add initialDelaySeconds/failureThreshold to the liveness and readiness probes so a probe cannot restart the pod mid-load <!-- t:e75u -->
- [ ] Phase 2 (ground truth): collect 'memory phase=' log lines from the crashing pod (kubectl logs --previous -c app | grep '^memory phase='), the app container resources block, and the git-sync container args including --period, --webhook-timeout and --webhook-backoff <!-- t:1l6m -->
- [ ] Phase 3 (real fix, biggest win): move the bleve index off the Go heap by adding a disk-backed emptyDir volume (NOT medium: Memory, NOT /git) and passing --search-index-path; validate search results are unchanged against the real vault <!-- t:r1ay -->
- [x] Phase 4 (real fix): make RuntimeState.Reload non-overlapping and idempotent - add a reload mutex or singleflight around loadSnapshot, skip the rebuild when EvalSymlinks(configuredRoot) equals the active Snapshot.ResolvedRoot, make reloadHandler answer 204 immediately and reload in the background, and reduce oldSnapshotCloseDelay from 30s to 5s <!-- t:sf4q -->
- [x] Phase 4b: add regression tests in pkg/server/runtime_test.go asserting that N concurrent Reload() calls perform exactly one loadSnapshot and that a reload against an unchanged resolved root is a no-op <!-- t:yfhy -->
- [ ] Phase 5: eliminate the duplicate per-note HTML copy (Note.HTML + Note.sourceHTML, ~68 MiB) by making replaceUnresolvedNoteEmbeds non-destructive and resolving note-embed placeholders at response time in api.getNote, so rebuildHTML becomes idempotent and sourceHTML can be deleted <!-- t:a1fh -->
- [x] Phase 6 (observability): add a --pprof-addr flag that serves net/http/pprof on a private listener, and extend /api/healthz with snapshotRevision, snapshotBuiltAt and reloadInFlight <!-- t:eyfa -->
- [ ] Phase 7 (follow-up ticket): design incremental reload - diff the old and new git worktrees, update only changed notes, reuse the existing search index, and solve the shared-index lifetime problem in closeSnapshotAfter <!-- t:6y94 -->
- [ ] Phase 7b (follow-up): make vault.ReloadNote stop re-running rebuildHTML over the whole vault on every single-file watcher event <!-- t:vwl6 -->
- [ ] Phase 1 remainder (gitops repo wesen/2026-03-27--hetzner-k3s, gitops/kustomize/retro-obsidian-publish/deployment.yaml): raise memory limit 1536Mi -> 3Gi and raise probe initialDelaySeconds above the 82s load time. GOMEMLIMIT no longer needs setting there - the binary now derives it from the cgroup (commit 945c2df). <!-- t:p7bq -->
- [ ] Phase 3 remainder (same gitops file): add a disk-backed emptyDir (NOT medium: Memory, NOT /git) mounted at /var/lib/publish-vault and pass --search-index-path. docker-compose.yml is already wired as the reference (commit 945c2df). <!-- t:fba8 -->
