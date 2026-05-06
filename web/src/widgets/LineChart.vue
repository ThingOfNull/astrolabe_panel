<script setup lang="ts">
import * as echarts from 'echarts/core';
import { LineChart } from 'echarts/charts';
import {
  GridComponent,
  LegendComponent,
  TitleComponent,
  TooltipComponent,
} from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';

import type { TimeSeriesPayload } from '@/api/types';
import type { Widget } from '@/canvas/types';
import { useMetricStore } from '@/stores/metric';
import { normalizeThresholds, pickColor } from '@/widgets/thresholds';
import type { Threshold } from '@/widgets/thresholds';

echarts.use([
  LineChart,
  CanvasRenderer,
  GridComponent,
  LegendComponent,
  TitleComponent,
  TooltipComponent,
]);

interface LineConfig {
  title?: string;
  smooth?: boolean;
  area_fill?: boolean;
  thresholds?: Threshold[];
}

const props = defineProps<{
  widget: Widget;
}>();

const metricStore = useMetricStore();
const containerRef = ref<HTMLDivElement | null>(null);
let chart: echarts.ECharts | null = null;

const cfg = computed<LineConfig>(() => {
  const raw = (props.widget.config ?? {}) as LineConfig;
  return { ...raw, thresholds: normalizeThresholds(raw.thresholds) };
});

const payload = computed(() => metricStore.payloadOf(props.widget.id));
const errorMsg = computed(() => metricStore.errorOf(props.widget.id));

function buildOption(): echarts.EChartsCoreOption {
  const ts = payload.value?.time_series as TimeSeriesPayload | undefined;
  const allSeries = ts?.series ?? [];
  // Threshold color uses last sample (max across series if many).
  let last: number | null = null;
  for (const s of allSeries) {
    const tail = s.points[s.points.length - 1];
    if (tail && (last === null || tail[1] > last)) last = tail[1];
  }
  const lineColor = pickColor(last, cfg.value.thresholds, '#00d4ff');
  const seriesData = allSeries.map((s, idx) => ({
    name: s.name,
    type: 'line' as const,
    showSymbol: false,
    smooth: cfg.value.smooth ?? true,
    areaStyle: cfg.value.area_fill ? { opacity: 0.18 } : undefined,
    lineStyle: { color: idx === 0 ? lineColor : undefined, width: 2 },
    itemStyle: { color: idx === 0 ? lineColor : undefined },
    data: s.points.map(([t, v]) => [t * 1000, v]),
  }));
  return {
    grid: { left: 36, right: 12, top: cfg.value.title ? 28 : 16, bottom: 24 },
    title: cfg.value.title
      ? { text: cfg.value.title, left: 'center', textStyle: { color: '#aaa', fontSize: 12 } }
      : undefined,
    legend: { textStyle: { color: '#888', fontSize: 10 }, top: cfg.value.title ? 4 : 0, left: 'center' },
    tooltip: { trigger: 'axis' },
    xAxis: {
      type: 'time',
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.2)' } },
      axisLabel: { color: '#888', fontSize: 10 },
    },
    yAxis: {
      type: 'value',
      name: ts?.unit ?? '',
      nameTextStyle: { color: '#888' },
      splitLine: { lineStyle: { color: 'rgba(255,255,255,0.05)' } },
      axisLabel: { color: '#888', fontSize: 10 },
    },
    series: seriesData,
  };
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

watch([payload, cfg], render);
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
      class="absolute inset-x-2 bottom-1 truncate rounded bg-red-700/40 px-2 py-1 text-[10px] text-red-100"
      :title="errorMsg"
    >
      {{ errorMsg }}
    </div>
  </div>
</template>
