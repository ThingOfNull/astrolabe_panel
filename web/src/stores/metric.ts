import { defineStore } from 'pinia';
import { computed, ref } from 'vue';

import { getEvents, type MetricSamplePayload } from '@/api/events';
import { getRpc } from '@/api/jsonrpc';
import type { DataPayload, MetricFetchResponse, MetricQuery, Shape } from '@/api/types';

export interface MetricBinding {
  widgetId: number;
  dataSourceId: number;
  query: MetricQuery;
  /** Cold-start hydration interval; overridden by shape default when missing. */
  intervalMs?: number;
}

interface InternalEntry {
  binding: MetricBinding;
  payload?: DataPayload;
  cachedAt?: number;
  error?: string;
  fallbackTimer?: ReturnType<typeof setInterval>;
}

const DEFAULT_INTERVAL_BY_SHAPE: Record<Shape, number> = {
  Scalar: 5_000,
  Categorical: 5_000,
  EntityList: 5_000,
  TimeSeries: 30_000,
};

// SSE is the primary push channel. This slow fallback guards against missed
// pushes (slow-client drops, dev proxies stripping text/event-stream, etc.)
// by triggering a real RPC fetch much less frequently than the old client did.
const FALLBACK_MULTIPLIER = 5;

function intervalFor(b: MetricBinding): number {
  if (b.intervalMs && b.intervalMs > 0) return b.intervalMs;
  return DEFAULT_INTERVAL_BY_SHAPE[b.query.shape] ?? 5_000;
}

export const useMetricStore = defineStore('metric', () => {
  const entries = ref<Record<number, InternalEntry>>({});
  let eventsDisposer: (() => void) | null = null;

  /** Attach a widget binding: hydrate once, install fallback poller, subscribe to SSE. */
  function bind(b: MetricBinding): void {
    unbind(b.widgetId);
    ensureEventsSubscribed();
    const entry: InternalEntry = { binding: b };
    entries.value = { ...entries.value, [b.widgetId]: entry };
    void doFetch(b.widgetId);
    const fallbackMs = intervalFor(b) * FALLBACK_MULTIPLIER;
    entry.fallbackTimer = setInterval(() => void doFetch(b.widgetId), fallbackMs);
  }

  /** Stop timers and drop cache entry for widgetId. */
  function unbind(widgetId: number): void {
    const e = entries.value[widgetId];
    if (e?.fallbackTimer) clearInterval(e.fallbackTimer);
    const { [widgetId]: _omit, ...rest } = entries.value;
    void _omit;
    entries.value = rest;
  }

  function unbindAll(): void {
    for (const id of Object.keys(entries.value)) {
      unbind(Number(id));
    }
    eventsDisposer?.();
    eventsDisposer = null;
  }

  async function doFetch(widgetId: number): Promise<void> {
    const e = entries.value[widgetId];
    if (!e) return;
    try {
      const r = await getRpc().call<MetricFetchResponse>('metric.fetch', {
        data_source_id: e.binding.dataSourceId,
        query: e.binding.query as unknown as Record<string, unknown>,
      });
      applySample(widgetId, r.payload, r.cached_at);
    } catch (err) {
      const msg = err && typeof err === 'object' && 'message' in err
        ? String((err as { message: unknown }).message)
        : String(err);
      entries.value = { ...entries.value, [widgetId]: { ...e, error: msg } };
    }
  }

  function applySample(widgetId: number, payload: DataPayload, cachedAt: number): void {
    const e = entries.value[widgetId];
    if (!e) return;
    entries.value = {
      ...entries.value,
      [widgetId]: { ...e, payload, cachedAt, error: undefined },
    };
  }

  /** One-off fetch without resetting poll interval. */
  function refresh(widgetId: number): Promise<void> {
    return doFetch(widgetId);
  }

  /**
   * Subscribe to metric.sample SSE events once. Each event carries
   * (data_source_id, path, dim) and we match it against every bound entry;
   * this is O(bindings) per event but binding count is small (~dozens).
   */
  function ensureEventsSubscribed(): void {
    if (eventsDisposer) return;
    eventsDisposer = getEvents().on('metric.sample', (ev: MetricSamplePayload) => {
      if (!ev) return;
      for (const [idStr, entry] of Object.entries(entries.value)) {
        const b = entry.binding;
        if (b.dataSourceId !== ev.data_source_id) continue;
        if (b.query.path !== ev.path) continue;
        if ((b.query.dim ?? '') !== (ev.dim ?? '') && (b.query.dim ?? '_') !== (ev.dim ?? '_')) continue;
        applySample(Number(idStr), ev.payload as DataPayload, ev.cached_at);
      }
    });
  }

  const payloadOf = computed(() => (widgetId: number): DataPayload | undefined => entries.value[widgetId]?.payload);
  const errorOf = computed(() => (widgetId: number): string | undefined => entries.value[widgetId]?.error);

  return { entries, bind, unbind, unbindAll, refresh, payloadOf, errorOf };
});
