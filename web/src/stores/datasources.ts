import { defineStore } from 'pinia';
import { computed, ref } from 'vue';

import { getRpc } from '@/api/jsonrpc';
import type { DataSourceView, MetricTree } from '@/api/types';

export const useDataSourceStore = defineStore('datasources', () => {
  const items = ref<DataSourceView[]>([]);
  const types = ref<string[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  // Per datasource id: cached metric trees (lazy-loaded).
  const trees = ref<Record<number, MetricTree>>({});

  async function fetchTypes(): Promise<void> {
    try {
      const r = await getRpc().call<{ types: string[] }>('datasource.types');
      types.value = r.types ?? [];
    } catch (err) {
      error.value = format(err);
    }
  }

  async function fetchAll(): Promise<void> {
    loading.value = true;
    error.value = null;
    try {
      const r = await getRpc().call<{ items: DataSourceView[] }>('datasource.list');
      items.value = r.items ?? [];
    } catch (err) {
      error.value = format(err);
    } finally {
      loading.value = false;
    }
  }

  async function create(input: Partial<DataSourceView>): Promise<DataSourceView> {
    const out = await getRpc().call<DataSourceView>('datasource.create', input as Record<string, unknown>);
    items.value = [...items.value, out];
    return out;
  }

  async function update(id: number, patch: Partial<DataSourceView>): Promise<DataSourceView> {
    const out = await getRpc().call<DataSourceView>('datasource.update', {
      id,
      ...patch,
    } as Record<string, unknown>);
    items.value = items.value.map((d) => (d.id === id ? out : d));
    delete trees.value[id];
    return out;
  }

  async function remove(id: number): Promise<void> {
    await getRpc().call('datasource.delete', { id });
    items.value = items.value.filter((d) => d.id !== id);
    delete trees.value[id];
  }

  async function testConnect(payload: Record<string, unknown>): Promise<{ ok: boolean; error?: string }> {
    return getRpc().call<{ ok: boolean; error?: string }>('datasource.testConnect', payload);
  }

  async function discover(id: number, force = false): Promise<MetricTree> {
    if (!force && trees.value[id]) {
      return trees.value[id];
    }
    const r = await getRpc().call<{ tree: MetricTree }>('datasource.discover', { id });
    trees.value = { ...trees.value, [id]: r.tree };
    return r.tree;
  }

  const byId = computed(() => (id: number): DataSourceView | undefined =>
    items.value.find((d) => d.id === id),
  );

  return {
    items,
    types,
    loading,
    error,
    trees,
    fetchTypes,
    fetchAll,
    create,
    update,
    remove,
    testConnect,
    discover,
    byId,
  };
});

function format(err: unknown): string {
  if (err && typeof err === 'object' && 'message' in err) {
    return String((err as { message: unknown }).message);
  }
  return String(err);
}
