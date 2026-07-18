import { FrontmatterPanel } from "./FrontmatterPanel";
import type { FrontmatterPanelWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

export const frontmatterPanelWidget = defineWidget<FrontmatterPanelWidgetProps>({
  type: "FrontmatterPanel",
  module: "widget.dsl",
  render: (props, _children, ctx) => (
    <FrontmatterPanel
      frontmatter={props.frontmatter}
      tags={props.tags}
      modTime={props.modTime}
      className={props.className}
      onTagClick={tag =>
        ctx.dispatchAction(
          props.onTagClick ?? {
            kind: "navigate",
            to: `/search?q=%23${encodeURIComponent(tag)}`,
          },
          { tag, componentType: "FrontmatterPanel" }
        )
      }
    />
  ),
});
