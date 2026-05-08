<script setup lang="ts">
/**
 * HeatmapCalendar: a calendar heatmap of a TimeSeries.
 *
 * Aggregates the time-series into per-day buckets (sum/avg via mode), then
 * renders the standard ECharts calendar visual. Best for "events per day"
 * style metrics — backups, login counts, request volumes.
 */
import * as echarts from 'echarts/core';
import { CalendarComponent, TooltipComponent, VisualMapComponent } from 'echarts/components';
import { HeatmapChart } from 'echarts/charts';
import { CanvasRenderer } from 'echarts/renderers';
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';

import type { TimeSeriesPayload } from '@/api/types';
import type { Widget } from '@/canvas/types';
import { useEChartsTheme } from '@/composables/useEChartsTheme';
import { useMetricStore } from '@/stores/metric';

echarts.use([HeatmapChart, CalendarComponent, TooltipComponent, VisualMapComponent, CanvasRenderer]);

interface HeatmapConfig {
  title?: string;
  /** 'sum' counts events per day; 'avg' averages numeric series. */
  mode?: 'sum' | 'avg';
  /** Number of trailing days to show; defaults to 90. */
  range_days?: number;
}

const props = defineProps<{ widget: Widget }>();

const metricStore = useMetricStore();
const theme = useEChartsTheme();
const containerRef = ref<HTMLDivElement | null>(null);
let chart: echarts.ECharts | null = null;

const cfg = computed<HeatmapConfig>(() => (props.widget.config ?? {}) as HeatmapConfig);
const payload = computed(() => metricStore.payloadOf(props.widget.id));
const errorMsg = computed(() => metricStore.errorOf(props.widget.id));

function aggregate(): { date: string; value: number }[] {
  const ts = payload.value?.time_series as TimeSeriesPayload | undefined;
  const all = ts?.series ?? [];
  const buckets = new Map<string, { sum: number; n: number }>();
  for (const s of all) {
    for (const [t, v] of s.points) {
      if (!Number.isFinite(t) || !Number.isFinite(v)) continue;
      const date = isoDay(t * 1000);
      const cur = buckets.get(date);
      if (!cur) buckets.set(date, { sum: v, n: 1 });
      else {
        cur.sum += v;
        cur.n += 1;
      }
    }
  }
  const mode = cfg.value.mode ?? 'sum';
  const out: { date: string; value: number }[] = [];
  for (const [date, b] of buckets) {
    out.push({ date, value: mode === 'avg' ? b.sum / b.n : b.sum });
  }
  out.sort((a, b) => (a.date < b.date ? -1 : 1));
  return out;
}

function isoDay(ms: number): string {
  const d = new Date(ms);
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

function buildOption(): echarts.EChartsCoreOption {
  const t = theme.value;
  const days = aggregate();
  const range = computeRange(days, cfg.value.range_days ?? 90);
  const max = days.reduce((m, d) => Math.max(m, d.value), 0) || 1;
  return {
    title: cfg.value.title
      ? { text: cfg.value.title, left: 'center', top: 4, textStyle: { color: t.textSecondary, fontSize: 12 } }
      : undefined,
    tooltip: {
      backgroundColor: t.surface,
      borderColor: t.border,
      textStyle: { color: t.textPrimary },
      formatter: (p: { value: [string, number] }) => `${p.value[0]}: ${p.value[1]}`,
    },
    visualMap: {
      min: 0,
      max,
      calculable: false,
      show: false,
      inRange: { color: ['transparent', t.accentBrand] },
    },
    calendar: {
      top: cfg.value.title ? 30 : 12,
      left: 24,
      right: 12,
      cellSize: ['auto', 14],
      range,
      itemStyle: { color: 'transparent', borderColor: t.splitLine },
      splitLine: { lineStyle: { color: t.splitLine } },
      yearLabel: { show: false },
      monthLabel: { color: t.textSecondary, fontSize: 10 },
      dayLabel: { color: t.textSecondary, fontSize: 9 },
    },
    series: [
      {
        type: 'heatmap',
        coordinateSystem: 'calendar',
        data: days.map((d) => [d.date, d.value]),
      },
    ],
  };
}

function computeRange(days: { date: string }[], window: number): [string, string] {
  if (days.length > 0) {
    const last = days[days.length - 1].date;
    const lastDate = new Date(last);
    const start = new Date(lastDate);
    start.setDate(start.getDate() - (window - 1));
    return [isoDay(start.getTime()), last];
  }
  const today = new Date();
  const start = new Date(today);
  start.setDate(start.getDate() - (window - 1));
  return [isoDay(start.getTime()), isoDay(today.getTime())];
}

function render(): void {
  if (!chart) return;
  chart.setOption(buildOption(), { notMerge: true });
}

function resize(): void {
  if (chart) chart.resize();
}

onMounted(() => {
  if (containerRef.value) {
    chart = echarts.init(containerRef.value, undefined, { renderer: 'canvas' });
    render();
  }
  window.addEventListener('resize', resize);
});

onBeforeUnmount(() => {
  window.removeEventListener('resize', resize);
  if (chart) {
    chart.dispose();
    chart = null;
  }
});

watch([payload, cfg, theme], render, { deep: true });
watch(() => [props.widget.w, props.widget.h], () => requestAnimationFrame(resize));
</script>

<template>
  <div class="relative h-full w-full p-2">
    <div
      ref="containerRef"
      class="h-full w-full"
    />
    <div
      v-if="errorMsg"
      class="absolute inset-x-2 bottom-1 truncate rounded bg-red-700/40 px-2 py-1 text-[length:var(--fs-xs)] text-red-100"
      :title="errorMsg"
    >
      {{ errorMsg }}
    </div>
  </div>
</template>
