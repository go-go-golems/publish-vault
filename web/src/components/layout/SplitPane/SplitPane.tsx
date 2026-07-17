/**
 * LAYOUT: SplitPane
 * Design: Retro System 1 — horizontal two-pane split with a draggable retro
 * handle. Wraps ui/resizable (react-resizable-panels) so consumers never
 * touch the shadcn primitive directly.
 */
import React from "react";
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle,
} from "../../ui/resizable";

export interface SplitPaneProps {
  /** Left/main pane content. */
  main: React.ReactNode;
  /** Right/secondary pane content. */
  side: React.ReactNode;
  /** Default size of the main pane in percent. */
  mainDefaultSize?: number;
  mainMinSize?: number;
  sideMinSize?: number;
  sideMaxSize?: number;
  className?: string;
}

export const SplitPane: React.FC<SplitPaneProps> = ({
  main,
  side,
  mainDefaultSize = 75,
  mainMinSize = 40,
  sideMinSize = 12,
  sideMaxSize = 40,
  className,
}) => (
  <ResizablePanelGroup direction="horizontal" className={className ?? "h-full"}>
    <ResizablePanel defaultSize={mainDefaultSize} minSize={mainMinSize} order={1}>
      {main}
    </ResizablePanel>
    <ResizableHandle withHandle className="retro-resize-handle" />
    <ResizablePanel
      defaultSize={100 - mainDefaultSize}
      minSize={sideMinSize}
      maxSize={sideMaxSize}
      order={2}
    >
      {side}
    </ResizablePanel>
  </ResizablePanelGroup>
);
