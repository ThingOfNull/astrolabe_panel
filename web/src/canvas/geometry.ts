import type { Rect } from './types';
import { DESIGN_GRID_HEIGHT, DESIGN_GRID_WIDTH } from './types';

/** Snap px to nearest grid multiple. */
export function snap(valuePx: number, basePx: number): number {
  if (basePx <= 0) return Math.round(valuePx);
  return Math.round(valuePx / basePx) * basePx;
}

/** Px to grid multipliers (round). */
export function pxToMultiplier(px: number, basePx: number): number {
  if (basePx <= 0) return Math.round(px);
  return Math.round(px / basePx);
}

/** Grid multipliers to px. */
export function multiplierToPx(mul: number, basePx: number): number {
  return mul * basePx;
}

/** Axis-aligned overlap (touching edges = no overlap here). */
export function rectsOverlap(a: Rect, b: Rect): boolean {
  return a.x < b.x + b.w && a.x + a.w > b.x && a.y < b.y + b.h && a.y + a.h > b.y;
}

/** True if candidate hits any obstacle rect. */
export function hasCollision(candidate: Rect, others: Iterable<Rect>): boolean {
  for (const o of others) {
    if (rectsOverlap(candidate, o)) return true;
  }
  return false;
}

/** Clamp rect inside logical canvas bounds. */
export function clampToCanvas(r: Rect, designW = DESIGN_GRID_WIDTH, designH = DESIGN_GRID_HEIGHT): Rect {
  const w = Math.max(1, Math.min(r.w, designW));
  const h = Math.max(1, Math.min(r.h, designH));
  const x = Math.max(0, Math.min(r.x, designW - w));
  const y = Math.max(0, Math.min(r.y, designH - h));
  return { x, y, w, h };
}

/**
 * Greedy placement: try desired rect, then expand in square rings up to radius 256.
 * On total failure returns clamped desired and caller may reject create.
 */
export function findFreeSpot(
  desired: Rect,
  others: Iterable<Rect>,
  designW = DESIGN_GRID_WIDTH,
  designH = DESIGN_GRID_HEIGHT,
): Rect {
  const obstacles: Rect[] = [];
  for (const o of others) obstacles.push(o);
  const clamped = clampToCanvas(desired, designW, designH);
  if (!hasCollision(clamped, obstacles)) return clamped;

  for (let radius = 1; radius <= 256; radius += 1) {
    for (let dx = -radius; dx <= radius; dx += 1) {
      for (let dy = -radius; dy <= radius; dy += 1) {
        if (Math.max(Math.abs(dx), Math.abs(dy)) !== radius) continue;
        const candidate = clampToCanvas(
          { x: clamped.x + dx, y: clamped.y + dy, w: clamped.w, h: clamped.h },
          designW,
          designH,
        );
        if (!hasCollision(candidate, obstacles)) return candidate;
      }
    }
  }
  return clamped;
}
