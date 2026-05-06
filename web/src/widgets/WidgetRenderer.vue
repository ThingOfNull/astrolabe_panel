<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue';

import type { MetricQuery } from '@/api/types';
import type { Widget } from '@/canvas/types';
import { useMetricStore } from '@/stores/metric';
import { useProbeStore } from '@/stores/probe';

import AggregatedSearch from './AggregatedSearch.vue';
import BarChart from './BarChart.vue';
import BigNumber from './BigNumber.vue';
import DividerBlock from './DividerBlock.vue';
import Gauge from './Gauge.vue';
import LineChart from './LineChart.vue';
import SmartLink from './SmartLink.vue';
import StatusGrid from './StatusGrid.vue';
import TextBlock from './TextBlock.vue';
import ClockWidget from './ClockWidget.vue';
import WeatherWidget from './WeatherWidget.vue';
import { ACCEPTED_SHAPES } from './types';

const props = defineProps<{
  widget: Widget;
  interactive: boolean;
  searchFocusKey?: number;
}>();

const probeStore = useProbeStore();
const metricStore = useMetricStore();

const linkStatus = computed(() => probeStore.statusOf(props.widget.id));

const isDataWidget = computed(() => Boolean(ACCEPTED_SHAPES[props.widget.type]));

// Data widgets: subscribe metric polling on mount / when binding changes.
watch(
  () => {
    const w = props.widget;
    return [
      w.id,
      w.type,
      w.data_source_id,
      JSON.stringify(w.metric_query ?? null),
    ] as const;
  },
  () => {
    if (!isDataWidget.value) {
      metricStore.unbind(props.widget.id);
      return;
    }
    const ds = props.widget.data_source_id;
    const mq = props.widget.metric_query as MetricQuery | null | undefined;
    if (!ds || !mq?.path || !mq.shape) {
      metricStore.unbind(props.widget.id);
      return;
    }
    metricStore.bind({
      widgetId: props.widget.id,
      dataSourceId: ds,
      query: mq,
    });
  },
  { immediate: true },
);

onBeforeUnmount(() => metricStore.unbind(props.widget.id));
</script>

<template>
  <SmartLink
    v-if="widget.type === 'link'"
    :widget="widget"
    :status="linkStatus"
    :interactive="interactive"
  />
  <AggregatedSearch
    v-else-if="widget.type === 'search'"
    :widget="widget"
    :interactive="interactive"
    :focus-key="searchFocusKey"
  />
  <Gauge
    v-else-if="widget.type === 'gauge'"
    :widget="widget"
  />
  <BigNumber
    v-else-if="widget.type === 'bignumber'"
    :widget="widget"
  />
  <LineChart
    v-else-if="widget.type === 'line'"
    :widget="widget"
  />
  <BarChart
    v-else-if="widget.type === 'bar'"
    :widget="widget"
  />
  <StatusGrid
    v-else-if="widget.type === 'grid'"
    :widget="widget"
  />
  <TextBlock
    v-else-if="widget.type === 'text'"
    :widget="widget"
  />
  <DividerBlock
    v-else-if="widget.type === 'divider'"
    :widget="widget"
  />
  <WeatherWidget
    v-else-if="widget.type === 'weather'"
    :widget="widget"
  />
  <ClockWidget
    v-else-if="widget.type === 'clock'"
    :widget="widget"
  />
  <div
    v-else
    class="flex h-full w-full items-center justify-center text-xs text-[color:var(--astro-text-secondary)]"
  >
    未知组件类型 {{ widget.type }}
  </div>
</template>
