import { Text } from "./Text";
import type { TextWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

export const textWidget = defineWidget<TextWidgetProps>({
  type: "Text",
  module: "widget.dsl",
  render: (props, children) => (
    <Text tone={props.tone ?? "default"} className={props.className}>
      {children}
    </Text>
  ),
});
