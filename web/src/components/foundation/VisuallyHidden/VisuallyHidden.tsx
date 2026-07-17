/**
 * FOUNDATION: VisuallyHidden
 * Screen-reader-only content; visually removed but kept in the a11y tree.
 */
import React from "react";

export interface VisuallyHiddenProps {
  children: React.ReactNode;
}

export const VisuallyHidden: React.FC<VisuallyHiddenProps> = ({ children }) => (
  <span className="sr-only">{children}</span>
);
