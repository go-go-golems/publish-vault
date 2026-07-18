import { BreadcrumbBar } from "./BreadcrumbBar";
import type { BreadcrumbBarWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

export const breadcrumbBarWidget = defineWidget<BreadcrumbBarWidgetProps>({
  type: "BreadcrumbBar",
  module: "widget.dsl",
  render: (props, _children, ctx) => (
    <BreadcrumbBar
      segments={props.segments}
      className={props.className}
      onNavigate={
        props.onSelect
          ? slug =>
              ctx.dispatchAction(props.onSelect as NonNullable<typeof props.onSelect>, {
                slug,
                componentType: "BreadcrumbBar",
              })
          : undefined
      }
    />
  ),
});
