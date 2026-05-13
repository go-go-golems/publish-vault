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

  return (
    normalized.find(({ slug }) => slug === "index")?.note.slug ??
    normalized.find(({ slug }) => slug.endsWith("/index"))?.note.slug ??
    normalized.find(({ title }) => title === "index")?.note.slug ??
    normalized.find(({ path }) => path.endsWith("/index.md") || path === "index.md")?.note.slug ??
    normalized.find(({ slug, title, path }) => slug.includes("index") || title.includes("index") || path.includes("index"))?.note.slug ??
    notes[0].slug
  );
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
