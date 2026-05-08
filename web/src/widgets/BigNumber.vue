<script setup lang="ts">
import { computed } from 'vue';

import Metric from '@/components/Metric.vue';
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

const scalarValue = computed<number | null>(() => {
  const p = payload.value;
  if (!p || p.shape !== 'Scalar' || !p.scalar) return null;
  return p.scalar.value;
});

const unit = computed<string>(() => {
  if (cfg.value.unit) return cfg.value.unit;
  const p = payload.value;
  return p?.scalar?.unit ?? '';
});

// Widget-relative sizing: pick Metric size tier by the widget's actual grid
// footprint rather than viewport width. Small widgets get a compact hero
// number; large widgets pour into the display tier.
const metricSize = computed<'md' | 'lg' | 'xl' | 'display'>(() => {
  const area = (props.widget.w ?? 16) * (props.widget.h ?? 10);
  if (area >= 384) return 'display';
  if (area >= 200) return 'xl';
  if (area >= 120) return 'lg';
  return 'md';
});

const valueColor = computed<string>(() => {
  return pickColor(scalarValue.value, cfg.value.thresholds, 'var(--astro-text-primary)');
});
</script>

<template>
  <div class="flex h-full w-full flex-col items-stretch justify-between p-3 text-center">
    <p
      v-if="cfg.title"
      class="text-[length:var(--fs-xs)] text-[color:var(--astro-text-secondary)]"
    >
      {{ cfg.title }}
    </p>
    <div class="flex flex-1 items-center justify-center">
      <Metric
        :value="scalarValue"
        :unit="unit || undefined"
        :precision="cfg.precision"
        :size="metricSize"
        :color="valueColor"
      />
    </div>
    <div
      v-if="errorMsg"
      class="truncate rounded bg-red-700/40 px-2 py-1 text-[length:var(--fs-xs)] text-red-100"
      :title="errorMsg"
    >
      {{ errorMsg }}
    </div>
  </div>
</template>
