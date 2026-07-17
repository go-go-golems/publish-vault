/**
 * ORGANISM PART: NoteBody
 * The dangerouslySetInnerHTML host for server-rendered note HTML — and
 * nothing else. Enhancement effects and click delegation live in NoteView;
 * this component only owns the container element.
 */
import React, { forwardRef } from "react";

export interface NoteBodyProps {
  html: string;
}

export const NoteBody = forwardRef<HTMLDivElement, NoteBodyProps>(
  ({ html }, ref) => (
    <div
      ref={ref}
      className="note-prose"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  )
);

NoteBody.displayName = "NoteBody";
