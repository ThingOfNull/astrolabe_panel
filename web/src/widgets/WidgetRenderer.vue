<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import type { MetricQuery } from '@/api/types';
import type { Widget } from '@/canvas/types';
import { useMetricStore } from '@/stores/metric';
import { useProbeStore } from '@/stores/probe';

import AggregatedSearch from './AggregatedSearch.vue';
import SmartLink from './SmartLink.vue';
import { widgetComponents } from './registry';
import { ACCEPTED_SHAPES } from './types';

const props = defineProps<{
  widget: Widget;
  interactive: boolean;
  searchFocusKey?: number;
}>();

const { t } = useI18n();
const probeStore = useProbeStore();
const metricStore = useMetricStore();

const linkStatus = computed(() => probeStore.statusOf(props.widget.id));

const isDataWidget = computed(() => Boolean(ACCEPTED_SHAPES[props.widget.type]));

// Dynamic dispatch: most widget renderers take exactly { widget }, so the
// dispatcher resolves a component from the registry. SmartLink and the
// aggregated search bar still need transport-specific props (status,
// focus-key) so they keep their dedicated v-if branches above.
const widgetComp = computed(() => widgetComponents[props.widget.type]);

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
  <component
    :is="widgetComp"
    v-else-if="widgetComp"
    :widget="widget"
  />
  <div
    v-else
    class="flex h-full w-full items-center justify-center text-xs text-[color:var(--astro-text-secondary)]"
  >
    {{ t('widget.unknownComponent', { type: widget.type }) }}
  </div>
</template>
