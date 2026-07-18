import { Caption } from "./Caption";
import type { CaptionWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

export const captionWidget = defineWidget<CaptionWidgetProps>({
  type: "Caption",
  module: "widget.dsl",
  render: (props, children) => <Caption className={props.className}>{children}</Caption>,
});
