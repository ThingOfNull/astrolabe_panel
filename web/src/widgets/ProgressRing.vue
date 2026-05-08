<script setup lang="ts">
/**
 * ProgressRing: SVG ring with a centered value.
 *
 * Quieter alternative to Gauge — no axis, no detail formatter, just one ring.
 * Reads a Scalar; normalizes to [0,1] using min/max from config or payload.
 */
import { computed } from 'vue';

import type { ScalarPayload } from '@/api/types';
import type { Widget } from '@/canvas/types';
import { useEChartsTheme } from '@/composables/useEChartsTheme';
import { useMetricStore } from '@/stores/metric';

interface ProgressRingConfig {
  title?: string;
  min?: number;
  max?: number;
  unit?: string;
  /** Stroke thickness as a fraction of radius; default 0.18. */
  thickness?: number;
}

const props = defineProps<{ widget: Widget }>();

const theme = useEChartsTheme();
const metricStore = useMetricStore();
const cfg = computed<ProgressRingConfig>(() => (props.widget.config ?? {}) as ProgressRingConfig);
const payload = computed(() => metricStore.payloadOf(props.widget.id));
const errorMsg = computed(() => metricStore.errorOf(props.widget.id));

const ratio = computed(() => {
  const sc = payload.value?.scalar as ScalarPayload | undefined;
  if (!sc) return 0;
  const min = cfg.value.min ?? 0;
  const max = cfg.value.max ?? 100;
  if (max <= min) return 0;
  const r = (sc.value - min) / (max - min);
  return Math.max(0, Math.min(1, r));
});

const valueLabel = computed(() => {
  const sc = payload.value?.scalar as ScalarPayload | undefined;
  const v = sc?.value;
  if (v == null) return '—';
  const fmt = new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(v);
  return `${fmt}${cfg.value.unit ?? sc?.unit ?? ''}`;
});

// Geometry: 100×100 viewBox so the ring scales with the SVG container.
const RADIUS = 42;
const CIRC = 2 * Math.PI * RADIUS;
const dashOffset = computed(() => CIRC * (1 - ratio.value));

const stroke = computed(() => {
  const t = (cfg.value.thickness ?? 0.18) * RADIUS;
  return Math.max(2, Math.min(t, RADIUS - 4));
});
</script>

<template>
  <div class="relative h-full w-full p-2">
    <p
      v-if="cfg.title"
      class="absolute left-3 top-1.5 text-[length:var(--fs-xs)] text-[color:var(--astro-text-secondary)]"
    >
      {{ cfg.title }}
    </p>
    <svg
      class="block h-full w-full"
      viewBox="0 0 100 100"
      preserveAspectRatio="xMidYMid meet"
      role="img"
      aria-hidden="true"
    >
      <circle
        cx="50"
        cy="50"
        :r="RADIUS"
        fill="none"
        :stroke="theme.splitLine"
        :stroke-width="stroke"
      />
      <circle
        cx="50"
        cy="50"
        :r="RADIUS"
        fill="none"
        :stroke="theme.accentBrand"
        :stroke-width="stroke"
        :stroke-dasharray="CIRC"
        :stroke-dashoffset="dashOffset"
        stroke-linecap="round"
        transform="rotate(-90 50 50)"
        style="transition: stroke-dashoffset 0.4s ease-out"
      />
      <text
        x="50"
        y="48"
        text-anchor="middle"
        dominant-baseline="middle"
        class="astro-mono-num"
        :fill="theme.textPrimary"
        font-size="14"
        font-family="JetBrains Mono, ui-monospace, monospace"
      >
        {{ valueLabel }}
      </text>
      <text
        x="50"
        y="62"
        text-anchor="middle"
        dominant-baseline="middle"
        :fill="theme.textSecondary"
        font-size="6"
      >
        {{ Math.round(ratio * 100) }}%
      </text>
    </svg>
    <div
      v-if="errorMsg"
      class="absolute inset-x-2 bottom-1 truncate rounded bg-red-700/40 px-2 py-1 text-[length:var(--fs-xs)] text-red-100"
      :title="errorMsg"
    >
      {{ errorMsg }}
    </div>
  </div>
</template>
