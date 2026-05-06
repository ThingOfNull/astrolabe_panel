<script setup lang="ts">
// Threshold rows: numeric value + CSS color + add/remove helpers (v-model list).

import { computed } from 'vue';

import type { Threshold } from '@/widgets/thresholds';

const props = defineProps<{
  modelValue?: Threshold[];
}>();
const emit = defineEmits<{
  'update:modelValue': [value: Threshold[]];
}>();

const items = computed<Threshold[]>(() => props.modelValue ?? []);

function update(idx: number, patch: Partial<Threshold>): void {
  const next = [...items.value];
  next[idx] = { ...next[idx], ...patch };
  emit('update:modelValue', next);
}

function add(): void {
  const last = items.value[items.value.length - 1];
  const nextValue = last ? last.value + 10 : 80;
  emit('update:modelValue', [...items.value, { value: nextValue, color: '#ff3131' }]);
}

function remove(idx: number): void {
  const next = items.value.filter((_, i) => i !== idx);
  emit('update:modelValue', next);
}
</script>

<template>
  <div class="space-y-1">
    <p class="text-xs text-[color:var(--astro-text-secondary)]">
      阈值色（按数值升序匹配）：
    </p>
    <div
      v-for="(t, idx) in items"
      :key="idx"
      class="flex items-center gap-2"
    >
      <span class="text-[10px] text-[color:var(--astro-text-secondary)]">≥</span>
      <input
        :value="t.value"
        type="number"
        class="w-20 rounded border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1 text-xs"
        @input="(e) => update(idx, { value: Number((e.target as HTMLInputElement).value) })"
      >
      <input
        :value="t.color"
        type="color"
        class="h-6 w-10 rounded border border-[color:var(--astro-glass-border)] bg-transparent"
        @input="(e) => update(idx, { color: (e.target as HTMLInputElement).value })"
      >
      <input
        :value="t.color"
        type="text"
        class="flex-1 rounded border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1 font-mono text-xs"
        @input="(e) => update(idx, { color: (e.target as HTMLInputElement).value })"
      >
      <button
        type="button"
        class="rounded border border-[color:var(--astro-glass-border)] px-2 py-1 text-[10px] hover:bg-white/5"
        @click="remove(idx)"
      >
        删除
      </button>
    </div>
    <button
      type="button"
      class="rounded border border-[color:var(--astro-glass-border)] px-2 py-1 text-[10px] hover:bg-white/5"
      @click="add"
    >
      + 添加阈值
    </button>
  </div>
</template>
