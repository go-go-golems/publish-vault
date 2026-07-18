/**
 * WidgetRenderer — generic Widget IR → React renderer.
 * Ported from rag-evaluation-system packages/rag-evaluation-site
 * src/widgets/WidgetRenderer.tsx (v0.1.7); UnknownWidget restyled as a retro
 * callout instead of the rag ErrorCallout atom.
 */
import {
  type CSSProperties,
  createElement,
  Fragment,
  type ReactNode,
  useEffect,
  useState,
} from "react";
import { bindAction, dispatchWidgetAction, type WidgetActionHandler } from "./actions";
import type { ComponentNode, WidgetNode } from "./ir";
import type { RenderContext, WidgetRegistry } from "./registry";

export interface WidgetRendererProps {
  node: WidgetNode;
  registry: WidgetRegistry;
  onAction?: WidgetActionHandler;
}

export function WidgetRenderer({ node, registry, onAction }: WidgetRendererProps) {
  const ctx = createRenderContext(registry, onAction);
  return <>{renderWidgetNode(node, ctx, registry)}</>;
}

export function WidgetToastRegion() {
  const [toast, setToast] = useState<{ message: string; tone?: string }>();
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | undefined;
    const listener = (event: Event) => {
      const detail = (event as CustomEvent<{ message?: string; toast?: string; tone?: string }>)
        .detail;
      const message = detail.message ?? detail.toast;
      if (!message) return;
      setToast({ message, tone: detail.tone });
      if (timer) clearTimeout(timer);
      timer = setTimeout(() => setToast(undefined), 4000);
    };
    window.addEventListener("widget:toast", listener);
    return () => {
      window.removeEventListener("widget:toast", listener);
      if (timer) clearTimeout(timer);
    };
  }, []);
  if (!toast) return null;
  return (
    <div
      aria-live="polite"
      aria-atomic="true"
      role="status"
      data-widget-toast-tone={toast.tone}
      className="retro-window px-3 py-2 text-xs font-bold"
      style={{ position: "fixed", right: 16, bottom: 16, zIndex: 1000, maxWidth: 420 }}
    >
      {toast.message}
    </div>
  );
}

function createRenderContext(
  registry: WidgetRegistry,
  onAction?: WidgetActionHandler
): RenderContext {
  const ctx: RenderContext = {
    renderNode: node => renderWidgetNode(node, ctx, registry),
    renderChildren: children => renderChildren(children, ctx, registry),
    renderValue: value => renderRenderableValue(value, ctx, registry),
    bindAction: (action, context) => bindAction(action, context, onAction),
    dispatchAction: (action, context) => dispatchWidgetAction(action, context, onAction),
  };
  return ctx;
}

function renderWidgetNode(
  node: WidgetNode,
  ctx: RenderContext,
  registry: WidgetRegistry
): ReactNode {
  switch (node.kind) {
    case "text":
      return node.text;
    case "element":
      return renderElementNode(node, ctx, registry);
    case "component":
      return renderComponentNode(node, ctx, registry);
    default:
      return null;
  }
}

function renderElementNode(
  node: Extract<WidgetNode, { kind: "element" }>,
  ctx: RenderContext,
  registry: WidgetRegistry
): ReactNode {
  const attrs = node.attrs ?? {};
  const children = renderChildren(node.children, ctx, registry);
  return createElement(
    node.tag,
    { ...attrs, style: attrs.style as CSSProperties | undefined },
    children
  );
}

function renderComponentNode(
  node: ComponentNode,
  ctx: RenderContext,
  registry: WidgetRegistry
): ReactNode {
  const adapter = registry.get(node.type);
  if (!adapter) {
    return <UnknownWidget node={node} />;
  }
  return adapter.render(node.props ?? {}, renderChildren(node.children, ctx, registry), ctx, node);
}

function renderChildren(
  children: WidgetNode[] | undefined,
  ctx: RenderContext,
  registry: WidgetRegistry
): ReactNode[] {
  return (children ?? []).map((child, index) => (
    <Fragment key={widgetNodeKey(child, index)}>{renderWidgetNode(child, ctx, registry)}</Fragment>
  ));
}

function widgetNodeKey(node: WidgetNode, index: number): string | number {
  if (node.kind === "component") {
    const props = node.props ?? {};
    if (typeof props.key === "string" || typeof props.key === "number") return props.key;
    if (typeof props.id === "string" || typeof props.id === "number") return props.id;
    return `${node.type}-${index}`;
  }
  if (node.kind === "element") {
    const attrs = node.attrs ?? {};
    if (typeof attrs.key === "string" || typeof attrs.key === "number") return attrs.key;
    if (typeof attrs.id === "string" || typeof attrs.id === "number") return attrs.id;
    return `${node.tag}-${index}`;
  }
  return index;
}

function renderRenderableValue(
  value: unknown,
  ctx: RenderContext,
  registry: WidgetRegistry
): ReactNode {
  if (value && typeof value === "object" && "kind" in value) {
    return renderWidgetNode(value as WidgetNode, ctx, registry);
  }
  return value == null ? null : String(value);
}

function UnknownWidget({ node }: { node: ComponentNode }) {
  return (
    <div className="callout callout-important">
      <div className="callout-title">
        <span className="callout-icon">⚠</span> Unknown widget
      </div>
      <div className="callout-body">
        Widget type <code>{node.type}</code> is not registered.
      </div>
    </div>
  );
}
