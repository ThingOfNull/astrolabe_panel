<script setup lang="ts">
/**
 * SmartGuides: Figma-style snap/alignment guides.
 *
 * Pure presentational: the parent passes the active rect (in canvas grid
 * units) and an array of "other" rects. We render the subset of guide lines
 * that match within a small tolerance — typically ±1 grid unit.
 *
 * Canvas units are converted to pixels via basePx inside this component, so
 * callers do not need to do any layout math.
 */

import { computed } from 'vue';

import type { Rect } from './types';

interface Props {
  /** Currently-dragging rect (in grid units); null hides all guides. */
  active: Rect | null;
  /** Other rects on the canvas (in grid units); excluded from this list. */
  others: Rect[];
  /** Grid unit in pixels. */
  basePx: number;
  /** Pixel tolerance for matching edges; defaults to ~1 grid unit. */
  tolerance?: number;
}

const props = withDefaults(defineProps<Props>(), { tolerance: 1 });

interface VLine {
  x: number;
  y1: number;
  y2: number;
}
interface HLine {
  y: number;
  x1: number;
  x2: number;
}

const guides = computed<{ v: VLine[]; h: HLine[] }>(() => {
  if (!props.active) return { v: [], h: [] };
  const a = props.active;
  const aLeft = a.x;
  const aRight = a.x + a.w;
  const aCenterX = a.x + a.w / 2;
  const aTop = a.y;
  const aBottom = a.y + a.h;
  const aCenterY = a.y + a.h / 2;

  const v: VLine[] = [];
  const h: HLine[] = [];
  const tol = props.tolerance;

  for (const o of props.others) {
    const oLeft = o.x;
    const oRight = o.x + o.w;
    const oCenterX = o.x + o.w / 2;
    const oTop = o.y;
    const oBottom = o.y + o.h;
    const oCenterY = o.y + o.h / 2;

    // Vertical lines: left / right / center-x
    for (const [ax, ox] of [
      [aLeft, oLeft],
      [aLeft, oRight],
      [aRight, oLeft],
      [aRight, oRight],
      [aCenterX, oCenterX],
    ] as const) {
      if (Math.abs(ax - ox) <= tol) {
        const y1 = Math.min(aTop, oTop);
        const y2 = Math.max(aBottom, oBottom);
        v.push({ x: ox, y1, y2 });
      }
    }
    // Horizontal lines: top / bottom / center-y
    for (const [ay, oy] of [
      [aTop, oTop],
      [aTop, oBottom],
      [aBottom, oTop],
      [aBottom, oBottom],
      [aCenterY, oCenterY],
    ] as const) {
      if (Math.abs(ay - oy) <= tol) {
        const x1 = Math.min(aLeft, oLeft);
        const x2 = Math.max(aRight, oRight);
        h.push({ y: oy, x1, x2 });
      }
    }
  }
  return { v, h };
});
</script>

<template>
  <svg
    v-if="active && (guides.v.length || guides.h.length)"
    class="pointer-events-none absolute inset-0 z-40"
    :width="'100%'"
    :height="'100%'"
    aria-hidden="true"
  >
    <line
      v-for="(g, i) in guides.v"
      :key="`v-${i}`"
      :x1="g.x * basePx"
      :y1="g.y1 * basePx"
      :x2="g.x * basePx"
      :y2="g.y2 * basePx"
      stroke="var(--astro-accent-selection, var(--astro-accent))"
      stroke-width="1"
      stroke-dasharray="4 4"
      opacity="0.85"
    />
    <line
      v-for="(g, i) in guides.h"
      :key="`h-${i}`"
      :x1="g.x1 * basePx"
      :y1="g.y * basePx"
      :x2="g.x2 * basePx"
      :y2="g.y * basePx"
      stroke="var(--astro-accent-selection, var(--astro-accent))"
      stroke-width="1"
      stroke-dasharray="4 4"
      opacity="0.85"
    />
  </svg>
</template>
