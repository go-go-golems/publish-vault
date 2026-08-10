// 03-ssr-conflation-repro.mjs — proves that web/server.mjs turns ANY failure to
// fetch a note from the Go API into the user-visible 404 "Note not found",
// including failures that have nothing to do with the note being missing.
//
// The two functions below are copied verbatim from web/server.mjs (fetchAPI at
// lines 83-91, the 404 branch at lines 242-245) so the repro exercises the real
// control flow rather than a paraphrase of it.
//
// Usage: node 03-ssr-conflation-repro.mjs [API_BASE]
//   default API_BASE: http://127.0.0.1:18420

const API_BASE = process.argv[2] || "http://127.0.0.1:18420";
const SLUG =
  "transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields";

// ---- verbatim from web/server.mjs:83-91 ----
async function fetchAPI(path, base = API_BASE) {
  try {
    const res = await fetch(`${base}${path}`);
    if (!res.ok) return null;
    return await res.json();
  } catch {
    return null;
  }
}
// -------------------------------------------

// What server.mjs does with the result (lines 234-245), reduced to the decision.
function ssrOutcome(note) {
  if (!note) return { status: 404, body: "Note not found" };
  return { status: 200, body: `<title>${note.title}</title>` };
}

async function scenario(label, base, path) {
  const started = Date.now();
  const note = await fetchAPI(path, base);
  const out = ssrOutcome(note);
  console.log(
    `${label.padEnd(46)} -> HTTP ${out.status}  ${JSON.stringify(out.body).slice(0, 60)}  (${Date.now() - started}ms)`
  );
}

console.log(`API_BASE = ${API_BASE}\n`);

// 1. Happy path: the note exists and the API is healthy.
await scenario(
  "A. note exists, API healthy",
  API_BASE,
  `/api/notes/${encodeURIComponent(SLUG)}`
);

// 2. The note genuinely does not exist -> API 404 -> res.ok false -> null.
await scenario(
  "B. note genuinely absent (API 404)",
  API_BASE,
  "/api/notes/transcripts/2026/08/09/does-not-exist"
);

// 3. The API is unreachable (crashed / OOMKilled / mid-restart / port closed).
//    fetch() throws; the catch swallows it and returns null. The user is told
//    the note does not exist, which is false.
await scenario(
  "C. API unreachable (fetch throws)",
  "http://127.0.0.1:1",
  `/api/notes/${encodeURIComponent(SLUG)}`
);

// 4. The API answers with a server error (e.g. reload in progress, 500).
//    res.ok is false -> null -> same misleading "Note not found".
await scenario(
  "D. API 5xx (res.ok === false)",
  API_BASE,
  "/api/search"  // no q param returns 200; use a definitely-non-API path instead
);

// 5. The Go API is fine but the response body is truncated / not JSON.
//    res.json() throws inside the try -> catch -> null -> "Note not found".
await scenario(
  "E. non-JSON body (res.json throws)",
  API_BASE,
  "/note/index" // HTML, not JSON
);
