<script setup lang="ts">
import * as echarts from 'echarts/core';
import { BarChart } from 'echarts/charts';
import {
  GridComponent,
  TitleComponent,
  TooltipComponent,
} from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';

import type { CategoricalPayload } from '@/api/types';
import type { Widget } from '@/canvas/types';
import { useEChartsTheme } from '@/composables/useEChartsTheme';
import { useMetricStore } from '@/stores/metric';
import { normalizeThresholds, pickColor } from '@/widgets/thresholds';
import type { Threshold } from '@/widgets/thresholds';

echarts.use([
  BarChart,
  CanvasRenderer,
  GridComponent,
  TitleComponent,
  TooltipComponent,
]);

interface BarConfig {
  title?: string;
  horizontal?: boolean;
  top_n?: number;
  thresholds?: Threshold[];
}

const props = defineProps<{
  widget: Widget;
}>();

const metricStore = useMetricStore();
const theme = useEChartsTheme();
const containerRef = ref<HTMLDivElement | null>(null);
let chart: echarts.ECharts | null = null;

const cfg = computed<BarConfig>(() => {
  const raw = (props.widget.config ?? {}) as BarConfig;
  return { ...raw, thresholds: normalizeThresholds(raw.thresholds) };
});

const payload = computed(() => metricStore.payloadOf(props.widget.id));
const errorMsg = computed(() => metricStore.errorOf(props.widget.id));

function buildOption(): echarts.EChartsCoreOption {
  const cat = payload.value?.categorical as CategoricalPayload | undefined;
  let items = cat?.items ?? [];
  if (cfg.value.top_n && cfg.value.top_n > 0) {
    items = [...items].sort((a, b) => b.value - a.value).slice(0, cfg.value.top_n);
  }
  const labels = items.map((i) => i.label);
  const values = items.map((i) => i.value);
  const horizontal = cfg.value.horizontal ?? true;
  const t = theme.value;
  const labelStyle = { color: t.textSecondary, fontSize: 10 };
  const splitLine = { lineStyle: { color: t.splitLine } };
  const axisLine = { lineStyle: { color: t.splitLine } };
  const valueAxis = {
    type: 'value' as const,
    axisLabel: labelStyle,
    splitLine,
    axisLine,
    name: cat?.unit ?? '',
    nameTextStyle: { color: t.textSecondary },
  };
  const categoryAxis = {
    type: 'category' as const,
    data: labels,
    axisLabel: { ...labelStyle, interval: 0, rotate: horizontal ? 0 : 30 },
    axisLine,
  };
  return {
    grid: {
      left: horizontal ? 90 : 36,
      right: 12,
      top: cfg.value.title ? 28 : 16,
      bottom: horizontal ? 24 : 60,
    },
    title: cfg.value.title
      ? { text: cfg.value.title, left: 'center', textStyle: { color: t.textSecondary, fontSize: 12 } }
      : undefined,
    tooltip: {
      trigger: 'axis',
      backgroundColor: t.surface,
      borderColor: t.border,
      textStyle: { color: t.textPrimary },
    },
    xAxis: horizontal ? valueAxis : categoryAxis,
    yAxis: horizontal ? categoryAxis : valueAxis,
    series: [
      {
        type: 'bar',
        data: values.map((v) => ({
          value: v,
          itemStyle: {
            color: pickColor(v, cfg.value.thresholds, t.accentInteractive),
            borderRadius: 4,
          },
        })),
        barMaxWidth: 18,
      },
    ],
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
      class="absolute inset-x-2 bottom-1 truncate rounded bg-red-700/40 px-2 py-1 text-[10px] text-red-100"
      :title="errorMsg"
    >
      {{ errorMsg }}
    </div>
  </div>
</template>
