/**
 * ORGANISM: BacklinksPanel
 * Design: Retro System 1 — window panel listing notes that link to the current note.
 */
import React from "react";
import { clsx } from "clsx";
import { BacklinkItem } from "../../molecules/BacklinkItem/BacklinkItem";
import { Divider } from "../../atoms/Divider/Divider";
import { Badge } from "../../atoms/Badge/Badge";
import { ScrollArea } from "../../atoms/ScrollArea/ScrollArea";
import { Icon } from "../../atoms/Icon/Icon";

export interface BacklinkEntry {
  slug: string;
  title: string;
  excerpt?: string;
}

export interface BacklinksPanelProps {
  backlinks: BacklinkEntry[];
  onNavigate: (slug: string) => void;
  className?: string;
  maxHeight?: string;
}

export const BacklinksPanel: React.FC<BacklinksPanelProps> = ({
  backlinks,
  onNavigate,
  className,
  maxHeight = "300px",
}) => {
  if (backlinks.length === 0) {
    return (
      <div className={clsx("text-[11px] text-[var(--color-muted-foreground)] italic px-1", className)}>
        No backlinks
      </div>
    );
  }

  return (
    <div className={clsx("retro-window", className)}>
      <div className="retro-window-title">
        <Icon name="link" size={11} />
        <span>Linked mentions</span>
        <Badge className="ml-1">{backlinks.length}</Badge>
        <div className="retro-titlebar-stripes" />
      </div>
      <ScrollArea maxHeight={maxHeight}>
        {backlinks.map((bl, i) => (
          <React.Fragment key={bl.slug}>
            <BacklinkItem
              slug={bl.slug}
              title={bl.title}
              excerpt={bl.excerpt}
              onClick={onNavigate}
            />
          </React.Fragment>
        ))}
      </ScrollArea>
    </div>
  );
};
