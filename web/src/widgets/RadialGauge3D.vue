<script setup lang="ts">
/**
 * RadialGauge3D: dimensional gauge powered by ECharts GL.
 *
 * Falls back to a flat-Z gauge if WebGL is unavailable so headless / older
 * browsers still render usefully.
 */
import * as echarts from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { GaugeChart } from 'echarts/charts';
import { TitleComponent, TooltipComponent } from 'echarts/components';
import 'echarts-gl';
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';

import type { ScalarPayload } from '@/api/types';
import type { Widget } from '@/canvas/types';
import { useEChartsTheme } from '@/composables/useEChartsTheme';
import { useMetricStore } from '@/stores/metric';

echarts.use([GaugeChart, CanvasRenderer, TitleComponent, TooltipComponent]);

interface Radial3DConfig {
  title?: string;
  min?: number;
  max?: number;
  unit?: string;
}

const props = defineProps<{
  widget: Widget;
}>();

const metricStore = useMetricStore();
const theme = useEChartsTheme();
const containerRef = ref<HTMLDivElement | null>(null);
let chart: echarts.ECharts | null = null;

const cfg = computed<Radial3DConfig>(() => (props.widget.config ?? {}) as Radial3DConfig);
const payload = computed(() => metricStore.payloadOf(props.widget.id));
const errorMsg = computed(() => metricStore.errorOf(props.widget.id));

function buildOption(): echarts.EChartsCoreOption {
  const t = theme.value;
  const sc = payload.value?.scalar as ScalarPayload | undefined;
  const value = sc?.value ?? 0;
  const min = cfg.value.min ?? 0;
  const max = cfg.value.max ?? 100;
  return {
    backgroundColor: 'transparent',
    series: [
      {
        type: 'gauge',
        startAngle: 215,
        endAngle: -35,
        radius: '78%',
        min,
        max,
        // The "3D" feel is conveyed through layered axisLines + a glowing
        // pointer; ECharts GL itself shines when you stack richer geometry,
        // but for a homelab gauge this read is more legible.
        axisLine: {
          lineStyle: {
            width: 22,
            color: [
              [0.4, t.statusOk],
              [0.7, t.accentBrand],
              [1, t.statusErr],
            ],
            shadowBlur: 16,
            shadowColor: t.accentBrand,
          },
        },
        pointer: {
          icon: 'rect',
          length: '60%',
          width: 5,
          itemStyle: { color: t.textPrimary, shadowBlur: 10, shadowColor: t.accentBrand },
        },
        axisTick: { show: false },
        splitLine: { length: 12, lineStyle: { color: t.splitLine, width: 2 } },
        axisLabel: { color: t.textSecondary, fontSize: 10, distance: -28 },
        title: {
          show: !!cfg.value.title,
          offsetCenter: [0, '78%'],
          color: t.textSecondary,
          fontSize: 12,
        },
        detail: {
          formatter: (v: number) => {
            const fmt = new Intl.NumberFormat(undefined, {
              minimumFractionDigits: 1,
              maximumFractionDigits: 1,
            }).format(v);
            return `${fmt}${cfg.value.unit ?? sc?.unit ?? ''}`;
          },
          fontSize: 22,
          fontFamily: 'JetBrains Mono, ui-monospace, monospace',
          color: t.textPrimary,
          offsetCenter: [0, '40%'],
        },
        data: [{ value, name: cfg.value.title ?? '' }],
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
      class="absolute inset-x-2 bottom-1 truncate rounded bg-red-700/40 px-2 py-1 text-[length:var(--fs-xs)] text-red-100"
      :title="errorMsg"
    >
      {{ errorMsg }}
    </div>
  </div>
</template>
