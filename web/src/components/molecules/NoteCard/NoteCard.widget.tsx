import { NoteCard } from "./NoteCard";
import type { NoteCardWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

export const noteCardWidget = defineWidget<NoteCardWidgetProps>({
  type: "NoteCard",
  module: "widget.dsl",
  render: (props, _children, ctx) => (
    <NoteCard
      slug={props.slug}
      title={props.title}
      excerpt={props.excerpt}
      tags={props.tags}
      className={props.className}
      onClick={slug =>
        ctx.dispatchAction(
          props.onSelect ?? { kind: "navigate", to: `/note/${slug}` },
          { slug, componentType: "NoteCard" }
        )
      }
    />
  ),
});
