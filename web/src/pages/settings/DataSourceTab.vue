<script setup lang="ts">
import { Icon } from '@iconify/vue';
import { computed, onMounted, ref } from 'vue';

import type { DataSourceView } from '@/api/types';
import { useDataSourceStore } from '@/stores/datasources';

import MetricTreeView from './MetricTreeView.vue';

const dsStore = useDataSourceStore();

const draftMode = ref<'list' | 'edit'>('list');
const draft = ref<Partial<DataSourceView> & { auth?: string; extra?: string }>(
  emptyDraft(),
);
const draftId = ref<number | null>(null);
const testResult = ref<{ ok: boolean; error?: string } | null>(null);
const expandedTreeId = ref<number | null>(null);

onMounted(async () => {
  await Promise.all([dsStore.fetchTypes(), dsStore.fetchAll()]);
});

function emptyDraft() {
  return {
    name: '',
    type: 'local',
    endpoint: '',
    auth: '',
    extra: '',
  } as Partial<DataSourceView> & { auth?: string; extra?: string };
}

function startCreate(): void {
  draftId.value = null;
  draft.value = emptyDraft();
  if (dsStore.types.length > 0) draft.value.type = dsStore.types[0];
  testResult.value = null;
  draftMode.value = 'edit';
}

function startEdit(d: DataSourceView): void {
  draftId.value = d.id;
  draft.value = {
    name: d.name,
    type: d.type,
    endpoint: d.endpoint,
    auth: stringifyJSON(d.auth),
    extra: stringifyJSON(d.extra),
  };
  testResult.value = null;
  draftMode.value = 'edit';
}

function backToList(): void {
  draftMode.value = 'list';
  draftId.value = null;
  testResult.value = null;
}

function stringifyJSON(v: unknown): string {
  if (v == null) return '';
  if (typeof v === 'string') return v;
  return JSON.stringify(v, null, 2);
}

function parseOptionalJSON(s: string | undefined): unknown {
  if (!s) return null;
  try {
    return JSON.parse(s);
  } catch {
    return null;
  }
}

function buildPayload(): Record<string, unknown> {
  return {
    name: draft.value.name,
    type: draft.value.type,
    endpoint: draft.value.endpoint ?? '',
    auth: parseOptionalJSON(draft.value.auth),
    extra: parseOptionalJSON(draft.value.extra),
  };
}

async function onTestConnect(): Promise<void> {
  testResult.value = null;
  try {
    const payload = buildPayload();
    if (draftId.value) {
      payload.id = draftId.value;
    }
    const r = await dsStore.testConnect(payload);
    testResult.value = r;
  } catch (err) {
    testResult.value = { ok: false, error: format(err) };
  }
}

async function onSave(): Promise<void> {
  try {
    if (draftId.value) {
      await dsStore.update(draftId.value, buildPayload() as Partial<DataSourceView>);
    } else {
      await dsStore.create(buildPayload() as Partial<DataSourceView>);
    }
    backToList();
  } catch (err) {
    testResult.value = { ok: false, error: format(err) };
  }
}

async function onDelete(d: DataSourceView): Promise<void> {
  if (!window.confirm(`确认删除数据源 "${d.name}"？引用此数据源的 widget 将解除绑定。`)) {
    return;
  }
  try {
    await dsStore.remove(d.id);
  } catch (err) {
    testResult.value = { ok: false, error: format(err) };
  }
}

async function toggleTree(id: number): Promise<void> {
  if (expandedTreeId.value === id) {
    expandedTreeId.value = null;
    return;
  }
  expandedTreeId.value = id;
  if (!dsStore.trees[id]) {
    try {
      await dsStore.discover(id);
    } catch (err) {
      testResult.value = { ok: false, error: format(err) };
      expandedTreeId.value = null;
    }
  }
}

const items = computed(() => dsStore.items);

function format(err: unknown): string {
  if (err && typeof err === 'object' && 'message' in err) {
    return String((err as { message: unknown }).message);
  }
  return String(err);
}

function healthDot(h: string | undefined): string {
  if (h === 'ok') return 'bg-[var(--astro-status-ok)]';
  if (h === 'error') return 'bg-[var(--astro-status-err)]';
  return 'bg-[var(--astro-status-unknown)]';
}
</script>

<template>
  <div class="space-y-3 text-sm">
    <div v-if="draftMode === 'list'">
      <div class="flex items-center justify-between">
        <p class="text-xs text-[color:var(--astro-text-secondary)]">
          数据源驱动 Gauge / BigNumber 等 BI 组件。
        </p>
        <button
          type="button"
          class="rounded border border-[color:var(--astro-glass-border)] px-2 py-1 text-xs hover:bg-white/5"
          @click="startCreate"
        >
          + 添加
        </button>
      </div>

      <ul class="mt-3 space-y-2">
        <li
          v-for="d in items"
          :key="d.id"
          class="rounded-md border border-[color:var(--astro-glass-border)] bg-black/20"
        >
          <div class="flex items-center gap-2 px-3 py-2">
            <span
              class="inline-block h-2 w-2 rounded-full"
              :class="healthDot(d.last_health)"
            />
            <div class="min-w-0 flex-1">
              <div class="truncate font-medium">
                {{ d.name }}
              </div>
              <div class="truncate text-[10px] text-[color:var(--astro-text-secondary)]">
                {{ d.type }}{{ d.endpoint ? ' · ' + d.endpoint : '' }}
              </div>
            </div>
            <button
              type="button"
              class="rounded px-2 py-1 text-xs hover:bg-white/5"
              @click="toggleTree(d.id)"
            >
              <Icon
                icon="mdi:tree"
                width="14"
                height="14"
              />
            </button>
            <button
              type="button"
              class="rounded px-2 py-1 text-xs hover:bg-white/5"
              @click="startEdit(d)"
            >
              编辑
            </button>
            <button
              type="button"
              class="rounded bg-red-600/40 px-2 py-1 text-xs hover:bg-red-600/70"
              @click="onDelete(d)"
            >
              删除
            </button>
          </div>
          <div
            v-if="expandedTreeId === d.id"
            class="border-t border-[color:var(--astro-glass-border)] px-3 py-2"
          >
            <MetricTreeView
              v-if="dsStore.trees[d.id]"
              :nodes="dsStore.trees[d.id].roots"
            />
            <p
              v-else
              class="text-xs text-[color:var(--astro-text-secondary)]"
            >
              加载指标树中…
            </p>
          </div>
        </li>
        <li
          v-if="items.length === 0"
          class="text-xs text-[color:var(--astro-text-secondary)]"
        >
          暂无数据源。点击右上角 + 添加。
        </li>
      </ul>
    </div>

    <div
      v-else
      class="space-y-3"
    >
      <div class="flex items-center justify-between">
        <h3 class="text-sm font-semibold">
          {{ draftId ? '编辑数据源' : '添加数据源' }}
        </h3>
        <button
          type="button"
          class="text-xs text-[color:var(--astro-text-secondary)] hover:text-[color:var(--astro-text-primary)]"
          @click="backToList"
        >
          返回列表
        </button>
      </div>
      <label class="block">
        <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">名称</span>
        <input
          v-model="draft.name"
          type="text"
          class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
        >
      </label>
      <label class="block">
        <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">类型</span>
        <select
          v-model="draft.type"
          class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
        >
          <option
            v-for="t in dsStore.types"
            :key="t"
            :value="t"
          >{{ t }}</option>
        </select>
      </label>
      <label class="block">
        <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">
          Endpoint（local 留空；docker 默认 unix:///var/run/docker.sock）
        </span>
        <input
          v-model="draft.endpoint"
          type="text"
          class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2 font-mono text-xs"
        >
      </label>
      <details>
        <summary class="cursor-pointer text-xs text-[color:var(--astro-text-secondary)]">
          高级（auth / extra JSON）
        </summary>
        <label class="mt-2 block">
          <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">auth (JSON)</span>
          <textarea
            v-model="draft.auth"
            rows="3"
            class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2 font-mono text-xs"
          />
        </label>
        <label class="mt-2 block">
          <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">extra (JSON)</span>
          <textarea
            v-model="draft.extra"
            rows="3"
            class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2 font-mono text-xs"
          />
        </label>
      </details>
      <div
        v-if="testResult"
        class="rounded p-2 text-xs"
        :class="testResult.ok ? 'bg-emerald-700/20 text-emerald-200' : 'bg-red-700/20 text-red-200'"
      >
        {{ testResult.ok ? '连接测试成功' : ('连接失败：' + (testResult.error ?? '')) }}
      </div>
      <div class="flex justify-end gap-2 pt-2">
        <button
          type="button"
          class="rounded-md border border-[color:var(--astro-glass-border)] px-3 py-1.5 text-xs hover:bg-white/5"
          @click="onTestConnect"
        >
          测试连接
        </button>
        <button
          type="button"
          class="rounded-md bg-[color:var(--astro-accent)] px-3 py-1.5 text-xs text-black hover:opacity-90"
          @click="onSave"
        >
          保存
        </button>
      </div>
    </div>
  </div>
</template>
