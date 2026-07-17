import { Divider } from "./Divider";
import type { DividerWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

export const dividerWidget = defineWidget<DividerWidgetProps>({
  type: "Divider",
  module: "widget.dsl",
  render: props => <Divider label={props.label} className={props.className} />,
});
