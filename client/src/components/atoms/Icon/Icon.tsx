/**
 * ATOM: Icon
 * Design: Retro System 1 — pixel-crisp SVG icons, monochrome.
 * Uses a subset of Lucide icons with retro sizing.
 */
import React from "react";
import { clsx } from "clsx";
import {
  FileText,
  Folder,
  FolderOpen,
  Search,
  ChevronRight,
  ChevronDown,
  Link,
  Hash,
  ArrowLeft,
  ArrowRight,
  X,
  Menu,
  Network,
  BookOpen,
  Clock,
  AlertCircle,
  Check,
  ExternalLink,
  Home,
} from "lucide-react";

export const ICON_MAP = {
  file: FileText,
  folder: Folder,
  "folder-open": FolderOpen,
  search: Search,
  "chevron-right": ChevronRight,
  "chevron-down": ChevronDown,
  link: Link,
  hash: Hash,
  "arrow-left": ArrowLeft,
  "arrow-right": ArrowRight,
  close: X,
  menu: Menu,
  graph: Network,
  book: BookOpen,
  clock: Clock,
  alert: AlertCircle,
  check: Check,
  "external-link": ExternalLink,
  home: Home,
} as const;

export type IconName = keyof typeof ICON_MAP;

export interface IconProps {
  name: IconName;
  size?: number;
  className?: string;
  strokeWidth?: number;
}

export const Icon: React.FC<IconProps> = ({
  name,
  size = 14,
  className,
  strokeWidth = 1.5,
}) => {
  const LucideIcon = ICON_MAP[name];
  return (
    <LucideIcon
      size={size}
      strokeWidth={strokeWidth}
      className={clsx("shrink-0", className)}
    />
  );
};
