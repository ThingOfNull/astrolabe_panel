import { defineStore } from 'pinia';
import { computed, ref } from 'vue';

import { getRpc } from '@/api/jsonrpc';
import type { DataPayload, MetricFetchResponse, MetricQuery, Shape } from '@/api/types';

export interface MetricBinding {
  widgetId: number;
  dataSourceId: number;
  query: MetricQuery;
  // Default poll interval by shape (scalar-like 5s, time series 30s).
  intervalMs?: number;
}

interface InternalEntry {
  binding: MetricBinding;
  payload?: DataPayload;
  cachedAt?: number;
  error?: string;
  timer?: ReturnType<typeof setInterval>;
}

const DEFAULT_INTERVAL_BY_SHAPE: Record<Shape, number> = {
  Scalar: 5_000,
  Categorical: 5_000,
  EntityList: 5_000,
  TimeSeries: 30_000,
};

function intervalFor(b: MetricBinding): number {
  if (b.intervalMs && b.intervalMs > 0) return b.intervalMs;
  return DEFAULT_INTERVAL_BY_SHAPE[b.query.shape] ?? 5_000;
}

export const useMetricStore = defineStore('metric', () => {
  const entries = ref<Record<number, InternalEntry>>({});

  /** Attach metric polling for a widget (replaces binding, restarts timers). */
  function bind(b: MetricBinding): void {
    unbind(b.widgetId);
    const entry: InternalEntry = { binding: b };
    entries.value = { ...entries.value, [b.widgetId]: entry };
    void doFetch(b.widgetId);
    entry.timer = setInterval(() => void doFetch(b.widgetId), intervalFor(b));
  }

  /** Stop timers and drop cache entry for widgetId. */
  function unbind(widgetId: number): void {
    const e = entries.value[widgetId];
    if (e?.timer) clearInterval(e.timer);
    const { [widgetId]: _omit, ...rest } = entries.value;
    void _omit;
    entries.value = rest;
  }

  function unbindAll(): void {
    for (const id of Object.keys(entries.value)) {
      unbind(Number(id));
    }
  }

  async function doFetch(widgetId: number): Promise<void> {
    const e = entries.value[widgetId];
    if (!e) return;
    try {
      const r = await getRpc().call<MetricFetchResponse>('metric.fetch', {
        data_source_id: e.binding.dataSourceId,
        query: e.binding.query as unknown as Record<string, unknown>,
      });
      const next: InternalEntry = { ...e, payload: r.payload, cachedAt: r.cached_at, error: undefined };
      entries.value = { ...entries.value, [widgetId]: next };
    } catch (err) {
      const msg = err && typeof err === 'object' && 'message' in err
        ? String((err as { message: unknown }).message)
        : String(err);
      entries.value = { ...entries.value, [widgetId]: { ...e, error: msg } };
    }
  }

  /** One-off fetch without resetting poll interval. */
  function refresh(widgetId: number): Promise<void> {
    return doFetch(widgetId);
  }

  const payloadOf = computed(() => (widgetId: number): DataPayload | undefined => entries.value[widgetId]?.payload);
  const errorOf = computed(() => (widgetId: number): string | undefined => entries.value[widgetId]?.error);

  return { entries, bind, unbind, unbindAll, refresh, payloadOf, errorOf };
});
