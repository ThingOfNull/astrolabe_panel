<script setup lang="ts">
/**
 * Bullet: scalar value plotted against three threshold ranges.
 *
 * Pattern from Stephen Few's bullet chart: one bar shows the actual value;
 * the background is divided into qualitative ranges (poor/satisfactory/good)
 * keyed off the configured min/max plus thresholds.
 */
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';

import type { ScalarPayload } from '@/api/types';
import type { Widget } from '@/canvas/types';
import { useEChartsTheme } from '@/composables/useEChartsTheme';
import { useMetricStore } from '@/stores/metric';

interface BulletConfig {
  title?: string;
  min?: number;
  max?: number;
  /** Optional target marker drawn as a thin tick; defaults to mid. */
  target?: number;
  /** Color stops mirror the gauge thresholds. Each entry: { value, color }. */
  thresholds?: { value: number; color: string }[];
  unit?: string;
}

const props = defineProps<{ widget: Widget }>();

const theme = useEChartsTheme();
const metricStore = useMetricStore();
const cfg = computed<BulletConfig>(() => (props.widget.config ?? {}) as BulletConfig);
const payload = computed(() => metricStore.payloadOf(props.widget.id));
const errorMsg = computed(() => metricStore.errorOf(props.widget.id));

const wrapper = ref<HTMLDivElement | null>(null);
const widthPx = ref(0);
const heightPx = ref(0);
let observer: ResizeObserver | null = null;
onMounted(() => {
  if (!wrapper.value) return;
  observer = new ResizeObserver(([entry]) => {
    widthPx.value = Math.max(0, entry.contentRect.width);
    heightPx.value = Math.max(0, entry.contentRect.height);
  });
  observer.observe(wrapper.value);
});
onBeforeUnmount(() => {
  observer?.disconnect();
  observer = null;
});

const value = computed(() => {
  const sc = payload.value?.scalar as ScalarPayload | undefined;
  return sc?.value ?? 0;
});

const min = computed(() => cfg.value.min ?? 0);
const max = computed(() => {
  const m = cfg.value.max ?? 100;
  return m > min.value ? m : min.value + 1;
});

function pct(v: number): number {
  const r = (v - min.value) / (max.value - min.value);
  return Math.max(0, Math.min(1, r));
}

const trackY = computed(() => Math.round(heightPx.value * 0.45));
const trackH = computed(() => Math.max(8, Math.round(heightPx.value * 0.35)));
const barH = computed(() => Math.max(4, Math.round(trackH.value * 0.5)));

// Build threshold bands across the track. If no thresholds are configured,
// fall back to a single band of the brand accent so the chart still reads.
const bands = computed(() => {
  const list = (cfg.value.thresholds ?? []).slice().sort((a, b) => a.value - b.value);
  if (list.length === 0) {
    return [{ from: 0, to: 1, color: `color-mix(in srgb, ${theme.value.accentBrand} 20%, transparent)` }];
  }
  const out: { from: number; to: number; color: string }[] = [];
  let prev = 0;
  for (const t of list) {
    const upper = pct(t.value);
    out.push({ from: prev, to: upper, color: blend(t.color, 0.35) });
    prev = upper;
  }
  if (prev < 1) {
    out.push({ from: prev, to: 1, color: blend(list[list.length - 1].color, 0.55) });
  }
  return out;
});

function blend(color: string, weight: number): string {
  // SVG-friendly tint: rely on color-mix where supported (modern browsers all
  // do at this point; fallback below) so themes propagate cleanly.
  return `color-mix(in srgb, ${color} ${Math.round(weight * 100)}%, transparent)`;
}

const valuePct = computed(() => pct(value.value));
const targetPct = computed(() => {
  if (cfg.value.target == null) return null;
  return pct(cfg.value.target);
});

const valueLabel = computed(() => {
  const v = value.value;
  const fmt = new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(v);
  return `${fmt}${cfg.value.unit ?? payload.value?.scalar?.unit ?? ''}`;
});
</script>

<template>
  <div
    ref="wrapper"
    class="relative h-full w-full p-2"
  >
    <p
      v-if="cfg.title"
      class="text-[length:var(--fs-xs)] text-[color:var(--astro-text-secondary)]"
    >
      {{ cfg.title }}
    </p>
    <svg
      :width="widthPx"
      :height="heightPx"
      class="absolute inset-0"
      role="img"
      aria-hidden="true"
    >
      <!-- Threshold bands -->
      <rect
        v-for="(b, i) in bands"
        :key="i"
        :x="8 + b.from * (widthPx - 16)"
        :y="trackY"
        :width="(b.to - b.from) * (widthPx - 16)"
        :height="trackH"
        :fill="b.color"
        rx="3"
        ry="3"
      />
      <!-- Value bar -->
      <rect
        :x="8"
        :y="trackY + (trackH - barH) / 2"
        :width="valuePct * (widthPx - 16)"
        :height="barH"
        :fill="theme.accentInteractive"
        rx="3"
        ry="3"
      />
      <!-- Target tick -->
      <line
        v-if="targetPct != null"
        :x1="8 + targetPct * (widthPx - 16)"
        :y1="trackY - 4"
        :x2="8 + targetPct * (widthPx - 16)"
        :y2="trackY + trackH + 4"
        :stroke="theme.textPrimary"
        stroke-width="2"
      />
    </svg>
    <span
      class="astro-mono-num absolute right-2 bottom-1 text-[length:var(--fs-sm)] text-[color:var(--astro-text-primary)]"
    >
      {{ valueLabel }}
    </span>
    <div
      v-if="errorMsg"
      class="absolute inset-x-2 bottom-1 truncate rounded bg-red-700/40 px-2 py-1 text-[length:var(--fs-xs)] text-red-100"
      :title="errorMsg"
    >
      {{ errorMsg }}
    </div>
  </div>
</template>
