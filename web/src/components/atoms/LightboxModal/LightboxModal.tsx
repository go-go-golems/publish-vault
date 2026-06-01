/**
 * ATOM: LightboxModal
 * Full-screen modal for viewing images and mermaid diagrams at maximum size.
 * Uses Radix Dialog for accessibility (focus trap, Escape key, overlay click).
 */
import React, { useCallback } from "react";
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { clsx } from "clsx";

export interface LightboxModalProps {
  /** Whether the modal is open */
  open: boolean;
  /** Callback when open state changes (Escape, overlay click, close button) */
  onOpenChange: (open: boolean) => void;
  /** Image source URL for image mode */
  imageSrc?: string;
  /** Alt text for the image */
  imageAlt?: string;
  /** Raw SVG HTML string for mermaid diagram mode */
  svgHtml?: string;
  /** Optional title shown as visually-hidden accessible label */
  ariaLabel?: string;
}

export const LightboxModal: React.FC<LightboxModalProps> = ({
  open,
  onOpenChange,
  imageSrc,
  imageAlt,
  svgHtml,
  ariaLabel,
}) => {
  const isImage = !!imageSrc;
  const isMermaid = !!svgHtml;

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen) {
        onOpenChange(false);
      }
    },
    [onOpenChange]
  );

  // Determine accessible label
  const label = ariaLabel || (isImage ? imageAlt || "Image viewer" : "Diagram viewer");

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        showCloseButton={true}
        className={
          "fixed inset-0 z-50 flex items-center justify-center " +
          "bg-black/90 border-none rounded-none p-0 gap-0 " +
          // Override default card-like styling
          "shadow-none " +
          // Animation: fade in/out
          "data-[state=open]:animate-in data-[state=closed]:animate-out " +
          "data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 " +
          "data-[state=closed]:zoom-out-100 data-[state=open]:zoom-in-100 " +
          "duration-200"
        }
        // Use inline styles to guarantee full-screen dimensions.
        // Tailwind cn() can lose to responsive variants like sm:max-w-lg
        // from the base DialogContent, and w-screen/h-screen may not
        // resolve correctly in Tailwind v4.
        style={{
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          width: "100vw",
          height: "100vh",
          maxWidth: "none",
          transform: "none",
          translate: "none",
        }}
      >
        {/* Visually-hidden accessible title/description */}
        <DialogTitle className="sr-only">{label}</DialogTitle>
        <DialogDescription className="sr-only">
          Full-screen view. Press Escape or click outside to close.
        </DialogDescription>

        <div className={clsx("lightbox-content w-full h-full flex items-center justify-center p-4")}>
          {isImage && (
            <img
              src={imageSrc}
              alt={imageAlt || ""}
              className="max-w-full max-h-full object-contain select-none"
              draggable={false}
            />
          )}
          {isMermaid && (
            <div
              className="mermaid-display max-w-full max-h-full overflow-auto bg-white/95 p-4 rounded"
              dangerouslySetInnerHTML={{ __html: svgHtml }}
            />
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
};
