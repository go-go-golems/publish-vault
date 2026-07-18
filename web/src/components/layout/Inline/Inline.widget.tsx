import { Inline } from "./Inline";
import type { InlineWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

export const inlineWidget = defineWidget<InlineWidgetProps>({
  type: "Inline",
  module: "widget.dsl",
  render: (props, children) => (
    <Inline gap={props.gap ?? "sm"} className={props.className}>
      {children}
    </Inline>
  ),
});
