<script setup lang="ts">
import { ref } from 'vue';

const props = withDefaults(
  defineProps<{
    accept?: string;
    disabled?: boolean;
    label: string;
  }>(),
  { accept: '*/*', disabled: false },
);

const emit = defineEmits<{
  change: [event: Event];
}>();

const inputRef = ref<HTMLInputElement | null>(null);

function pick(): void {
  if (props.disabled) return;
  inputRef.value?.click();
}

function onKeydown(e: KeyboardEvent): void {
  if (props.disabled) return;
  if (e.key !== 'Enter' && e.key !== ' ') return;
  e.preventDefault();
  pick();
}

function onChange(e: Event): void {
  emit('change', e);
}
</script>

<template>
  <div class="inline-flex min-w-0 flex-col gap-1">
    <input
      ref="inputRef"
      type="file"
      class="pointer-events-none absolute h-0 w-0 overflow-hidden opacity-0"
      tabindex="-1"
      :accept="accept"
      :disabled="disabled"
      :aria-label="label"
      @change="onChange"
    >
    <div
      role="button"
      tabindex="0"
      class="cursor-pointer rounded-md border border-[color:var(--astro-glass-border)] bg-[color:var(--astro-glass-bg)] px-3 py-2 text-center text-xs text-[color:var(--astro-text-primary)] outline-none transition hover:border-[color:var(--astro-accent)]/50 hover:bg-white/5 focus-visible:ring-2 focus-visible:ring-[color:var(--astro-accent)] disabled:cursor-not-allowed disabled:opacity-45"
      :aria-disabled="disabled === true"
      @click="pick"
      @keydown="onKeydown"
    >
      {{ label }}
    </div>
  </div>
</template>
