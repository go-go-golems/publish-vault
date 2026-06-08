# Tasks

## TODO

- [x] Add tasks here

- [x] Create favicon handler in internal/server/favicon.go with cascading resolution (CLI override → vault root → 404)
- [x] Add --favicon CLI flag to serve command and pass FaviconPath through Config
- [x] Wire favicon handler into router in server.go (both SSR and non-SSR modes)
- [x] Write unit tests for favicon handler in favicon_test.go
- [x] Add link rel=icon tags to web/index.html
- [x] Create go-go-parc favicon SVG and ICO in vault root
- [ ] Update diary and ticket bookkeeping
