# Tasks

## TODO

- [ ] Add tasks here

- [x] Phase 1: Create internal/ignore package (Ignore type, Load/Match/MatchAbs) with table-driven tests covering gitignore subset
- [x] Phase 2: Wire ignore into vault (NewWithOptions, LoadAll SkipDir checks, IsIgnored accessor, ReloadNote/ReadRaw guards) + tests
- [x] Phase 3: Wire ignore into watcher (skip ignored dirs in New, drop ignored events in loop) + test
- [x] Phase 4: Wire ignore into asset handler + raw serving (404 on ignored) + server test
- [ ] Phase 5: Add --vault-ignore CLI flag, thread through server.Config/RuntimeOptions/loadSnapshot
- [ ] Phase 6: Update README.md with an 'Excluding paths' section
