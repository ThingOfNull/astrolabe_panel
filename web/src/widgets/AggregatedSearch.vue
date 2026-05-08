<script setup lang="ts">
import { Icon } from '@iconify/vue';
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import type { Widget } from '@/canvas/types';

import { DEFAULT_SEARCH_ENGINES, type AggregatedSearchConfig, type SearchEngine } from './types';

const props = defineProps<{
  widget: Widget;
  interactive: boolean;
  focusKey?: number;
}>();

const { t } = useI18n();

const STORAGE_KEY = 'astrolabe.search.engineId';

const cfg = computed<AggregatedSearchConfig>(() => {
  const c = (props.widget.config ?? {}) as Partial<AggregatedSearchConfig>;
  const engines = c.engines && c.engines.length > 0 ? c.engines : DEFAULT_SEARCH_ENGINES;
  return {
    engines,
    default_engine_id: c.default_engine_id ?? engines[0]?.id ?? 'google',
  };
});

// Default footprint in grid units (60x6).
const DEFAULT_W = 60;
const DEFAULT_H = 6;
const scale = computed(() => {
  const rw = props.widget.w / DEFAULT_W;
  const rh = props.widget.h / DEFAULT_H;
  return Math.max(1, Math.min(rw, rh));
});
const s = computed(() => scale.value);

const inputRef = ref<HTMLInputElement | null>(null);
const dropdownOpen = ref(false);
const query = ref('');
const currentEngineId = ref<string>('');

function ensureEngineSelected(): void {
  const engines = cfg.value.engines;
  const stored = typeof window !== 'undefined' ? window.localStorage.getItem(STORAGE_KEY) : null;
  const fromStorage = engines.find((e) => e.id === stored);
  const fromDefault = engines.find((e) => e.id === cfg.value.default_engine_id);
  currentEngineId.value = fromStorage?.id ?? fromDefault?.id ?? engines[0]?.id ?? 'google';
}

const currentEngine = computed<SearchEngine | undefined>(() =>
  cfg.value.engines.find((e) => e.id === currentEngineId.value),
);

function selectEngine(id: string): void {
  currentEngineId.value = id;
  dropdownOpen.value = false;
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(STORAGE_KEY, id);
  }
  inputRef.value?.focus();
}

function submit(): void {
  if (!props.interactive) return;
  const q = query.value.trim();
  if (!q) return;
  const eng = currentEngine.value;
  if (!eng) return;
  const target = eng.url.replace('{q}', encodeURIComponent(q));
  window.open(target, '_blank', 'noopener');
  query.value = '';
}

function focusInput(): void {
  inputRef.value?.focus();
  inputRef.value?.select();
}

onMounted(ensureEngineSelected);
watch(() => cfg.value.engines.length, ensureEngineSelected);

watch(
  () => props.focusKey,
  () => {
    if (props.interactive) {
      focusInput();
    }
  },
);

function closeOnOutside(e: MouseEvent): void {
  if (!dropdownOpen.value) return;
  const tgt = e.target as Node;
  const root = inputRef.value?.parentElement?.parentElement;
  if (root && !root.contains(tgt)) {
    dropdownOpen.value = false;
  }
}

onMounted(() => document.addEventListener('click', closeOnOutside));
onUnmounted(() => document.removeEventListener('click', closeOnOutside));

const btnStyle = computed(() => ({
  gap: `${Math.round(4 * s.value)}px`,
  padding: `${Math.round(4 * s.value)}px ${Math.round(8 * s.value)}px`,
  fontSize: `${Math.round(14 * s.value)}px`,
}));

const dropdownItemStyle = computed(() => ({
  gap: `${Math.round(8 * s.value)}px`,
  padding: `${Math.round(4 * s.value)}px ${Math.round(8 * s.value)}px`,
  fontSize: `${Math.round(14 * s.value)}px`,
}));

const engineIconSize = computed(() => Math.round(18 * s.value));
const chevronIconSize = computed(() => Math.round(14 * s.value));
const dropdownIconSize = computed(() => Math.round(16 * s.value));

const formStyle = computed(() => ({
  gap: `${Math.round(8 * s.value)}px`,
  paddingLeft: `${Math.round(16 * s.value)}px`,
  paddingRight: `${Math.round(16 * s.value)}px`,
}));

const inputStyle = computed(() => ({
  padding: `${Math.round(8 * s.value)}px ${Math.round(12 * s.value)}px`,
  fontSize: `${Math.round(14 * s.value)}px`,
}));

const submitStyle = computed(() => ({
  padding: `${Math.round(8 * s.value)}px ${Math.round(12 * s.value)}px`,
  fontSize: `${Math.round(14 * s.value)}px`,
}));

const dropdownStyle = computed(() => ({
  marginTop: `${Math.round(4 * s.value)}px`,
  minWidth: `${Math.round(160 * s.value)}px`,
  padding: `${Math.round(4 * s.value)}px`,
}));
</script>

<template>
  <form
    class="flex h-full w-full items-center"
    :class="{ 'pointer-events-none opacity-70': !interactive }"
    :style="formStyle"
    @submit.prevent="submit"
  >
    <div class="relative">
      <button
        type="button"
        class="flex items-center rounded-md border border-[color:var(--astro-glass-border)] hover:bg-white/5"
        :style="btnStyle"
        @click="dropdownOpen = !dropdownOpen"
      >
        <Icon
          v-if="currentEngine?.icon?.value"
          :icon="currentEngine.icon.value"
          :width="engineIconSize"
          :height="engineIconSize"
        />
        <span>{{ currentEngine?.label ?? t('search.defaultButton') }}</span>
        <Icon
          icon="mdi:chevron-down"
          :width="chevronIconSize"
          :height="chevronIconSize"
        />
      </button>
      <ul
        v-if="dropdownOpen"
        class="absolute left-0 top-full z-10 rounded-md border border-[color:var(--astro-glass-border)] bg-[color:var(--astro-glass-bg)] shadow-lg"
        :style="dropdownStyle"
      >
        <li
          v-for="eng in cfg.engines"
          :key="eng.id"
          class="flex cursor-pointer items-center rounded hover:bg-white/10"
          :style="dropdownItemStyle"
          @click="selectEngine(eng.id)"
        >
          <Icon
            v-if="eng.icon?.value"
            :icon="eng.icon.value"
            :width="dropdownIconSize"
            :height="dropdownIconSize"
          />
          {{ eng.label }}
        </li>
      </ul>
    </div>
    <input
      ref="inputRef"
      v-model="query"
      type="search"
      :placeholder="t('search.focusPlaceholder')"
      class="flex-1 rounded-md border border-[color:var(--astro-glass-border)] bg-transparent focus:outline-none focus:ring-1 focus:ring-[color:var(--astro-accent)]"
      :style="inputStyle"
    >
    <button
      type="submit"
      class="rounded-md border border-[color:var(--astro-glass-border)] hover:bg-white/5"
      :style="submitStyle"
    >
      {{ t('search.button') }}
    </button>
  </form>
</template>
