/**
 * ORGANISM PART: NoteBody
 * The dangerouslySetInnerHTML host for server-rendered note HTML — and
 * nothing else. Enhancement effects and click delegation live in the owner
 * (NoteView / NoteHtml); this component only owns the container element.
 *
 * memo() is load-bearing, not an optimization: React 19 re-applies
 * dangerouslySetInnerHTML on every re-render (the {__html} object is new
 * each time), which resets innerHTML and silently destroys everything the
 * enhancement pipeline injected (heading anchors, copy buttons, mermaid
 * SVGs, resolved embeds) without the [resolvedHtml]-keyed effects re-running.
 * Memoizing on the html string means the DOM is only rewritten when the
 * content actually changes — which is exactly when the effects re-run.
 */
import React, { forwardRef, memo } from "react";

export interface NoteBodyProps {
  html: string;
}

export const NoteBody = memo(
  forwardRef<HTMLDivElement, NoteBodyProps>(({ html }, ref) => (
    <div
      ref={ref}
      className="note-prose"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  ))
);

NoteBody.displayName = "NoteBody";
