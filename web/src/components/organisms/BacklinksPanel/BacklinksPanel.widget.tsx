import { BacklinksPanel } from "./BacklinksPanel";
import type { BacklinksPanelWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

export const backlinksPanelWidget = defineWidget<BacklinksPanelWidgetProps>({
  type: "BacklinksPanel",
  module: "widget.dsl",
  render: (props, _children, ctx) => (
    <BacklinksPanel
      backlinks={props.entries}
      className={props.className}
      onNavigate={slug =>
        ctx.dispatchAction(
          props.onSelect ?? { kind: "navigate", to: `/note/${slug}` },
          { slug, componentType: "BacklinksPanel" }
        )
      }
    />
  ),
});
