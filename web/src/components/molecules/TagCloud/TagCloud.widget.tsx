import { TagCloud } from "./TagCloud";
import type { TagCloudWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

export const tagCloudWidget = defineWidget<TagCloudWidgetProps>({
  type: "TagCloud",
  module: "widget.dsl",
  render: (props, _children, ctx) => (
    <TagCloud
      className={props.className}
      tags={props.tags.map(t =>
        typeof t === "string" ? { tag: t, count: 0 } : { tag: t.tag, count: t.count ?? 0 }
      )}
      onTagClick={tag =>
        ctx.dispatchAction(
          props.onSelect ?? {
            kind: "navigate",
            to: `/search?q=%23${encodeURIComponent(tag)}`,
          },
          { tag, componentType: "TagCloud" }
        )
      }
    />
  ),
});
