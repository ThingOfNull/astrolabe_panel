// Viewport tiers:
// - <768 compact: single column feed
// - else scale: fit canvas to viewport width (cap at 1:1 when wide enough)
//
// Keeps home canvas center aligned with viewport vs settings preview.
// Editor still uses full coordinate space (mode forced to full there).

import { computed, onBeforeUnmount, onMounted, ref } from 'vue';

import { DESIGN_GRID_WIDTH } from '@/canvas/types';

export type ViewportMode = 'full' | 'scale' | 'compact';

export function useViewportMode(basePx: () => number) {
  const width = ref<number>(typeof window !== 'undefined' ? window.innerWidth : 1024);

  function update(): void {
    if (typeof window !== 'undefined') {
      width.value = window.innerWidth;
    }
  }

  onMounted(() => {
    update();
    window.addEventListener('resize', update);
  });
  onBeforeUnmount(() => {
    window.removeEventListener('resize', update);
  });

  const mode = computed<ViewportMode>(() => {
    if (width.value < 768) return 'compact';
    return 'scale';
  });

  // Whole-canvas scale from (viewport padding) / design width; clamp [0.4, 1].
  // Wider than design -> no upscale; narrow -> shrink to avoid horizontal clip.
  // Padding matches HomePage <main class="p-6"> (24px * 2).
  const scale = computed<number>(() => {
    if (mode.value !== 'scale') return 1;
    const designWidth = DESIGN_GRID_WIDTH * basePx();
    const padding = 48;
    const fit = (width.value - padding) / designWidth;
    return Math.max(0.4, Math.min(1, fit));
  });

  return { width, mode, scale };
}
