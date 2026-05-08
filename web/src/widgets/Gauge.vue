<script setup lang="ts">
import * as echarts from 'echarts/core';
import { GaugeChart } from 'echarts/charts';
import { CanvasRenderer } from 'echarts/renderers';
import { TitleComponent, TooltipComponent } from 'echarts/components';
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';

import type { ScalarPayload } from '@/api/types';
import type { Widget } from '@/canvas/types';
import { useEChartsTheme } from '@/composables/useEChartsTheme';
import { useMetricStore } from '@/stores/metric';
import { normalizeThresholds, pickColor } from '@/widgets/thresholds';
import type { GaugeConfig } from '@/widgets/types';

echarts.use([GaugeChart, CanvasRenderer, TitleComponent, TooltipComponent]);

const props = defineProps<{
  widget: Widget;
}>();

const metricStore = useMetricStore();
const theme = useEChartsTheme();
const containerRef = ref<HTMLDivElement | null>(null);
let chart: echarts.ECharts | null = null;

const cfg = computed<GaugeConfig>(() => {
  const c = (props.widget.config ?? {}) as Partial<GaugeConfig>;
  return {
    title: c.title ?? '',
    min: c.min ?? 0,
    max: c.max ?? 100,
    unit: c.unit ?? '',
    thresholds: normalizeThresholds(c.thresholds),
  };
});

const payload = computed(() => metricStore.payloadOf(props.widget.id));
const errorMsg = computed(() => metricStore.errorOf(props.widget.id));

function value(): number {
  const p = payload.value;
  if (!p || p.shape !== 'Scalar' || !p.scalar) return 0;
  return (p.scalar as ScalarPayload).value;
}

function buildOption(): echarts.EChartsCoreOption {
  const v = value();
  const t = theme.value;
  const accent = pickColor(v, cfg.value.thresholds, t.accentBrand);
  const range = Math.max(1, (cfg.value.max ?? 100) - (cfg.value.min ?? 0));
  const min = cfg.value.min ?? 0;
  const stops = (cfg.value.thresholds ?? [])
    .map((th) => [Math.min(1, Math.max(0, (th.value - min) / range)), th.color] as [number, string])
    .sort((a, b) => a[0] - b[0]);
  const ranges: [number, string][] = [];
  let cursor = 0;
  // Base track uses a low-saturation mix of the accent so it stays legible
  // on both dark and light surfaces.
  const trackBase = `color-mix(in srgb, ${t.accentBrand} 28%, transparent)`;
  let prevColor = stops[0]?.[1] ?? trackBase;
  for (const [pos, color] of stops) {
    if (pos > cursor) ranges.push([pos, prevColor]);
    prevColor = color;
    cursor = pos;
  }
  if (cursor < 1) ranges.push([1, prevColor]);
  const axisLineColor = ranges.length > 0 ? ranges : [[1, trackBase]];
  return {
    series: [
      {
        type: 'gauge',
        startAngle: 200,
        endAngle: -20,
        min: cfg.value.min,
        max: cfg.value.max,
        progress: { show: true, width: 14, itemStyle: { color: accent } },
        axisLine: { lineStyle: { color: axisLineColor, width: 14 } },
        axisTick: { show: false },
        axisLabel: { color: t.textSecondary, fontSize: 10 },
        splitLine: { length: 8, lineStyle: { color: t.splitLine } },
        pointer: { width: 4, itemStyle: { color: accent } },
        anchor: { show: true, size: 10, itemStyle: { color: t.textPrimary } },
        title: {
          show: !!cfg.value.title,
          fontSize: 12,
          color: t.textSecondary,
          offsetCenter: [0, '70%'],
        },
        detail: {
          formatter: (val: number) => {
            const d = 1;
            const str = new Intl.NumberFormat(undefined, {
              minimumFractionDigits: d,
              maximumFractionDigits: d,
            }).format(val);
            return `${str}${cfg.value.unit ?? ''}`;
          },
          fontSize: 22,
          fontFamily: 'JetBrains Mono, ui-monospace, monospace',
          color: accent,
          offsetCenter: [0, '40%'],
        },
        data: [{ value: v, name: cfg.value.title ?? '' }],
      },
    ],
  };
}

function render(): void {
  if (!chart) return;
  chart.setOption(buildOption());
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
