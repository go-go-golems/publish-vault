import { SectionBlock } from "./SectionBlock";
import type { SectionBlockWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

// IR uses `label`; the React component names it `title`. `level`, `density`
// and `rule` are accepted but currently rendered with the single retro
// section style.
export const sectionBlockWidget = defineWidget<SectionBlockWidgetProps>({
  type: "SectionBlock",
  module: "widget.dsl",
  render: (props, children) => (
    <SectionBlock title={props.label} caption={props.caption} className={props.className}>
      {children}
    </SectionBlock>
  ),
});
