import { Provider } from "react-redux";
import { Route, Switch, useLocation } from "wouter";
import { store } from "./store/store";
import { VaultLayout } from "./components/pages/VaultLayout/VaultLayout";
import { NotePage } from "./components/pages/NotePage/NotePage";
import { SearchPage } from "./components/pages/SearchPage/SearchPage";

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
  return <NotePage slug="index" />;
}

function NoteRoute(props: { params?: Record<string, string | undefined> }) {
  // regexparam (used by Wouter) uses "*" as the key for wildcard segments
  const raw = props.params?.["*"] ?? "index";
  const slug = decodeURIComponent(raw);
  return <NotePage slug={slug} />;
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
