<script setup lang="ts">
/**
 * ShortcutHelp: a `?` icon button that reveals a compact shortcut legend.
 *
 * Keeps the previous long-form shortcut string out of the toolbar status bar
 * so critical info (selection count, errors) has room to breathe.
 */
import { Icon } from '@iconify/vue';
import { onBeforeUnmount, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const open = ref(false);

function toggle(): void {
  open.value = !open.value;
}

function onClickOutside(e: MouseEvent): void {
  if (!open.value) return;
  const el = rootRef.value;
  if (!el) return;
  if (!el.contains(e.target as Node)) open.value = false;
}

function onKey(e: KeyboardEvent): void {
  if (e.key === 'Escape') open.value = false;
}

const rootRef = ref<HTMLElement | null>(null);

onMounted(() => {
  document.addEventListener('mousedown', onClickOutside);
  document.addEventListener('keydown', onKey);
});
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onClickOutside);
  document.removeEventListener('keydown', onKey);
});
</script>

<template>
  <div
    ref="rootRef"
    class="relative inline-flex"
  >
    <button
      type="button"
      class="astro-btn-icon border border-[color:var(--astro-glass-border)] p-1.5 rounded-[var(--radius-sm)]"
      :title="t('settings.shortcuts')"
      @click="toggle"
    >
      <Icon
        icon="mdi:help-circle-outline"
        width="16"
        height="16"
      />
    </button>
    <div
      v-if="open"
      class="astro-glass astro-glass--surface-2 absolute right-0 top-full z-40 mt-2 w-[360px] p-3 text-[length:var(--fs-xs)] leading-relaxed"
      :style="{ boxShadow: 'var(--elev-2)' }"
      @click.stop
    >
      {{ t('settings.shortcuts') }}
    </div>
  </div>
</template>
