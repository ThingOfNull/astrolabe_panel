import { defineStore } from 'pinia';
import { ref } from 'vue';

import { getEvents } from '@/api/events';
import { getRpc } from '@/api/jsonrpc';
import type { Rect, Widget } from '@/canvas/types';

export const useWidgetStore = defineStore('widgets', () => {
  const widgets = ref<Widget[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchAll(): Promise<void> {
    loading.value = true;
    error.value = null;
    try {
      const result = await getRpc().call<{ items: Widget[] }>('widget.list');
      widgets.value = result.items ?? [];
    } catch (err) {
      error.value = formatError(err);
    } finally {
      loading.value = false;
    }
  }

  async function create(input: Partial<Widget>): Promise<Widget> {
    const created = await getRpc().call<Widget>('widget.create', input as Record<string, unknown>);
    upsertLocal(created);
    return created;
  }

  async function update(id: number, patch: Partial<Widget>): Promise<Widget> {
    const updated = await getRpc().call<Widget>('widget.update', {
      id,
      ...patch,
    } as Record<string, unknown>);
    upsertLocal(updated);
    return updated;
  }

  async function move(id: number, rect: Rect): Promise<Widget> {
    return update(id, rect);
  }

  // Batch persist positions (multi-drag, marquee move, etc.).
  async function moveMany(updates: { id: number; rect: Rect }[]): Promise<Widget[]> {
    if (updates.length === 0) return [];
    const items = updates.map(({ id, rect }) => {
      const cur = widgets.value.find((w) => w.id === id);
      return {
        id,
        x: rect.x,
        y: rect.y,
        w: rect.w,
        h: rect.h,
        z_index: cur?.z_index ?? 0,
      };
    });
    const result = await getRpc().call<{ items: Widget[] }>('widget.batchUpdate', { items });
    const next = result.items ?? [];
    const byId = new Map(next.map((w) => [w.id, w]));
    widgets.value = widgets.value.map((w) => byId.get(w.id) ?? w);
    return next;
  }

  async function remove(id: number): Promise<void> {
    await getRpc().call('widget.delete', { id });
    removeLocal(id);
  }

  function upsertLocal(w: Widget): void {
    const idx = widgets.value.findIndex((x) => x.id === w.id);
    if (idx >= 0) {
      const next = widgets.value.slice();
      next[idx] = w;
      widgets.value = next;
    } else {
      widgets.value = [...widgets.value, w];
    }
  }

  function removeLocal(id: number): void {
    widgets.value = widgets.value.filter((w) => w.id !== id);
  }

  /**
   * Wire SSE events so other clients' edits (or server-side fixes) reflect
   * locally without polling. Safe to call multiple times: each invocation
   * returns a disposer for the handlers it installed.
   */
  function subscribeEvents(): () => void {
    const ev = getEvents();
    const offCreated = ev.on('widget.created', (payload) => {
      if (isWidget(payload)) upsertLocal(payload);
    });
    const offChanged = ev.on('widget.changed', (payload) => {
      if (isWidget(payload)) upsertLocal(payload);
    });
    const offDeleted = ev.on('widget.deleted', (payload) => {
      if (payload && typeof payload.id === 'number') removeLocal(payload.id);
    });
    return () => {
      offCreated();
      offChanged();
      offDeleted();
    };
  }

  return {
    widgets,
    loading,
    error,
    fetchAll,
    create,
    update,
    move,
    moveMany,
    remove,
    subscribeEvents,
  };
});

function isWidget(v: unknown): v is Widget {
  return !!v && typeof v === 'object' && 'id' in (v as Record<string, unknown>);
}

function formatError(err: unknown): string {
  if (err && typeof err === 'object' && 'message' in err) {
    return String((err as { message: unknown }).message);
  }
  return String(err);
}
