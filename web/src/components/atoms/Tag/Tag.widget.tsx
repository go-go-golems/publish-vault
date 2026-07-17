import { Tag } from "./Tag";
import type { TagWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

// Demonstrates adapting an existing publish-vault atom to the widget IR:
// the registry decouples IR types from any one component library.
export const tagWidget = defineWidget<TagWidgetProps>({
  type: "Tag",
  module: "widget.dsl",
  render: (props, children) => (
    <Tag
      label={props.label ?? children.map(child => String(child)).join("")}
      className={props.className}
    />
  ),
});
