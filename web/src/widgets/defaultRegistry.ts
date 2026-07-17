/**
 * publish-vault's v1 default widget registry (see design doc §7.5).
 * Every component type emitted by the example pages under
 * examples/widget-pages/ must have an adapter here; unknown types render the
 * UnknownWidget callout.
 */
import { tagWidget } from "../components/atoms/Tag/Tag.widget";
import { captionWidget } from "../components/foundation/Caption/Caption.widget";
import { dividerWidget } from "../components/foundation/Divider/Divider.widget";
import { textWidget } from "../components/foundation/Text/Text.widget";
import { inlineWidget } from "../components/layout/Inline/Inline.widget";
import { panelWidget } from "../components/layout/Panel/Panel.widget";
import { sectionBlockWidget } from "../components/layout/SectionBlock/SectionBlock.widget";
import { stackWidget } from "../components/layout/Stack/Stack.widget";
import { dataTableWidget } from "../components/molecules/DataTable/DataTable.widget";
import { keyValueStripWidget } from "../components/molecules/KeyValueStrip/KeyValueStrip.widget";
import { createWidgetRegistry, type WidgetAdapter } from "./registry";

export const defaultWidgetAdapters: readonly WidgetAdapter[] = [
  stackWidget,
  inlineWidget,
  panelWidget,
  sectionBlockWidget,
  dataTableWidget,
  keyValueStripWidget,
  textWidget,
  captionWidget,
  dividerWidget,
  tagWidget,
] as WidgetAdapter[];

export const defaultWidgetRegistry = createWidgetRegistry(defaultWidgetAdapters);
