/**
 * MOLECULE: FileTreeItem
 * Design: Retro System 1 — compact tree node with folder/file icon, indent, expand toggle.
 */
import React, { useState } from "react";
import { clsx } from "clsx";
import { Icon } from "../../atoms/Icon/Icon";
import type { FileNode } from "../../../types";

export interface FileTreeItemProps {
  node: FileNode;
  depth?: number;
  activeSlug?: string;
  onSelect?: (slug: string) => void;
}

export const FileTreeItem: React.FC<FileTreeItemProps> = ({
  node,
  depth = 0,
  activeSlug,
  onSelect,
}) => {
  const [expanded, setExpanded] = useState(depth < 2);

  if (node.isFolder) {
    return (
      <div>
        <button
          type="button"
          className={clsx(
            "retro-tree-item w-full text-left",
            "hover:bg-[var(--color-ink)] hover:text-[var(--color-paper)]"
          )}
          style={{ paddingLeft: `${8 + depth * 14}px` }}
          onClick={() => setExpanded((v) => !v)}
        >
          <Icon
            name={expanded ? "chevron-down" : "chevron-right"}
            size={11}
            className="shrink-0 text-[var(--color-muted-foreground)]"
          />
          <Icon
            name={expanded ? "folder-open" : "folder"}
            size={12}
            className="shrink-0"
          />
          <span className="truncate font-bold">{node.name}</span>
        </button>
        {expanded && node.children?.map((child) => (
          <FileTreeItem
            key={child.path}
            node={child}
            depth={depth + 1}
            activeSlug={activeSlug}
            onSelect={onSelect}
          />
        ))}
      </div>
    );
  }

  const isActive = node.slug === activeSlug;
  return (
    <button
      type="button"
      className={clsx("retro-tree-item w-full text-left", isActive && "active")}
      style={{ paddingLeft: `${8 + depth * 14 + 14}px` }}
      onClick={() => node.slug && onSelect?.(node.slug)}
    >
      <Icon name="file" size={11} className="shrink-0 opacity-60" />
      <span className="truncate">{node.name}</span>
    </button>
  );
};
