# Tasks

## TODO

- [ ] Phase 1: Write internal/parser/math.go — MathSpan, ScanMath (state machine with code-span/fence/escape skipping and Pandoc $ rules), ReplaceMath placeholder emission <!-- t:y31f -->
- [ ] Phase 1: Write internal/parser/math_test.go table test covering the 20 spec cases (currency, code fences, escapes, align, wiki-links-in-math, round-trip fidelity) <!-- t:fwwv -->
- [ ] Phase 1: Wire replaceMathInBody into parser.Parse before replaceWikiLinks <!-- t:a8x0 -->
- [ ] Phase 2: Add stripMathDelimiters to stripMarkdown so search indexes TeX bodies without delimiters <!-- t:11un -->
- [ ] Phase 2: Add vault test asserting rebuildHTML leaves .math regions byte-identical <!-- t:nyuy -->
- [ ] Phase 3: Add @mathjax/src dependency and write web/src/lib/mathjax.ts (TeX->SVG singleton, typesetTeX, ensureMathStyles) <!-- t:vrrv -->
- [ ] Phase 3: Add web/src/lib/mathjax.server.ts SSR stub and the conditional @mathjax alias in vite.config.ts <!-- t:pl03 -->
- [ ] Phase 3: Implement enhanceMath in noteEnhancements.ts (idempotent, cancellable, claim-before-await) <!-- t:3aho -->
- [ ] Phase 3: Wire math prop + effect into NoteHtml.tsx and re-run enhanceMath on resolved embeds <!-- t:x7ep -->
- [ ] Phase 3: Add .math / .math-display / state styling to web/src/styles/prose.css <!-- t:77ka -->
- [ ] Phase 4: Plumb math toggle through widget IR props, NoteHtml.widget.tsx, and vaultwidgets vw.noteHtml <!-- t:kijw -->
- [ ] Phase 4: Add vault-example math showcase note, Storybook story, and README documentation <!-- t:zut1 -->
- [ ] Phase 5: Verify MathJax lands in its own lazy chunk; measure bundle size and trim TeX packages if needed <!-- t:w36q -->
- [ ] Phase 5: Manual verification checklist — dark mode, mobile overflow, no-JS fallback, embeds, agent markdown mirror, search <!-- t:nqgf -->
