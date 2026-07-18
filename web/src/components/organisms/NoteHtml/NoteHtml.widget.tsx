import { NoteHtml } from "./NoteHtml";
import type { NoteHtmlWidgetProps } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";

// Wiki-link clicks become dispatched navigate actions so the page's
// onAction owns link behavior (IR-consistent, overridable).
export const noteHtmlWidget = defineWidget<NoteHtmlWidgetProps>({
  type: "NoteHtml",
  module: "widget.dsl",
  render: (props, _children, ctx) => (
    <NoteHtml
      html={props.html}
      slug={props.slug}
      mermaid={props.mermaid}
      highlight={props.highlight}
      embeds={props.embeds}
      anchors={props.anchors}
      onWikiLinkNavigate={slug =>
        ctx.dispatchAction(
          { kind: "navigate", to: `/note/${slug}` },
          { slug, componentType: "NoteHtml" }
        )
      }
    />
  ),
});
