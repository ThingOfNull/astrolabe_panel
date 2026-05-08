<script setup lang="ts">
/**
 * LiquidFill: percentage ball.
 *
 * Best for "fullness" metrics — disk usage, memory, battery — where users
 * want one glance at "how full is the bucket". Reads a Scalar shape; the
 * value is normalized into [0,1] using min/max from the metric or config.
 */
import * as echarts from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import 'echarts-liquidfill';
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';

import type { ScalarPayload } from '@/api/types';
import type { Widget } from '@/canvas/types';
import { useEChartsTheme } from '@/composables/useEChartsTheme';
import { useMetricStore } from '@/stores/metric';

echarts.use([CanvasRenderer]);

interface LiquidFillConfig {
  title?: string;
  /** Manual normalization range; falls back to ScalarPayload.min/max or 0..100. */
  min?: number;
  max?: number;
  /** Optional label suffix shown beside the percentage; defaults to '%'. */
  unit?: string;
  /** When true, treats the raw value as already 0..1 (skips normalization). */
  raw_ratio?: boolean;
}

const props = defineProps<{
  widget: Widget;
}>();

const metricStore = useMetricStore();
const theme = useEChartsTheme();
const containerRef = ref<HTMLDivElement | null>(null);
let chart: echarts.ECharts | null = null;

const cfg = computed<LiquidFillConfig>(() => (props.widget.config ?? {}) as LiquidFillConfig);
const payload = computed(() => metricStore.payloadOf(props.widget.id));
const errorMsg = computed(() => metricStore.errorOf(props.widget.id));

function ratio(): number {
  const p = payload.value;
  if (!p || p.shape !== 'Scalar' || !p.scalar) return 0;
  const sc = p.scalar as ScalarPayload;
  if (cfg.value.raw_ratio) return clamp01(sc.value);
  const min = cfg.value.min ?? 0;
  const max = cfg.value.max ?? 100;
  if (max <= min) return 0;
  return clamp01((sc.value - min) / (max - min));
}

function buildOption(): echarts.EChartsCoreOption {
  const t = theme.value;
  const r = ratio();
  // Three layered series with descending amplitudes give the painterly water
  // look without a custom shader. The brand accent picks up theme changes
  // automatically through useEChartsTheme.
  return {
    series: [
      {
        type: 'liquidFill',
        data: [r, r * 0.92, r * 0.85],
        radius: '78%',
        color: [t.accentBrand, t.accentInteractive, t.accentBrand],
        backgroundStyle: {
          color: 'transparent',
          borderColor: t.border,
          borderWidth: 1,
        },
        outline: {
          show: true,
          borderDistance: 4,
          itemStyle: {
            borderColor: t.border,
            borderWidth: 2,
            shadowBlur: 12,
            shadowColor: t.accentBrand,
          },
        },
        label: {
          formatter: () => `${Math.round(r * 100)}${cfg.value.unit ?? '%'}`,
          fontSize: 32,
          fontFamily: 'JetBrains Mono, ui-monospace, monospace',
          fontWeight: 600,
          color: t.textPrimary,
        },
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

function clamp01(n: number): number {
  if (!Number.isFinite(n)) return 0;
  if (n < 0) return 0;
  if (n > 1) return 1;
  return n;
}
</script>

<template>
  <div class="relative h-full w-full p-2">
    <p
      v-if="cfg.title"
      class="absolute left-3 top-2 z-10 text-[length:var(--fs-xs)] text-[color:var(--astro-text-secondary)]"
    >
      {{ cfg.title }}
    </p>
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
