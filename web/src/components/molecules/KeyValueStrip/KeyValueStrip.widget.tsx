import { KeyValueStrip } from "./KeyValueStrip";
import type { KeyValueStripWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

export const keyValueStripWidget = defineWidget<KeyValueStripWidgetProps>({
  type: "KeyValueStrip",
  module: "widget.dsl",
  render: (props, _children, ctx) => (
    <KeyValueStrip
      className={props.className}
      items={props.items.map((item, index) => ({
        key: item.key?.kind === "text" ? item.key.text : String(index),
        label: ctx.renderNode(item.label),
        value: ctx.renderNode(item.value),
      }))}
    />
  ),
});
