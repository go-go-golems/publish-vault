import hljs from "highlight.js/lib/core";
import type { LanguageFn } from "highlight.js";

type LanguageModule = { default: LanguageFn };
type LanguageLoader = () => Promise<LanguageModule>;

const languageModules = import.meta.glob<LanguageModule>(
  "../vendor/highlight-languages/*.ts"
);

const languageFiles: Record<string, string> = {
  bash: "../vendor/highlight-languages/bash.ts",
  css: "../vendor/highlight-languages/css.ts",
  go: "../vendor/highlight-languages/go.ts",
  javascript: "../vendor/highlight-languages/javascript.ts",
  json: "../vendor/highlight-languages/json.ts",
  markdown: "../vendor/highlight-languages/markdown.ts",
  python: "../vendor/highlight-languages/python.ts",
  sql: "../vendor/highlight-languages/sql.ts",
  typescript: "../vendor/highlight-languages/typescript.ts",
  xml: "../vendor/highlight-languages/xml.ts",
  yaml: "../vendor/highlight-languages/yaml.ts",
};

const aliases: Record<string, string> = {
  bash: "bash",
  cjs: "javascript",
  css: "css",
  htm: "xml",
  html: "xml",
  javascript: "javascript",
  javaScript: "javascript",
  js: "javascript",
  json: "json",
  markdown: "markdown",
  md: "markdown",
  mjs: "javascript",
  py: "python",
  python: "python",
  sh: "bash",
  shell: "bash",
  sql: "sql",
  ts: "typescript",
  tsx: "typescript",
  typescript: "typescript",
  xml: "xml",
  xhtml: "xml",
  yaml: "yaml",
  yml: "yaml",
  go: "go",
};

// Unlabelled blocks used to be auto-detected against highlight.js's full registry.
// Preserve a useful, bounded approximation without shipping every language.
const autoDetectLanguages = [
  "bash",
  "css",
  "go",
  "javascript",
  "json",
  "markdown",
  "python",
  "sql",
  "typescript",
  "xml",
  "yaml",
];

const loadedLanguages = new Set<string>();
const pendingLanguages = new Map<string, Promise<void>>();

function normalizeLanguage(block: HTMLElement): string | undefined {
  const languageClass = Array.from(block.classList).find(
    className =>
      className.startsWith("language-") || className.startsWith("lang-")
  );
  if (!languageClass) return undefined;

  const prefix = languageClass.startsWith("language-") ? "language-" : "lang-";
  const requested = languageClass.slice(prefix.length).toLowerCase();
  return aliases[requested];
}

async function loadLanguage(name: string): Promise<void> {
  if (loadedLanguages.has(name)) return;

  const pending = pendingLanguages.get(name);
  if (pending) return pending;

  const path = languageFiles[name];
  const loader = path
    ? (languageModules[path] as LanguageLoader | undefined)
    : undefined;
  if (!loader) return;

  const promise = loader()
    .then(module => {
      hljs.registerLanguage(name, module.default);
      loadedLanguages.add(name);
    })
    .finally(() => {
      pendingLanguages.delete(name);
    });

  pendingLanguages.set(name, promise);
  return promise;
}

/**
 * Load only the language definitions represented by code blocks under `root`,
 * then highlight those blocks. Language chunks are cached for the SPA session.
 */
export async function highlightCodeBlocks(root: HTMLElement): Promise<void> {
  const blocks = Array.from(
    root.querySelectorAll<HTMLElement>("pre code:not(.language-mermaid)")
  );
  if (blocks.length === 0) return;

  const requestedLanguages = new Set<string>();
  for (const block of blocks) {
    const language = normalizeLanguage(block);
    if (language) {
      requestedLanguages.add(language);
    } else {
      autoDetectLanguages.forEach(name => requestedLanguages.add(name));
    }
  }

  await Promise.all(Array.from(requestedLanguages).map(loadLanguage));

  for (const block of blocks) {
    if (block.dataset.highlighted) continue;

    const language = normalizeLanguage(block);
    if (!language || hljs.getLanguage(language)) {
      hljs.highlightElement(block);
    }
    block.dataset.highlighted = "true";
  }
}
