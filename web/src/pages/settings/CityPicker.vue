<script setup lang="ts">
/**
 * City picker with search and pinyin grouping (Meizu city list).
 */

import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { onClickOutside } from '@vueuse/core';

import type { LetterGroup } from '@/lib/meizuCityIndex';
import {
  filterMeizuCities,
  findMeizuCityById,
  groupRowsByLetter,
} from '@/lib/meizuCityIndex';

const LIMIT = 400;

const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    modelValue: number | null;
    disabled?: boolean;
  }>(),
  { disabled: false },
);

const emit = defineEmits<{
  (e: 'update:modelValue', id: number | null): void;
  (e: 'pickLabel', label: string): void;
}>();

const rootEl = ref<HTMLElement | null>(null);
const open = ref(false);
const query = ref('');

watch(
  () => props.modelValue,
  () => {
    const row =
      props.modelValue != null && props.modelValue > 0
        ? findMeizuCityById(props.modelValue)
        : undefined;
    if (row) {
      query.value = row.countyname;
    } else {
      query.value = '';
    }
  },
  { immediate: true },
);

const filteredFlat = computed(() => filterMeizuCities(query.value, LIMIT));

const grouped = computed<LetterGroup[]>(() =>
  groupRowsByLetter(filteredFlat.value),
);

const panelHint = computed(() =>
  query.value.trim()
    ? t('cityPicker.hintMatch', { limit: LIMIT })
    : t('cityPicker.hintBrowse', { limit: LIMIT }),
);

onClickOutside(rootEl, () => {
  open.value = false;
});

function togglePanel(): void {
  if (props.disabled) return;
  open.value = !open.value;
}

function pick(row: { areaid: number; countyname: string }): void {
  emit('update:modelValue', row.areaid);
  emit('pickLabel', row.countyname);
  query.value = row.countyname;
  open.value = false;
}

function onInputFocus(): void {
  if (props.disabled) return;
  open.value = true;
}

function onInputBlur(): void {
  window.setTimeout(() => {
    syncQueryFromBlur();
  }, 120);
}

function clearSelection(): void {
  emit('update:modelValue', null);
  query.value = '';
  emit('pickLabel', '');
}

function syncQueryFromBlur(): void {
  const row =
    props.modelValue != null && props.modelValue > 0
      ? findMeizuCityById(props.modelValue)
      : undefined;
  if (row) {
    query.value = row.countyname;
    return;
  }
  query.value = '';
}
</script>

<template>
  <div
    ref="rootEl"
    class="relative"
  >
    <div class="flex gap-2">
      <div class="relative min-w-0 flex-1">
        <input
          v-model="query"
          :disabled="disabled"
          type="text"
          class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-[color:var(--astro-bg-base)] px-3 py-2 text-sm text-[color:var(--astro-text-primary)] outline-none placeholder:text-[color:var(--astro-text-secondary)] focus:border-[color:var(--astro-accent)]"
          :placeholder="t('cityPicker.placeholder')"
          @focus="onInputFocus"
          @keydown.escape="open = false"
          @blur="onInputBlur"
        >
        <button
          type="button"
          class="absolute right-2 top-1/2 z-[1] -translate-y-1/2 px-1 text-xs text-[color:var(--astro-text-secondary)] hover:text-[color:var(--astro-text-primary)]"
          tabindex="-1"
          :disabled="disabled"
          :title="t('cityPicker.expand')"
          @mousedown.prevent="togglePanel()"
        >
          ▾
        </button>
      </div>
      <button
        v-if="modelValue != null && modelValue > 0"
        type="button"
        class="shrink-0 rounded-md border border-[color:var(--astro-glass-border)] px-2 py-1 text-xs text-[color:var(--astro-text-secondary)] hover:bg-white/10"
        :disabled="disabled"
        @click="clearSelection"
      >
        {{ t('cityPicker.clear') }}
      </button>
    </div>

    <div
      v-if="open && !disabled"
      class="absolute left-0 right-0 top-full z-50 mt-1 max-h-72 overflow-y-auto rounded-lg border border-[color:var(--astro-glass-border)] bg-[color:var(--astro-bg-base)] py-2 shadow-xl"
      @mousedown.prevent
    >
      <p class="sticky top-0 z-[1] border-b border-[color:var(--astro-glass-border)] bg-[color:var(--astro-bg-base)] px-3 py-2 text-[10px] text-[color:var(--astro-text-secondary)]">
        {{ panelHint }}
      </p>
      <template v-if="grouped.length > 0">
        <template
          v-for="g in grouped"
          :key="g.letter"
        >
          <div class="sticky top-8 bg-[color:var(--astro-glass-bg)]/95 px-3 py-1 text-[11px] font-semibold uppercase tracking-wide text-[color:var(--astro-accent)] backdrop-blur-sm">
            {{ g.letter }}
          </div>
          <button
            v-for="r in g.items"
            :key="r.areaid"
            type="button"
            class="flex w-full items-center justify-between gap-2 px-3 py-1.5 text-left text-sm text-[color:var(--astro-text-primary)] hover:bg-[color:var(--astro-accent)]/15"
            :class="
              modelValue === r.areaid ? 'bg-[color:var(--astro-accent)]/20' : ''
            "
            @click="pick(r)"
          >
            <span>{{ r.countyname }}</span>
            <span class="astro-mono-num shrink-0 text-[10px] text-[color:var(--astro-text-secondary)]">
              {{ r.areaid }}
            </span>
          </button>
        </template>
      </template>
      <div
        v-else
        class="px-3 py-6 text-center text-xs text-[color:var(--astro-text-secondary)]"
      >
        {{ t('cityPicker.noMatch') }}
      </div>
    </div>
  </div>
</template>
