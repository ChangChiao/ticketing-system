"use client";

import { useRef, useEffect, useState, useCallback } from "react";
import type { EventSectionDetail } from "@/lib/api";

interface VenueMapProps {
  sections: EventSectionDetail[];
  stageConfig?: { x: number; y: number; width: number; height: number };
  onSelect: (sectionId: string) => void;
  selectedSectionId: string | null;
}

function getSectionColor(remaining: number, quota: number): string {
  if (remaining === 0) return "#9ca3af"; // grey
  const ratio = remaining / quota;
  if (ratio > 0.5) return "#22c55e"; // green
  if (ratio > 0.1) return "#eab308"; // yellow
  return "#ef4444"; // red
}

function getSectionBorderColor(remaining: number, quota: number): string {
  if (remaining === 0) return "#6b7280";
  const ratio = remaining / quota;
  if (ratio > 0.5) return "#16a34a";
  if (ratio > 0.1) return "#ca8a04";
  return "#dc2626";
}

export default function VenueMap({
  sections,
  stageConfig,
  onSelect,
  selectedSectionId,
}: VenueMapProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [hoveredSection, setHoveredSection] = useState<EventSectionDetail | null>(null);
  const [tooltipPos, setTooltipPos] = useState({ x: 0, y: 0 });

  const stage = stageConfig || { x: 400, y: 50, width: 200, height: 60 };

  const drawCanvas = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const dpr = window.devicePixelRatio || 1;
    const rect = canvas.getBoundingClientRect();
    canvas.width = rect.width * dpr;
    canvas.height = rect.height * dpr;
    ctx.scale(dpr, dpr);

    // Clear with dark background
    ctx.fillStyle = "#212121";
    ctx.fillRect(0, 0, rect.width, rect.height);

    // Draw stage
    const scaleX = rect.width / 800;
    const scaleY = rect.height / 600;

    ctx.fillStyle = "#2D2D2D";
    const stageX = (stage.x - stage.width / 2) * scaleX;
    const stageY = stage.y * scaleY;
    const stageW = stage.width * scaleX;
    const stageH = stage.height * scaleY;

    ctx.beginPath();
    ctx.roundRect(stageX, stageY, stageW, stageH, [0, 0, 12, 12]);
    ctx.fill();

    ctx.fillStyle = "#777777";
    ctx.font = `600 ${14 * Math.min(scaleX, scaleY)}px "Oswald", sans-serif`;
    ctx.textAlign = "center";
    ctx.letterSpacing = "4px";
    ctx.fillText("STAGE", stage.x * scaleX, (stage.y + stage.height / 2 + 5) * scaleY);
    ctx.letterSpacing = "0px";

    // Draw sections
    for (const section of sections) {
      const polygon = section.polygon as number[][];
      if (!polygon || polygon.length === 0) continue;

      ctx.beginPath();
      ctx.moveTo(polygon[0][0] * scaleX, polygon[0][1] * scaleY);
      for (let i = 1; i < polygon.length; i++) {
        ctx.lineTo(polygon[i][0] * scaleX, polygon[i][1] * scaleY);
      }
      ctx.closePath();

      const isSelected = section.section_id === selectedSectionId;
      const isHovered = hoveredSection?.section_id === section.section_id;
      const color = getSectionColor(section.remaining, section.quota);
      const borderColor = getSectionBorderColor(section.remaining, section.quota);

      // Fill with transparency
      ctx.fillStyle = color;
      ctx.globalAlpha = isSelected ? 0.5 : isHovered ? 0.4 : 0.25;
      ctx.fill();
      ctx.globalAlpha = 1;

      // Border
      ctx.strokeStyle = isSelected ? "#FF6B35" : borderColor;
      ctx.lineWidth = isSelected ? 3 : 2;
      ctx.stroke();

      // Section name
      const centerX = (polygon.reduce((sum, p) => sum + p[0], 0) / polygon.length) * scaleX;
      const centerY = (polygon.reduce((sum, p) => sum + p[1], 0) / polygon.length) * scaleY;

      ctx.fillStyle = "#FFFFFF";
      ctx.font = `600 ${12 * Math.min(scaleX, scaleY)}px "JetBrains Mono", monospace`;
      ctx.textAlign = "center";
      ctx.fillText(section.section_name, centerX, centerY - 4);

      ctx.fillStyle = "#FFFFFF";
      ctx.font = `400 ${10 * Math.min(scaleX, scaleY)}px "JetBrains Mono", monospace`;
      ctx.fillText(`NT$${section.price.toLocaleString()}`, centerX, centerY + 12);
    }
  }, [sections, selectedSectionId, hoveredSection, stage]);

  useEffect(() => {
    drawCanvas();
    window.addEventListener("resize", drawCanvas);
    return () => window.removeEventListener("resize", drawCanvas);
  }, [drawCanvas]);

  const findSection = (clientX: number, clientY: number) => {
    const canvas = canvasRef.current;
    if (!canvas) return null;

    const rect = canvas.getBoundingClientRect();
    const x = ((clientX - rect.left) / rect.width) * 800;
    const y = ((clientY - rect.top) / rect.height) * 600;

    for (const section of sections) {
      const polygon = section.polygon as number[][];
      if (!polygon || polygon.length === 0) continue;
      if (isPointInPolygon(x, y, polygon)) return section;
    }
    return null;
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    const section = findSection(e.clientX, e.clientY);
    setHoveredSection(section);
    if (section) {
      setTooltipPos({ x: e.clientX, y: e.clientY });
    }
  };

  const handleClick = (e: React.MouseEvent) => {
    const section = findSection(e.clientX, e.clientY);
    if (section && section.remaining > 0) {
      onSelect(section.section_id);
    }
  };

  return (
    <div className="relative h-full">
      <canvas
        ref={canvasRef}
        className="w-full h-full cursor-pointer rounded-[var(--radius)]"
        onMouseMove={handleMouseMove}
        onMouseLeave={() => setHoveredSection(null)}
        onClick={handleClick}
      />

      {hoveredSection && (
        <div
          className="fixed z-50 bg-[var(--bg-elevated)] border border-[var(--bg-placeholder)] text-sm px-4 py-3 rounded-xl shadow-lg pointer-events-none"
          style={{
            left: tooltipPos.x + 12,
            top: tooltipPos.y - 50,
          }}
        >
          <p className="font-mono text-xs font-semibold text-[var(--text-primary)]">
            {hoveredSection.section_name}
          </p>
          <p className="font-mono text-[11px] text-[var(--accent-orange)] mt-1">
            NT$ {hoveredSection.price.toLocaleString()}
          </p>
          <p className="font-mono text-[11px] text-[var(--text-secondary)] mt-0.5">
            {hoveredSection.remaining === 0
              ? "// sold_out"
              : `// ${hoveredSection.remaining} remaining`}
          </p>
        </div>
      )}
    </div>
  );
}

function isPointInPolygon(x: number, y: number, polygon: number[][]): boolean {
  let inside = false;
  for (let i = 0, j = polygon.length - 1; i < polygon.length; j = i++) {
    const xi = polygon[i][0], yi = polygon[i][1];
    const xj = polygon[j][0], yj = polygon[j][1];

    const intersect =
      yi > y !== yj > y && x < ((xj - xi) * (y - yi)) / (yj - yi) + xi;
    if (intersect) inside = !inside;
  }
  return inside;
}
