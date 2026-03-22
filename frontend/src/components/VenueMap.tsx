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

    // Clear
    ctx.clearRect(0, 0, rect.width, rect.height);

    // Draw stage
    ctx.fillStyle = "#1f2937";
    ctx.fillRect(
      (stage.x - stage.width / 2) * (rect.width / 800),
      stage.y * (rect.height / 600),
      stage.width * (rect.width / 800),
      stage.height * (rect.height / 600)
    );
    ctx.fillStyle = "#ffffff";
    ctx.font = "bold 16px sans-serif";
    ctx.textAlign = "center";
    ctx.fillText(
      "舞台",
      stage.x * (rect.width / 800),
      (stage.y + stage.height / 2 + 6) * (rect.height / 600)
    );

    // Draw sections
    for (const section of sections) {
      const polygon = section.polygon as number[][];
      if (!polygon || polygon.length === 0) continue;

      const scaleX = rect.width / 800;
      const scaleY = rect.height / 600;

      ctx.beginPath();
      ctx.moveTo(polygon[0][0] * scaleX, polygon[0][1] * scaleY);
      for (let i = 1; i < polygon.length; i++) {
        ctx.lineTo(polygon[i][0] * scaleX, polygon[i][1] * scaleY);
      }
      ctx.closePath();

      // Fill color
      const isSelected = section.section_id === selectedSectionId;
      const isHovered = hoveredSection?.section_id === section.section_id;

      ctx.fillStyle = getSectionColor(section.remaining, section.quota);
      ctx.globalAlpha = isSelected ? 1 : isHovered ? 0.9 : 0.7;
      ctx.fill();
      ctx.globalAlpha = 1;

      // Border
      ctx.strokeStyle = isSelected ? "#7c3aed" : "#374151";
      ctx.lineWidth = isSelected ? 3 : 1;
      ctx.stroke();

      // Section name
      const centerX =
        (polygon.reduce((sum, p) => sum + p[0], 0) / polygon.length) * scaleX;
      const centerY =
        (polygon.reduce((sum, p) => sum + p[1], 0) / polygon.length) * scaleY;

      ctx.fillStyle = "#ffffff";
      ctx.font = "bold 14px sans-serif";
      ctx.textAlign = "center";
      ctx.fillText(section.section_name, centerX, centerY - 4);

      ctx.font = "12px sans-serif";
      ctx.fillText(`NT$${section.price.toLocaleString()}`, centerX, centerY + 14);
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
    <div className="relative">
      <canvas
        ref={canvasRef}
        className="w-full aspect-[4/3] cursor-pointer bg-gray-100 rounded-lg"
        onMouseMove={handleMouseMove}
        onMouseLeave={() => setHoveredSection(null)}
        onClick={handleClick}
      />

      {hoveredSection && (
        <div
          className="fixed z-50 bg-gray-900 text-white text-sm px-3 py-2 rounded shadow-lg pointer-events-none"
          style={{
            left: tooltipPos.x + 12,
            top: tooltipPos.y - 40,
          }}
        >
          <p className="font-bold">{hoveredSection.section_name}</p>
          <p>NT$ {hoveredSection.price.toLocaleString()}</p>
          <p>
            剩餘:{" "}
            {hoveredSection.remaining === 0
              ? "售完"
              : `${hoveredSection.remaining} 張`}
          </p>
        </div>
      )}
    </div>
  );
}

function isPointInPolygon(x: number, y: number, polygon: number[][]): boolean {
  let inside = false;
  for (let i = 0, j = polygon.length - 1; i < polygon.length; j = i++) {
    const xi = polygon[i][0],
      yi = polygon[i][1];
    const xj = polygon[j][0],
      yj = polygon[j][1];

    const intersect =
      yi > y !== yj > y && x < ((xj - xi) * (y - yi)) / (yj - yi) + xi;
    if (intersect) inside = !inside;
  }
  return inside;
}
