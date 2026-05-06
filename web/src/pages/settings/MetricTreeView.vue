<script setup lang="ts">
import { computed, ref } from 'vue';

import type { MetricNode, Shape } from '@/api/types';

const props = defineProps<{
  nodes: MetricNode[];
  /** When set, dim leaves that do not support this shape. */
  filterShape?: Shape;
  selectedPath?: string;
}>();

const emit = defineEmits<{
  (e: 'select', payload: { node: MetricNode; shape: Shape }): void;
}>();

const expanded = ref<Record<string, boolean>>({});

function toggle(path: string): void {
  expanded.value = { ...expanded.value, [path]: !expanded.value[path] };
}

function isCompatible(node: MetricNode): boolean {
  if (!props.filterShape) return true;
  return node.shapes.includes(props.filterShape);
}

function pickShape(node: MetricNode): Shape | undefined {
  if (props.filterShape && node.shapes.includes(props.filterShape)) return props.filterShape;
  return node.shapes[0];
}

const allExpanded = computed(() => expanded.value);

function onLeafClick(node: MetricNode): void {
  const shape = pickShape(node);
  if (!shape) return;
  emit('select', { node, shape });
}
</script>

<template>
  <ul class="space-y-1 text-sm">
    <li
      v-for="node in nodes"
      :key="node.path"
    >
      <div
        v-if="!node.leaf"
        class="flex cursor-pointer items-center gap-1 rounded px-2 py-1 hover:bg-white/5"
        @click="toggle(node.path)"
      >
        <span class="inline-block w-4 text-xs text-[color:var(--astro-text-secondary)]">
          {{ allExpanded[node.path] ? '▾' : '▸' }}
        </span>
        <span>{{ node.label }}</span>
        <span class="ml-auto font-mono text-[10px] text-[color:var(--astro-text-secondary)]">
          {{ node.path }}
        </span>
      </div>
      <div
        v-else
        class="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-white/5"
        :class="{
          'opacity-40 pointer-events-none': !isCompatible(node),
          'bg-white/10': selectedPath === node.path,
        }"
        @click="onLeafClick(node)"
      >
        <span class="inline-block w-4" />
        <span>{{ node.label }}</span>
        <span class="ml-2 text-[10px] text-[color:var(--astro-text-secondary)]">
          {{ node.shapes.join('/') }}{{ node.unit ? ' · ' + node.unit : '' }}
        </span>
        <span class="ml-auto font-mono text-[10px] text-[color:var(--astro-text-secondary)]">
          {{ node.path }}
        </span>
      </div>
      <div
        v-if="!node.leaf && allExpanded[node.path]"
        class="ml-4 border-l border-[color:var(--astro-glass-border)] pl-2"
      >
        <MetricTreeView
          :nodes="node.children ?? []"
          :filter-shape="filterShape"
          :selected-path="selectedPath"
          @select="(p) => emit('select', p)"
        />
      </div>
    </li>
  </ul>
</template>

<script lang="ts">
// Vue needs explicit name for self-recursive SFC.
export default { name: 'MetricTreeView' };
</script>
