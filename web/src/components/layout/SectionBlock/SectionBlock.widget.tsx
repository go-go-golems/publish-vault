import { SectionBlock } from "./SectionBlock";
import type { SectionBlockWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

// IR uses `label`; the React component names it `title`. `level`, `density`
// and `rule` are accepted but currently rendered with the single retro
// section style.
//
// v3 wraps page-level `.view()` nodes in an auto-section labeled "Content"
// (structural filler, not an authored heading). Render those unlabeled so
// widget pages keep the article layout instead of repeating "Content"
// rules between every page-level view.
export const sectionBlockWidget = defineWidget<SectionBlockWidgetProps>({
  type: "SectionBlock",
  module: "widget.dsl",
  render: (props, children) => {
    if (props.label === "Content" && !props.caption) {
      return <section className={props.className}>{children}</section>;
    }
    return (
      <SectionBlock title={props.label} caption={props.caption} className={props.className}>
        {children}
      </SectionBlock>
    );
  },
});
