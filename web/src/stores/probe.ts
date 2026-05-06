import { defineStore } from 'pinia';
import { computed, ref } from 'vue';

import { getRpc } from '@/api/jsonrpc';

export interface ProbeStatusItem {
  widget_id: number;
  status: 'ok' | 'down' | 'unknown';
  latency_ms: number;
  checked_at: string;
}

export const useProbeStore = defineStore('probe', () => {
  const statuses = ref<Record<number, ProbeStatusItem>>({});

  async function fetchAll(): Promise<void> {
    try {
      const res = await getRpc().call<{ items: ProbeStatusItem[] }>('probe.status');
      const next: Record<number, ProbeStatusItem> = {};
      for (const item of res.items ?? []) {
        next[item.widget_id] = item;
      }
      statuses.value = next;
    } catch (err) {
      console.warn('[probe] fetch failed', err);
    }
  }

  const statusOf = computed(() => (id: number): 'ok' | 'down' | 'unknown' => {
    return statuses.value[id]?.status ?? 'unknown';
  });

  return { statuses, fetchAll, statusOf };
});
