<script setup lang="ts">
// Icon picker: ICONIFY search, REMOTE URL, INTERNAL multipart /api/upload.

import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import ThemedFileTrigger from '@/components/ThemedFileTrigger.vue';
import { httpDeleteUpload, httpListUploads, httpUpload } from '@/api/httpUpload';
import { getRpc } from '@/api/jsonrpc';
import { formatRpcError } from '@/lib/rpcError';

const { t } = useI18n();

interface Props {
  modelValue: { icon_type: 'ICONIFY' | 'REMOTE' | 'INTERNAL'; icon_value: string };
}
const props = defineProps<Props>();
const emit = defineEmits<{
  'update:modelValue': [value: Props['modelValue']];
}>();

const tab = computed({
  get: () => props.modelValue.icon_type,
  set: (v) => emit('update:modelValue', { icon_type: v, icon_value: props.modelValue.icon_value }),
});

const value = computed({
  get: () => props.modelValue.icon_value,
  set: (v) =>
    emit('update:modelValue', { icon_type: props.modelValue.icon_type, icon_value: v }),
});

// ---- ICONIFY search ----
const query = ref('');
const searching = ref(false);
const searchError = ref<string | null>(null);
const searchResults = ref<string[]>([]);
let searchTimer: ReturnType<typeof setTimeout> | null = null;

async function runSearch(): Promise<void> {
  searchError.value = null;
  if (query.value.trim().length < 2) {
    searchResults.value = [];
    return;
  }
  searching.value = true;
  try {
    const r = await getRpc().call<{ icons: string[]; total: number }>('iconify.search', {
      query: query.value.trim(),
      limit: 60,
    });
    searchResults.value = r.icons ?? [];
  } catch (err) {
    searchError.value = formatRpcError(err, t);
  } finally {
    searching.value = false;
  }
}

watch(query, () => {
  if (searchTimer) clearTimeout(searchTimer);
  searchTimer = setTimeout(runSearch, 250);
});

// ---- INTERNAL uploads ----
const uploads = ref<{ name: string; url: string }[]>([]);
const uploadError = ref<string | null>(null);
const uploading = ref(false);

async function refreshUploads(): Promise<void> {
  uploadError.value = null;
  try {
    const r = await httpListUploads();
    uploads.value = r.items ?? [];
  } catch (err) {
    uploadError.value = formatRpcError(err, t);
  }
}

async function onUploadFile(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  uploadError.value = null;
  uploading.value = true;
  try {
    if (file.size > 1 << 20) {
      throw new Error(t('iconPicker.tooLarge'));
    }
    const r = await httpUpload('icon', file);
    await refreshUploads();
    value.value = r.name;
  } catch (err) {
    uploadError.value = formatRpcError(err, t);
  } finally {
    uploading.value = false;
    input.value = '';
  }
}

async function onDeleteUpload(name: string): Promise<void> {
  uploadError.value = null;
  try {
    await httpDeleteUpload(name);
    await refreshUploads();
    if (value.value === name || value.value.endsWith('/' + name)) {
      value.value = '';
    }
  } catch (err) {
    uploadError.value = formatRpcError(err, t);
  }
}

watch(
  () => tab.value,
  (t) => {
    if (t === 'INTERNAL' && uploads.value.length === 0) {
      void refreshUploads();
    }
  },
  { immediate: true },
);
</script>

<template>
  <div class="space-y-3 text-sm">
    <div class="flex gap-1 rounded border border-[color:var(--astro-glass-border)] p-1">
      <button
        type="button"
        class="flex-1 rounded px-2 py-1 text-xs"
        :class="tab === 'ICONIFY' ? 'bg-[color:var(--astro-accent)] text-black' : ''"
        @click="tab = 'ICONIFY'"
      >
        {{ t('iconPicker.tabIconify') }}
      </button>
      <button
        type="button"
        class="flex-1 rounded px-2 py-1 text-xs"
        :class="tab === 'INTERNAL' ? 'bg-[color:var(--astro-accent)] text-black' : ''"
        @click="tab = 'INTERNAL'"
      >
        {{ t('iconPicker.tabInternal') }}
      </button>
      <button
        type="button"
        class="flex-1 rounded px-2 py-1 text-xs"
        :class="tab === 'REMOTE' ? 'bg-[color:var(--astro-accent)] text-black' : ''"
        @click="tab = 'REMOTE'"
      >
        {{ t('iconPicker.tabRemote') }}
      </button>
    </div>

    <!-- Current value -->
    <label class="block">
      <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('iconPicker.currentIcon') }}</span>
      <input
        v-model="value"
        type="text"
        :placeholder="tab === 'ICONIFY' ? t('iconPicker.placeholderIconify') : tab === 'INTERNAL' ? t('iconPicker.placeholderInternal') : t('iconPicker.placeholderRemote')"
        class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2 font-mono text-xs"
      >
    </label>

    <!-- ICONIFY search -->
    <div
      v-if="tab === 'ICONIFY'"
      class="space-y-2"
    >
      <input
        v-model="query"
        type="text"
        :placeholder="t('iconPicker.searchPlaceholder')"
        class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
      >
      <p
        v-if="searching"
        class="text-xs text-[color:var(--astro-text-secondary)]"
      >
        {{ t('iconPicker.searching') }}
      </p>
      <p
        v-if="searchError"
        class="text-xs text-red-300"
      >
        {{ searchError }}
      </p>
      <div
        v-if="searchResults.length"
        class="grid max-h-56 grid-cols-6 gap-2 overflow-y-auto rounded border border-[color:var(--astro-glass-border)] p-2"
      >
        <button
          v-for="icon in searchResults"
          :key="icon"
          type="button"
          class="flex h-12 flex-col items-center justify-center rounded border border-transparent text-[10px] hover:border-[color:var(--astro-accent)]"
          :class="{ 'border-[color:var(--astro-accent)]': value === icon }"
          :title="icon"
          @click="value = icon"
        >
          <img
            :src="`https://api.iconify.design/${icon}.svg?color=%23ffffff`"
            class="h-6 w-6"
            :alt="icon"
          >
          <span class="mt-0.5 max-w-full truncate">{{ icon.split(':')[1] }}</span>
        </button>
      </div>
    </div>

    <!-- INTERNAL uploads -->
    <div
      v-if="tab === 'INTERNAL'"
      class="space-y-2"
    >
      <div class="flex items-center gap-2">
        <ThemedFileTrigger
          :label="t('iconPicker.pickUpload')"
          accept="image/svg+xml,image/png,image/jpeg,image/webp,image/gif"
          :disabled="uploading"
          @change="onUploadFile"
        />
        <p
          v-if="uploading"
          class="text-xs text-[color:var(--astro-text-secondary)]"
        >
          {{ t('iconPicker.uploading') }}
        </p>
      </div>
      <p class="text-[10px] text-[color:var(--astro-text-secondary)]">
        {{ t('iconPicker.uploadHint') }}
      </p>
      <p
        v-if="uploadError"
        class="text-xs text-red-300"
      >
        {{ uploadError }}
      </p>
      <div
        v-if="uploads.length"
        class="grid max-h-56 grid-cols-6 gap-2 overflow-y-auto rounded border border-[color:var(--astro-glass-border)] p-2"
      >
        <div
          v-for="item in uploads"
          :key="item.name"
          class="group relative flex h-14 flex-col items-center justify-center rounded border"
          :class="value === item.name ? 'border-[color:var(--astro-accent)]' : 'border-transparent hover:border-[color:var(--astro-glass-border)]'"
        >
          <button
            type="button"
            class="flex h-full w-full flex-col items-center justify-center"
            :title="item.name"
            @click="value = item.name"
          >
            <img
              :src="item.url"
              class="h-6 w-6 object-contain"
              :alt="item.name"
            >
            <span class="mt-0.5 max-w-full truncate text-[10px]">{{ item.name.slice(0, 8) }}</span>
          </button>
          <button
            type="button"
            class="absolute right-0.5 top-0.5 hidden rounded bg-red-700/70 px-1 text-[10px] text-white group-hover:block"
            :title="t('iconPicker.delete')"
            @click="onDeleteUpload(item.name)"
          >
            ✕
          </button>
        </div>
      </div>
    </div>

    <!-- REMOTE preview -->
    <div
      v-if="tab === 'REMOTE' && value"
      class="flex h-14 w-14 items-center justify-center rounded border border-[color:var(--astro-glass-border)]"
    >
      <img
        :src="value"
        class="h-10 w-10 object-contain"
        :alt="t('iconPicker.preview')"
      >
    </div>
  </div>
</template>
