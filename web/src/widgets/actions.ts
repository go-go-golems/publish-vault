/**
 * Widget action dispatcher — resolves serialized ActionSpec data into
 * navigation, server round-trips (POST /api/widget/actions/{name}), clipboard,
 * download, and DOM-event behaviors.
 * Ported verbatim from rag-evaluation-system packages/rag-evaluation-site
 * src/widgets/actions.ts (v0.1.7); only the linkButton/toggle types trimmed
 * with the cell set.
 */
import type {
	ActionSpec,
	JsonObject,
	JsonValue,
	PayloadTemplateSpec,
	TemplatePartSpec,
	TemplateSpec,
} from "./ir";

export interface WidgetActionContext {
	row?: JsonObject;
	rowKey?: string;
	value?: string | number | boolean | null;
	componentType?: string;
	[key: string]: unknown;
}

export interface ServerActionResult {
	ok: boolean;
	refresh?: boolean;
	toast?: string;
	patch?: JsonObject;
	data?: JsonObject;
	error?: string;
	fieldErrors?: Record<string, string>;
	undo?: ActionSpec;
}

export type WidgetActionHandler = (
	action: ActionSpec,
	context: WidgetActionContext,
) => void | Promise<void>;

export function confirmWidgetAction(
	action: ActionSpec,
	context: WidgetActionContext = {},
): boolean {
	if (!action.confirm || typeof window === "undefined" || typeof window.confirm !== "function") {
		return true;
	}
	return window.confirm(renderTemplate(action.confirm, context, { encode: false }));
}

export function dispatchWidgetAction(
	action: ActionSpec,
	context: WidgetActionContext = {},
	onAction?: WidgetActionHandler,
): void {
	if (onAction) {
		onAction(action, context);
		return;
	}

	if (!confirmWidgetAction(action, context)) {
		return;
	}

	if (action.kind === "copy") {
		const value =
			action.value ?? (action.field && context.row ? String(context.row[action.field] ?? "") : "");
		if (value) {
			navigator.clipboard?.writeText(value).catch(() => {});
		}
		return;
	}

	if (action.kind === "event") {
		if (action.event === "print") {
			window.print();
			return;
		}
		if (action.event === "fullscreen") {
			const target = document.documentElement;
			if (!document.fullscreenElement) {
				void target.requestFullscreen?.();
			} else {
				void document.exitFullscreen?.();
			}
			return;
		}
		window.dispatchEvent(
			new CustomEvent(action.event, {
				detail: { ...resolveActionPayload(action.detail, context), context },
			}),
		);
		return;
	}

	if (action.kind === "openOverlay" || action.kind === "closeOverlay") {
		window.dispatchEvent(new CustomEvent("widget:overlay", { detail: { action, context } }));
		return;
	}

	if (action.kind === "navigate") {
		const target = structuredNavigationTarget(action, context);
		if (action.replace) window.history.replaceState(action.params ?? {}, "", target);
		else window.history.pushState(action.params ?? {}, "", target);
		window.dispatchEvent(new PopStateEvent("popstate"));
		return;
	}

	if (action.kind === "download") {
		const target = interpolate(action.to, context);
		const anchor = document.createElement("a");
		anchor.href = target;
		anchor.download = "";
		anchor.style.display = "none";
		document.body.appendChild(anchor);
		anchor.click();
		anchor.remove();
		return;
	}

	if (action.kind === "server") {
		void fetch(`/api/widget/actions/${encodeURIComponent(action.name)}`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ payload: resolveActionPayload(action.payload, context), context }),
		}).then(async (response) => {
			const result = (await response.json().catch(() => undefined)) as
				| ServerActionResult
				| undefined;
			window.dispatchEvent(
				new CustomEvent("widget:action-result", {
					detail: { action, context, responseOk: response.ok, result },
				}),
			);
			if (result?.toast || result?.error)
				window.dispatchEvent(
					new CustomEvent("widget:toast", {
						detail: {
							message: result.toast ?? result.error,
							tone: response.ok && result.ok !== false ? "success" : "danger",
						},
					}),
				);
			if (response.ok && result?.refresh) window.dispatchEvent(new PopStateEvent("popstate"));
		});
	}
}

export function structuredNavigationTarget(
	action: Extract<ActionSpec, { kind: "navigate" }>,
	context: WidgetActionContext,
): string {
	const base = new URL(interpolate(action.to, context), window.location.href);
	const next = new URLSearchParams();
	for (const key of action.preserveQuery ?? []) {
		for (const value of new URL(window.location.href).searchParams.getAll(key))
			next.append(key, value);
	}
	for (const [key, raw] of Object.entries(action.query ?? {})) {
		const values = Array.isArray(raw) ? raw : [raw];
		for (const value of values) {
			const resolved = resolveTemplatePartOrLiteral(value, context);
			if (action.omitEmpty && (resolved == null || resolved === "")) continue;
			next.append(key, String(resolved ?? ""));
		}
	}
	if (action.query || action.preserveQuery) base.search = next.toString();
	return `${base.pathname}${base.search}${base.hash}`;
}

export function bindAction(
	action: ActionSpec | undefined,
	context: WidgetActionContext,
	onAction?: WidgetActionHandler,
): (() => void) | undefined {
	if (!action) return undefined;
	return () => {
		if (onAction) {
			onAction(action, context);
			return;
		}
		dispatchWidgetAction(action, context);
	};
}

export function resolveActionPayload(
	payload: JsonObject | PayloadTemplateSpec | undefined,
	context: WidgetActionContext,
): JsonObject {
	if (!payload) return {};
	if (isPayloadTemplate(payload)) {
		const out: JsonObject = {};
		for (const [key, value] of Object.entries(payload.fields)) {
			out[key] = resolveTemplatePartOrLiteral(value, context);
		}
		return out;
	}
	const out: JsonObject = {};
	for (const [key, value] of Object.entries(payload)) {
		out[key] = resolveTemplatePartOrLiteral(value, context);
	}
	return out;
}

// URL targets need encoded values; human-facing text (confirm prompts) must
// stay raw — pass { encode: false } there.
function interpolate(
	template: string,
	context: WidgetActionContext,
	options: { encode?: boolean } = {},
): string {
	const encode = options.encode ?? true;
	return template.replace(
		/\$\{([^}]+)\}|\$([A-Za-z0-9_.-]+)/g,
		(_match, braced: string | undefined, bare: string | undefined) => {
			const path = braced ?? bare ?? "";
			const value = lookupContext(path, context);
			const text = String(value ?? "");
			return encode ? encodeURIComponent(text) : text;
		},
	);
}

function renderTemplate(
	template: string | TemplateSpec,
	context: WidgetActionContext,
	options: { encode?: boolean } = {},
): string {
	if (typeof template === "string") return interpolate(template, context, options);
	return template.parts.map((part) => String(resolveTemplatePart(part, context) ?? "")).join("");
}

function resolveTemplatePartOrLiteral(
	value: TemplatePartSpec | JsonValue,
	context: WidgetActionContext,
): JsonValue {
	if (isTemplatePart(value)) return toJsonValue(resolveTemplatePart(value, context));
	if (isAccessor(value)) return toJsonValue(resolveAccessor(value, context));
	if (isConstBinding(value)) return value.value;
	return value;
}

type AccessorBinding = {
	kind: "accessor";
	mode: "field" | "path" | "map" | "template" | "context";
	field?: string;
	path?: string;
	mapField?: string;
	template?: string;
};

type ConstBinding = { kind: "const"; value: JsonValue };

function isAccessor(value: TemplatePartSpec | JsonValue): value is AccessorBinding {
	return Boolean(
		value &&
			typeof value === "object" &&
			!Array.isArray(value) &&
			"kind" in value &&
			(value as { kind?: unknown }).kind === "accessor",
	);
}

function isConstBinding(value: TemplatePartSpec | JsonValue): value is ConstBinding {
	return Boolean(
		value &&
			typeof value === "object" &&
			!Array.isArray(value) &&
			"kind" in value &&
			"value" in value &&
			(value as { kind?: unknown }).kind === "const",
	);
}

function resolveAccessor(accessor: AccessorBinding, context: WidgetActionContext): unknown {
	const row = context.row ?? {};
	switch (accessor.mode) {
		case "context":
			return lookupContext(accessor.path ?? "", context);
		case "field":
			return lookupContext(accessor.field ?? "", row as WidgetActionContext);
		case "path":
			return lookupContext(accessor.path ?? "", row as WidgetActionContext);
		case "map":
			return lookupContext(accessor.mapField ?? "", row as WidgetActionContext);
		case "template":
			return interpolate(accessor.template ?? "", { ...context, row }, { encode: false });
	}
}

function resolveTemplatePart(part: TemplatePartSpec, context: WidgetActionContext): unknown {
	if (part.kind === "path") return lookupContext(part.path, context);
	if (part.kind === "text") return part.text;
	return part.value;
}

function toJsonValue(value: unknown): JsonValue {
	if (value === undefined) return null;
	if (
		value === null ||
		typeof value === "string" ||
		typeof value === "number" ||
		typeof value === "boolean"
	)
		return value;
	if (Array.isArray(value)) return value.map(toJsonValue);
	if (typeof value === "object") {
		const out: JsonObject = {};
		for (const [key, child] of Object.entries(value as Record<string, unknown>)) {
			out[key] = toJsonValue(child);
		}
		return out;
	}
	return String(value);
}

function isPayloadTemplate(value: JsonObject | PayloadTemplateSpec): value is PayloadTemplateSpec {
	return (
		value.kind === "payloadTemplate" && typeof value.fields === "object" && value.fields !== null
	);
}

function isTemplatePart(value: TemplatePartSpec | JsonValue): value is TemplatePartSpec {
	return Boolean(
		value &&
			typeof value === "object" &&
			"kind" in value &&
			(value.kind === "path" || value.kind === "text" || value.kind === "literal"),
	);
}

function lookupContext(path: string, context: WidgetActionContext): unknown {
	const parts = path.split(".").filter(Boolean);
	let current: unknown = context;
	for (const part of parts) {
		if (!current || typeof current !== "object") return undefined;
		current = (current as Record<string, unknown>)[part];
	}
	return current;
}
