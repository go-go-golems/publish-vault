import { Stack } from "./Stack";
import type { StackWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

export const stackWidget = defineWidget<StackWidgetProps>({
  type: "Stack",
  module: "widget.dsl",
  render: (props, children) => (
    <Stack gap={props.gap ?? "md"} className={props.className}>
      {children}
    </Stack>
  ),
});
