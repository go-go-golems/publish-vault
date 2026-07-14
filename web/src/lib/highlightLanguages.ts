import hljs from "highlight.js/lib/core";
import type { LanguageFn } from "highlight.js";

type LanguageModule = { default: LanguageFn };
type LanguageLoader = () => Promise<LanguageModule>;

// Keep the imports explicit rather than globbing node_modules. Vite emits one
// browser chunk per dynamic import, and the SSR build can parse this map without
// attempting to evaluate import.meta.glob on the server.
const languageModules: Record<string, () => Promise<LanguageModule>> = {
  bash: () => import("../vendor/highlight-languages/bash"),
  css: () => import("../vendor/highlight-languages/css"),
  go: () => import("../vendor/highlight-languages/go"),
  javascript: () => import("../vendor/highlight-languages/javascript"),
  json: () => import("../vendor/highlight-languages/json"),
  markdown: () => import("../vendor/highlight-languages/markdown"),
  python: () => import("../vendor/highlight-languages/python"),
  sql: () => import("../vendor/highlight-languages/sql"),
  typescript: () => import("../vendor/highlight-languages/typescript"),
  xml: () => import("../vendor/highlight-languages/xml"),
  yaml: () => import("../vendor/highlight-languages/yaml"),
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

  const loader = languageModules[name] as LanguageLoader | undefined;
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
