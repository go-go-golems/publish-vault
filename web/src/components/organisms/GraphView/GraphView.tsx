/**
 * ORGANISM: GraphView
 * Design: Retro System 1 — canvas-based force-directed graph, monochrome with blue active node.
 * Uses a simple spring simulation without external dependencies.
 */
import React, {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { clsx } from "clsx";
import type { GraphData, GraphNode, GraphEdge } from "../../../types";

interface NodeState {
  id: string;
  title: string;
  x: number;
  y: number;
  vx: number;
  vy: number;
  radius: number;
}

export interface GraphViewProps {
  data: GraphData;
  activeNodeId?: string;
  onNodeClick?: (id: string) => void;
  className?: string;
  width?: number;
  height?: number;
}

const NODE_RADIUS = 5;
const ACTIVE_RADIUS = 8;
const REPULSION = 3000;
const SPRING_LENGTH = 80;
const SPRING_K = 0.05;
const DAMPING = 0.85;
const ITERATIONS = 200; // pre-warm simulation

export const GraphView: React.FC<GraphViewProps> = ({
  data,
  activeNodeId,
  onNodeClick,
  className,
  width = 400,
  height = 300,
}) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const nodesRef = useRef<NodeState[]>([]);
  const animRef = useRef<number>(0);
  const [hoveredId, setHoveredId] = useState<string | null>(null);

  // Build adjacency for spring forces
  const edgeSet = useRef<Set<string>>(new Set());

  const nodesData = data.nodes ?? [];
  const edgesData = data.edges ?? [];

  // Initialize nodes
  useEffect(() => {
    if (!nodesData.length) return;

    edgeSet.current = new Set(
      edgesData.map((e) => `${e.source}|${e.target}`)
    );

    const cx = width / 2;
    const cy = height / 2;
    nodesRef.current = nodesData.map((n, i) => {
      const angle = (i / nodesData.length) * 2 * Math.PI;
      const r = Math.min(width, height) * 0.3;
      return {
        id: n.id,
        title: n.title,
        x: cx + r * Math.cos(angle) + (Math.random() - 0.5) * 20,
        y: cy + r * Math.sin(angle) + (Math.random() - 0.5) * 20,
        vx: 0,
        vy: 0,
        radius: n.id === activeNodeId ? ACTIVE_RADIUS : NODE_RADIUS,
      };
    });

    // Pre-warm
    for (let i = 0; i < ITERATIONS; i++) {
      tick(nodesRef.current, edgesData, edgeSet.current, width, height);
    }
  }, [nodesData, edgesData, activeNodeId, width, height]);

  // Animation loop
  const draw = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const nodes = nodesRef.current;
    tick(nodes, edgesData, edgeSet.current, width, height);

    ctx.clearRect(0, 0, width, height);

    // Background
    ctx.fillStyle = "#f0ede8";
    ctx.fillRect(0, 0, width, height);

    // Edges
    ctx.strokeStyle = "#1a1a1a";
    ctx.lineWidth = 0.5;
    ctx.globalAlpha = 0.3;
    for (const edge of edgesData) {
      const src = nodes.find((n) => n.id === edge.source);
      const tgt = nodes.find((n) => n.id === edge.target);
      if (!src || !tgt) continue;
      ctx.beginPath();
      ctx.moveTo(src.x, src.y);
      ctx.lineTo(tgt.x, tgt.y);
      ctx.stroke();
    }
    ctx.globalAlpha = 1;

    // Nodes
    for (const node of nodes) {
      const isActive = node.id === activeNodeId;
      const isHovered = node.id === hoveredId;

      ctx.beginPath();
      ctx.arc(node.x, node.y, node.radius, 0, 2 * Math.PI);

      if (isActive) {
        ctx.fillStyle = "#0000cc";
        ctx.strokeStyle = "#1a1a1a";
        ctx.lineWidth = 1;
      } else if (isHovered) {
        ctx.fillStyle = "#555";
        ctx.strokeStyle = "#1a1a1a";
        ctx.lineWidth = 1;
      } else {
        ctx.fillStyle = "#1a1a1a";
        ctx.strokeStyle = "#1a1a1a";
        ctx.lineWidth = 0.5;
      }
      ctx.fill();
      ctx.stroke();

      // Label for active or hovered
      if (isActive || isHovered) {
        ctx.font = "bold 9px monospace";
        ctx.fillStyle = "#1a1a1a";
        ctx.textAlign = "center";
        ctx.fillText(
          node.title.length > 20 ? node.title.slice(0, 18) + "…" : node.title,
          node.x,
          node.y - node.radius - 3
        );
      }
    }

    animRef.current = requestAnimationFrame(draw);
  }, [edgesData, activeNodeId, hoveredId, width, height]);

  useEffect(() => {
    animRef.current = requestAnimationFrame(draw);
    return () => cancelAnimationFrame(animRef.current);
  }, [draw]);

  // Mouse interaction
  const getNodeAt = useCallback(
    (x: number, y: number): NodeState | null => {
      for (const node of nodesRef.current) {
        const dx = node.x - x;
        const dy = node.y - y;
        if (Math.sqrt(dx * dx + dy * dy) <= node.radius + 4) return node;
      }
      return null;
    },
    []
  );

  const handleMouseMove = useCallback(
    (e: React.MouseEvent<HTMLCanvasElement>) => {
      const rect = canvasRef.current!.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;
      const node = getNodeAt(x, y);
      setHoveredId(node?.id ?? null);
    },
    [getNodeAt]
  );

  const handleClick = useCallback(
    (e: React.MouseEvent<HTMLCanvasElement>) => {
      const rect = canvasRef.current!.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;
      const node = getNodeAt(x, y);
      if (node) onNodeClick?.(node.id);
    },
    [getNodeAt, onNodeClick]
  );

  return (
    <canvas
      ref={canvasRef}
      width={width}
      height={height}
      className={clsx(
        "retro-graph-canvas",
        hoveredId ? "cursor-pointer" : "cursor-default",
        className
      )}
      onMouseMove={handleMouseMove}
      onMouseLeave={() => setHoveredId(null)}
      onClick={handleClick}
    />
  );
};

// ── Physics ────────────────────────────────────────────────────────

function tick(
  nodes: NodeState[],
  edges: GraphEdge[],
  edgeSet: Set<string>,
  width: number,
  height: number
) {
  // Repulsion
  for (let i = 0; i < nodes.length; i++) {
    for (let j = i + 1; j < nodes.length; j++) {
      const a = nodes[i];
      const b = nodes[j];
      const dx = b.x - a.x;
      const dy = b.y - a.y;
      const dist = Math.sqrt(dx * dx + dy * dy) || 1;
      const force = REPULSION / (dist * dist);
      const fx = (dx / dist) * force;
      const fy = (dy / dist) * force;
      a.vx -= fx;
      a.vy -= fy;
      b.vx += fx;
      b.vy += fy;
    }
  }

  // Spring attraction
  for (const edge of edges) {
    const src = nodes.find((n) => n.id === edge.source);
    const tgt = nodes.find((n) => n.id === edge.target);
    if (!src || !tgt) continue;
    const dx = tgt.x - src.x;
    const dy = tgt.y - src.y;
    const dist = Math.sqrt(dx * dx + dy * dy) || 1;
    const force = SPRING_K * (dist - SPRING_LENGTH);
    const fx = (dx / dist) * force;
    const fy = (dy / dist) * force;
    src.vx += fx;
    src.vy += fy;
    tgt.vx -= fx;
    tgt.vy -= fy;
  }

  // Center gravity
  const cx = width / 2;
  const cy = height / 2;
  for (const node of nodes) {
    node.vx += (cx - node.x) * 0.003;
    node.vy += (cy - node.y) * 0.003;
  }

  // Integrate
  for (const node of nodes) {
    node.vx *= DAMPING;
    node.vy *= DAMPING;
    node.x += node.vx;
    node.y += node.vy;
    // Clamp to canvas
    node.x = Math.max(node.radius + 2, Math.min(width - node.radius - 2, node.x));
    node.y = Math.max(node.radius + 2, Math.min(height - node.radius - 2, node.y));
  }
}
