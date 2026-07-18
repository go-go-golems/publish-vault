import {
  Suspense,
  useEffect,
  useState,
  type ComponentType,
  type ReactNode,
} from "react";
import { Provider } from "react-redux";
import {
  BrowserRouter,
  Route,
  Routes,
  useLocation,
  useParams,
} from "react-router-dom";
import { store } from "./store/store";
import { VaultLayout } from "./components/pages/VaultLayout/VaultLayout";
import type { NotePageProps } from "./components/pages/NotePage/NotePage";
import { SearchPage } from "./components/pages/SearchPage/SearchPage";
import { WidgetPage } from "./components/pages/WidgetPage/WidgetPage";
import { SidebarSlotProvider } from "./widgets/sidebarSlot";
import { Icon } from "./components/atoms/Icon/Icon";
import {
  useListNotesQuery,
  useGetConfigQuery,
  type NoteListItem,
} from "./store/vaultApi";

export interface AppRoutesProps {
  NotePageComponent: ComponentType<NotePageProps>;
  initialHomeSlug?: string;
  suspendNotePage?: boolean;
}

export function NotePageFallback() {
  return (
    <div className="flex items-center justify-center h-full gap-2 text-[var(--color-muted-foreground)] text-xs">
      <Icon name="file" size={14} className="animate-pulse" />
      Loading note…
    </div>
  );
}

export function AppRoutes({
  NotePageComponent,
  initialHomeSlug,
  suspendNotePage = false,
}: AppRoutesProps) {
  const { data: config } = useGetConfigQuery();
  const location = useLocation();
  // Widget pages can declare an app shell whose sidebar replaces the vault
  // tree (see widgets/sidebarSlot.tsx).
  const [sidebarOverride, setSidebarOverride] = useState<ReactNode | null>(null);

  useEffect(() => {
    if (
      location.pathname === "/" ||
      location.pathname.startsWith("/note/") ||
      location.pathname.startsWith("/w/") ||
      location.pathname === "/search"
    )
      return;
    document.title =
      config?.pageTitle || config?.vaultName || "Retro Obsidian Publish";
  }, [config?.pageTitle, config?.vaultName, location.pathname]);

  const routes = (
    <Routes>
      <Route
        path="/"
        element={
          <HomeRedirect
            NotePageComponent={NotePageComponent}
            initialHomeSlug={initialHomeSlug}
          />
        }
      />
      <Route
        path="/note/*"
        element={<NoteRoute NotePageComponent={NotePageComponent} />}
      />
      <Route path="/search" element={<SearchRoute />} />
      <Route path="/w/:pageId" element={<WidgetRoute />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  );

  return (
    <SidebarSlotProvider value={setSidebarOverride}>
      <VaultLayout
        vaultName={config?.vaultName}
        sidebar={sidebarOverride ?? undefined}
      >
        {suspendNotePage ? (
          <Suspense fallback={<NotePageFallback />}>{routes}</Suspense>
        ) : (
          routes
        )}
      </VaultLayout>
    </SidebarSlotProvider>
  );
}

function HomeRedirect({ NotePageComponent, initialHomeSlug }: AppRoutesProps) {
  const {
    data: notes,
    isLoading,
    isError,
  } = useListNotesQuery(undefined, {
    skip: Boolean(initialHomeSlug),
  });
  const homeSlug = initialHomeSlug ?? chooseHomeSlug(notes ?? []);

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
        <p className="text-xs text-[var(--color-muted-foreground)]">
          The vault did not return any notes.
        </p>
      </div>
    );
  }

  return <NotePageComponent slug={homeSlug} />;
}

function chooseHomeSlug(notes: NoteListItem[]): string | undefined {
  if (notes.length === 0) return undefined;

  const normalized = notes.map(note => ({
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
    .filter(
      ({ slug, path }) =>
        (slug === "index" || slug.endsWith("/index")) &&
        !slug.includes("/sources/") &&
        !path.includes("/sources/")
    )
    .sort((a, b) => indexScore(a) - indexScore(b));

  return (
    preferredHomeSlugs
      .map(
        candidate =>
          normalized.find(({ slug }) => slug === candidate)?.note.slug
      )
      .find(Boolean) ??
    normalized.find(({ title }) => ["index", "home", "readme"].includes(title))
      ?.note.slug ??
    normalized.find(
      ({ path }) =>
        path === "index.md" || path === "home.md" || path === "readme.md"
    )?.note.slug ??
    eligibleIndexes[0]?.note.slug ??
    normalized.find(({ slug }) => slug.includes("project-index"))?.note.slug ??
    notes[0].slug
  );
}

function indexScore(item: { slug: string; title: string; path: string }) {
  const depth = item.slug.split("/").length;
  const sourcePenalty =
    item.slug.includes("/sources/") || item.path.includes("/sources/")
      ? 1000
      : 0;
  const titlePenalty = item.title === "index" ? 0 : 10;
  return sourcePenalty + depth + titlePenalty;
}

function NoteRoute({ NotePageComponent }: AppRoutesProps) {
  // React Router uses "*" as the key for wildcard path segments.
  const raw = useParams()["*"] ?? "";
  const slug = decodeURIComponent(raw);
  return slug ? (
    <NotePageComponent slug={slug} />
  ) : (
    <HomeRedirect NotePageComponent={NotePageComponent} />
  );
}

function SearchRoute() {
  return <SearchPage />;
}

function WidgetRoute() {
  const pageId = useParams().pageId ?? "";
  return <WidgetPage pageId={pageId} />;
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

export default function App({
  NotePageComponent,
  initialHomeSlug,
  suspendNotePage,
}: AppRoutesProps) {
  return (
    <Provider store={store}>
      <BrowserRouter>
        <AppRoutes
          NotePageComponent={NotePageComponent}
          initialHomeSlug={initialHomeSlug}
          suspendNotePage={suspendNotePage}
        />
      </BrowserRouter>
    </Provider>
  );
}
