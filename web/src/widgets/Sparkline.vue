<script setup lang="ts">
/**
 * Sparkline: minimalist time-series strip rendered with d3-shape SVG paths.
 *
 * Picked over ECharts for small (≤ 18×6 grid) widgets where the chart
 * library's overhead (axes, instance churn, ~150KB) drowns the actual data.
 * No tooltips, no axes — by design. For a richer trend chart use the LineChart
 * widget instead.
 */
import { area, curveMonotoneX, line } from 'd3-shape';
import { extent, max as d3max, min as d3min } from 'd3-array';
import { scaleLinear } from 'd3-scale';
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';

import type { TimeSeriesPayload } from '@/api/types';
import type { Widget } from '@/canvas/types';
import { useEChartsTheme } from '@/composables/useEChartsTheme';
import { useMetricStore } from '@/stores/metric';

interface SparklineConfig {
  title?: string;
  /** Show the latest value annotation; default true. */
  show_value?: boolean;
  /** Render the area under the line; default true. */
  area_fill?: boolean;
}

const props = defineProps<{ widget: Widget }>();

const theme = useEChartsTheme();
const metricStore = useMetricStore();
const cfg = computed<SparklineConfig>(() => (props.widget.config ?? {}) as SparklineConfig);
const payload = computed(() => metricStore.payloadOf(props.widget.id));
const errorMsg = computed(() => metricStore.errorOf(props.widget.id));

// Reactive container size; cheap ResizeObserver beats hand-coding aspect.
const wrapper = ref<HTMLDivElement | null>(null);
const widthPx = ref(0);
const heightPx = ref(0);
let observer: ResizeObserver | null = null;
onMounted(() => {
  if (!wrapper.value) return;
  observer = new ResizeObserver(([entry]) => {
    const r = entry.contentRect;
    widthPx.value = Math.max(0, r.width);
    heightPx.value = Math.max(0, r.height);
  });
  observer.observe(wrapper.value);
});
onBeforeUnmount(() => {
  observer?.disconnect();
  observer = null;
});

interface Point {
  t: number;
  v: number;
}

const points = computed<Point[]>(() => {
  const ts = payload.value?.time_series as TimeSeriesPayload | undefined;
  const series = ts?.series?.[0];
  if (!series) return [];
  return series.points
    .filter(([t, v]) => Number.isFinite(t) && Number.isFinite(v))
    .map(([t, v]) => ({ t, v }));
});

const padding = { top: 12, right: 8, bottom: 4, left: 8 };

const linePath = computed(() => {
  if (points.value.length < 2 || widthPx.value === 0 || heightPx.value === 0) return '';
  const { x, y } = scales();
  return line<Point>()
    .x((p) => x(p.t))
    .y((p) => y(p.v))
    .curve(curveMonotoneX)(points.value) ?? '';
});

const areaPath = computed(() => {
  if (!cfg.value.area_fill && cfg.value.area_fill !== undefined) return '';
  if (points.value.length < 2 || widthPx.value === 0 || heightPx.value === 0) return '';
  const { x, y, yMin } = scales();
  return area<Point>()
    .x((p) => x(p.t))
    .y0(y(yMin))
    .y1((p) => y(p.v))
    .curve(curveMonotoneX)(points.value) ?? '';
});

const lastPoint = computed(() => {
  if (widthPx.value === 0 || heightPx.value === 0) return null;
  const last = points.value[points.value.length - 1];
  if (!last) return null;
  const { x, y } = scales();
  return { cx: x(last.t), cy: y(last.v), value: last.v };
});

function scales() {
  const innerW = Math.max(1, widthPx.value - padding.left - padding.right);
  const innerH = Math.max(1, heightPx.value - padding.top - padding.bottom);
  const [tMin, tMax] = extent(points.value, (p) => p.t) as [number, number];
  const yMin = (d3min(points.value, (p) => p.v) ?? 0) as number;
  const yMax = (d3max(points.value, (p) => p.v) ?? 1) as number;
  // Avoid a flat line collapsing into a point.
  const yPad = yMax === yMin ? Math.abs(yMax) * 0.05 || 1 : (yMax - yMin) * 0.08;
  const x = scaleLinear()
    .domain([tMin, tMax])
    .range([padding.left, padding.left + innerW]);
  const y = scaleLinear()
    .domain([yMin - yPad, yMax + yPad])
    .range([padding.top + innerH, padding.top]);
  return { x, y, yMin: yMin - yPad };
}

const lastFmt = computed(() => {
  const v = lastPoint.value?.value;
  if (v == null) return '';
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 1,
  }).format(v);
});
</script>

<template>
  <div
    ref="wrapper"
    class="relative h-full w-full p-2"
  >
    <p
      v-if="cfg.title"
      class="absolute left-3 top-1.5 z-10 text-[length:var(--fs-xs)] text-[color:var(--astro-text-secondary)]"
    >
      {{ cfg.title }}
    </p>
    <span
      v-if="(cfg.show_value !== false) && lastPoint"
      class="absolute right-3 top-1.5 z-10 astro-mono-num text-[length:var(--fs-sm)] text-[color:var(--astro-text-primary)]"
    >
      {{ lastFmt }}
    </span>
    <svg
      :width="widthPx"
      :height="heightPx"
      class="absolute inset-0"
      role="img"
      aria-hidden="true"
    >
      <path
        v-if="areaPath"
        :d="areaPath"
        :fill="theme.areaFill"
      />
      <path
        v-if="linePath"
        :d="linePath"
        fill="none"
        :stroke="theme.accentBrand"
        stroke-width="1.6"
        stroke-linejoin="round"
      />
      <circle
        v-if="lastPoint"
        :cx="lastPoint.cx"
        :cy="lastPoint.cy"
        r="3"
        :fill="theme.accentBrand"
      />
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
