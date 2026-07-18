/**
 * Sidebar slot plumbing: lets a routed widget page inject a JS-declared
 * sidebar into VaultLayout (which wraps the routes above them).
 *
 * AppRoutes owns the state and passes the current override to VaultLayout's
 * `sidebar` prop; WidgetPage sets/clears the override via context based on
 * the resolved page shell.
 */
import {
  createContext,
  useContext,
  useEffect,
  type Dispatch,
  type ReactNode,
  type SetStateAction,
} from "react";

export type SidebarOverrideSetter = Dispatch<SetStateAction<ReactNode | null>>;

const SidebarSlotContext = createContext<SidebarOverrideSetter | null>(null);

export const SidebarSlotProvider = SidebarSlotContext.Provider;

/**
 * Sets the VaultLayout sidebar override while the calling component is
 * mounted with a non-null element; clears it on unmount/change.
 */
export function useSidebarOverride(element: ReactNode | null): void {
  const setOverride = useContext(SidebarSlotContext);
  useEffect(() => {
    if (!setOverride || element == null) return;
    setOverride(element);
    return () => setOverride(null);
  }, [setOverride, element]);
}
