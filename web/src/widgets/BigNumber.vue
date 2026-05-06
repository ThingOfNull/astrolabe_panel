<script setup lang="ts">
import { computed } from 'vue';

import type { Widget } from '@/canvas/types';
import { useMetricStore } from '@/stores/metric';
import { normalizeThresholds, pickColor } from '@/widgets/thresholds';
import type { BigNumberConfig } from '@/widgets/types';

const props = defineProps<{
  widget: Widget;
}>();

const metricStore = useMetricStore();

const cfg = computed<BigNumberConfig>(() => {
  const c = (props.widget.config ?? {}) as Partial<BigNumberConfig>;
  return {
    title: c.title ?? '',
    unit: c.unit ?? '',
    precision: c.precision ?? 1,
    thresholds: normalizeThresholds(c.thresholds),
  };
});

const payload = computed(() => metricStore.payloadOf(props.widget.id));
const errorMsg = computed(() => metricStore.errorOf(props.widget.id));

const display = computed<string>(() => {
  const p = payload.value;
  if (!p || p.shape !== 'Scalar' || !p.scalar) return '—';
  const v = p.scalar.value;
  return v.toFixed(Math.max(0, cfg.value.precision ?? 1));
});

const unit = computed<string>(() => {
  if (cfg.value.unit) return cfg.value.unit;
  const p = payload.value;
  return p?.scalar?.unit ?? '';
});

const valueColor = computed<string>(() => {
  const p = payload.value;
  const raw = p?.scalar?.value;
  return pickColor(raw ?? null, cfg.value.thresholds, 'var(--astro-text-primary)');
});
</script>

<template>
  <div class="flex h-full w-full flex-col items-stretch justify-between p-3 text-center">
    <p class="text-xs text-[color:var(--astro-text-secondary)]">
      {{ cfg.title }}
    </p>
    <div
      class="flex flex-1 items-center justify-center gap-1 font-mono"
      :style="{ color: valueColor }"
    >
      <span class="astro-mono-num text-[clamp(1.6rem,5vw,3.5rem)] font-semibold leading-none">{{ display }}</span>
      <span
        v-if="unit"
        class="text-base text-[color:var(--astro-text-secondary)]"
      >{{ unit }}</span>
    </div>
    <div
      v-if="errorMsg"
      class="truncate rounded bg-red-700/40 px-2 py-1 text-[10px] text-red-100"
      :title="errorMsg"
    >
      {{ errorMsg }}
    </div>
  </div>
</template>
