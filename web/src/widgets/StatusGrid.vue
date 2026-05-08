<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import type { EntityListItem, EntityListPayload } from '@/api/types';
import type { Widget } from '@/canvas/types';
import { useMetricStore } from '@/stores/metric';

interface StatusGridConfig {
  title?: string;
  cell_min_px?: number;
}

const props = defineProps<{
  widget: Widget;
}>();

const { t } = useI18n();
const metricStore = useMetricStore();

const cfg = computed<StatusGridConfig>(() => (props.widget.config ?? {}) as StatusGridConfig);

const items = computed<EntityListItem[]>(() => {
  const ent = metricStore.payloadOf(props.widget.id)?.entity_list as EntityListPayload | undefined;
  return ent?.items ?? [];
});

const errorMsg = computed(() => metricStore.errorOf(props.widget.id));

function statusClass(s: EntityListItem['status']): string {
  switch (s) {
    case 'ok':
      return 'bg-[var(--astro-status-ok)]/80 border-[var(--astro-status-ok)]';
    case 'warn':
      return 'bg-amber-400/80 border-amber-300';
    case 'down':
      return 'bg-[var(--astro-status-err)]/80 border-[var(--astro-status-err)]';
    default:
      return 'bg-[var(--astro-status-unknown)]/60 border-[var(--astro-status-unknown)]';
  }
}

function tooltip(item: EntityListItem): string {
  const parts = [`${item.label}`, t('statusgrid.statusLabel', { status: item.status })];
  if (item.extra) {
    for (const [k, v] of Object.entries(item.extra)) {
      if (typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean') {
        parts.push(`${k}: ${v}`);
      }
    }
  }
  return parts.join('\n');
}
</script>

<template>
  <div class="relative flex h-full w-full flex-col p-2">
    <p
      v-if="cfg.title"
      class="mb-1 text-xs text-[color:var(--astro-text-secondary)]"
    >
      {{ cfg.title }}
    </p>
    <div
      class="grid flex-1 gap-1.5 overflow-auto"
      :style="{ gridTemplateColumns: `repeat(auto-fill, minmax(${cfg.cell_min_px ?? 64}px, 1fr))` }"
    >
      <div
        v-for="item in items"
        :key="item.id"
        :title="tooltip(item)"
        class="flex h-10 items-center justify-center truncate rounded border px-2 text-[10px] font-mono text-white"
        :class="statusClass(item.status)"
      >
        {{ item.label }}
      </div>
      <p
        v-if="items.length === 0"
        class="col-span-full text-center text-[10px] text-[color:var(--astro-text-secondary)]"
      >
        {{ t('statusgrid.empty') }}
      </p>
    </div>
    <div
      v-if="errorMsg"
      class="absolute inset-x-2 bottom-1 truncate rounded bg-red-700/40 px-2 py-1 text-[10px] text-red-100"
      :title="errorMsg"
    >
      {{ errorMsg }}
    </div>
  </div>
</template>
