import { Provider } from "react-redux";
import { Route, Switch, useLocation } from "wouter";
import { store } from "./store/store";
import { VaultLayout } from "./components/pages/VaultLayout/VaultLayout";
import { NotePage } from "./components/pages/NotePage/NotePage";
import { SearchPage } from "./components/pages/SearchPage/SearchPage";
import { Icon } from "./components/atoms/Icon/Icon";
import { useListNotesQuery, type NoteListItem } from "./store/vaultApi";

function Router() {
  return (
    <VaultLayout>
      <Switch>
        <Route path="/" component={HomeRedirect} />
        <Route path="/note/*" component={NoteRoute} />
        <Route path="/search" component={SearchRoute} />
        <Route component={NotFoundPage} />
      </Switch>
    </VaultLayout>
  );
}

function HomeRedirect() {
  const { data: notes, isLoading, isError } = useListNotesQuery();
  const homeSlug = chooseHomeSlug(notes ?? []);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full gap-2 text-[var(--color-muted-foreground)] text-xs">
        <Icon name="file" size={14} className="animate-pulse" />
        Loading vault…
      </div>
    );
  }

  if (isError || !homeSlug) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-2 text-[var(--color-destructive-accent)]">
        <Icon name="alert" size={24} />
        <p className="text-sm font-bold">No home note found</p>
        <p className="text-xs text-[var(--color-muted-foreground)]">The vault did not return any notes.</p>
      </div>
    );
  }

  return <NotePage slug={homeSlug} />;
}

function chooseHomeSlug(notes: NoteListItem[]): string | undefined {
  if (notes.length === 0) return undefined;

  const normalized = notes.map((note) => ({
    note,
    slug: note.slug.toLowerCase(),
    title: note.title.toLowerCase(),
    path: note.path.toLowerCase(),
  }));

  const preferredHomeSlugs = [
    "index",
    "home",
    "readme",
    "projects/00-project-index-repos-and-concepts",
    "research/institute/guidelines/guidelines-index",
  ];

  const eligibleIndexes = normalized
    .filter(({ slug, path }) => (slug === "index" || slug.endsWith("/index")) && !slug.includes("/sources/") && !path.includes("/sources/"))
    .sort((a, b) => indexScore(a) - indexScore(b));

  return (
    preferredHomeSlugs.map((candidate) => normalized.find(({ slug }) => slug === candidate)?.note.slug).find(Boolean) ??
    normalized.find(({ title }) => ["index", "home", "readme"].includes(title))?.note.slug ??
    normalized.find(({ path }) => path === "index.md" || path === "home.md" || path === "readme.md")?.note.slug ??
    eligibleIndexes[0]?.note.slug ??
    normalized.find(({ slug }) => slug.includes("project-index"))?.note.slug ??
    notes[0].slug
  );
}

function indexScore(item: { slug: string; title: string; path: string }) {
  const depth = item.slug.split("/").length;
  const sourcePenalty = item.slug.includes("/sources/") || item.path.includes("/sources/") ? 1000 : 0;
  const titlePenalty = item.title === "index" ? 0 : 10;
  return sourcePenalty + depth + titlePenalty;
}

function NoteRoute(props: { params?: Record<string, string | undefined> }) {
  // regexparam (used by Wouter) uses "*" as the key for wildcard segments
  const raw = props.params?.["*"] ?? "";
  const slug = decodeURIComponent(raw);
  return slug ? <NotePage slug={slug} /> : <HomeRedirect />;
}

function SearchRoute() {
  return <SearchPage />;
}

function NotFoundPage() {
  return (
    <div className="p-8 note-prose">
      <h1>404 — Note not found</h1>
      <p>The note you are looking for does not exist in this vault.</p>
      <p>
        <a href="/" className="wiki-link">
          Return to Index
        </a>
      </p>
    </div>
  );
}

export default function App() {
  return (
    <Provider store={store}>
      <Router />
    </Provider>
  );
}
