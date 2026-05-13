/**
 * ORGANISM: Sidebar
 * Design: Retro System 1 — fixed-width panel with window chrome, file tree, search.
 * Integrates RTK Query for tree data; search is passed as callback.
 */
import React, { useState } from "react";
import { clsx } from "clsx";
import { SearchBar } from "../../molecules/SearchBar/SearchBar";
import { FileTreeItem } from "../../molecules/FileTreeItem/FileTreeItem";
import { ScrollArea } from "../../atoms/ScrollArea/ScrollArea";
import { Divider } from "../../atoms/Divider/Divider";
import { Icon } from "../../atoms/Icon/Icon";
import type { FileNode } from "../../../types";

export interface SidebarProps {
  tree: FileNode | null;
  activeSlug?: string;
  onSelectNote: (slug: string) => void;
  onSearch: (query: string) => void;
  vaultName?: string;
  isLoading?: boolean;
  className?: string;
}

export const Sidebar: React.FC<SidebarProps> = ({
  tree,
  activeSlug,
  onSelectNote,
  onSearch,
  vaultName = "Vault",
  isLoading,
  className,
}) => {
  return (
    <aside
      className={clsx(
        "retro-window flex flex-col h-full",
        "shrink-0",
        className
      )}
    >
      {/* Title bar */}
      <div className="retro-window-title">
        <Icon name="book" size={11} />
        <span className="truncate">{vaultName}</span>
        <div className="retro-titlebar-stripes" />
      </div>

      {/* Search */}
      <div className="p-2 border-b border-[var(--color-ink)]">
        <SearchBar onSearch={onSearch} />
      </div>

      {/* File tree */}
      <ScrollArea className="flex-1 py-1">
        {isLoading ? (
          <div className="flex items-center gap-2 px-3 py-2 text-[11px] text-[var(--color-muted-foreground)]">
            <Icon name="file" size={11} className="animate-pulse" />
            Loading vault…
          </div>
        ) : tree ? (
          tree.children?.map((node) => (
            <FileTreeItem
              key={node.path}
              node={node}
              depth={0}
              activeSlug={activeSlug}
              onSelect={onSelectNote}
            />
          ))
        ) : (
          <div className="px-3 py-2 text-[11px] text-[var(--color-muted-foreground)] italic">
            No notes found
          </div>
        )}
      </ScrollArea>

      {/* Footer */}
      <div className="border-t border-[var(--color-ink)] px-2 py-1 flex items-center gap-1 text-[10px] text-[var(--color-muted-foreground)]">
        <Icon name="book" size={10} />
        <span>Obsidian Publish</span>
      </div>
    </aside>
  );
};
