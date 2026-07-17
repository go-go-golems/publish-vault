import type { PanelWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

const TONE_CLASSES: Record<string, string> = {
  default: "callout-note",
  success: "callout-summary",
  warning: "callout-warning",
  danger: "callout-important",
  info: "callout-info",
};

// IR Panels carry a title + tone (widget.ui.callout lowers to this); render
// with the retro callout skin rather than the plain layout Panel so tones
// stay visually meaningful.
export const panelWidget = defineWidget<PanelWidgetProps>({
  type: "Panel",
  module: "widget.dsl",
  render: (props, children) => (
    <div className={`callout ${TONE_CLASSES[props.tone ?? "default"]} ${props.className ?? ""}`}>
      {props.title && <div className="callout-title">{props.title}</div>}
      <div className="callout-body">{children}</div>
    </div>
  ),
});
