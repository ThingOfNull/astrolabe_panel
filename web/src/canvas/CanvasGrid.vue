<script setup lang="ts">
/**
 * CanvasGrid: two-layer editing-mode grid.
 *
 * Old version: a single dense radial-gradient every `basePx` px; at basePx=10
 * the editor looked like noise. Replaced with a primary step (5× basePx) and
 * a softer secondary step (1× basePx) so the eye can snap to both fine and
 * coarse granularities. Colors derive from the current theme instead of
 * hardcoded `rgba(255,255,255,0.18)` so the light theme stays usable.
 */
import { computed } from 'vue';

const props = defineProps<{
  basePx: number;
  visible: boolean;
}>();

const coarse = computed(() => props.basePx * 5);
const fine = computed(() => props.basePx);

const styleVars = computed(() => ({
  // Coarse dots (stronger) stack over fine dots (faint).
  backgroundImage: [
    // Coarse 5× grid
    `radial-gradient(circle, color-mix(in srgb, var(--astro-text-secondary) 30%, transparent) 1px, transparent 1.5px)`,
    // Fine 1× grid
    `radial-gradient(circle, color-mix(in srgb, var(--astro-text-secondary) 12%, transparent) 0.8px, transparent 1px)`,
  ].join(', '),
  backgroundSize: `${coarse.value}px ${coarse.value}px, ${fine.value}px ${fine.value}px`,
  backgroundPosition: '0 0, 0 0',
}));
</script>

<template>
  <div
    v-if="visible"
    class="pointer-events-none absolute inset-0"
    :style="styleVars"
    aria-hidden="true"
  />
</template>
