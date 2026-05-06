<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import type { WidgetAppearance } from '@/widgets/types';

const { t } = useI18n();

const props = defineProps<{
  modelValue: WidgetAppearance;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: WidgetAppearance): void;
}>();

function emitFull(next: WidgetAppearance): void {
  emit('update:modelValue', next);
}

function setVariant(variant: WidgetAppearance['variant']): void {
  const base: WidgetAppearance = { variant };
  if (variant === 'glass') {
    base.blur_px =
      props.modelValue.blur_px != null && props.modelValue.blur_px > 0
        ? Math.min(props.modelValue.blur_px, 64)
        : 16;
  }
  if (variant === 'solid') {
    base.solid_color =
      props.modelValue.solid_color && props.modelValue.solid_color.trim() !== ''
        ? props.modelValue.solid_color
        : 'rgba(15, 23, 42, 0.85)';
  }
  emitFull(base);
}

const blurPx = computed({
  get: () => props.modelValue.blur_px ?? 16,
  set: (n: number) => {
    emitFull({
      ...props.modelValue,
      variant: 'glass',
      blur_px: Number.isFinite(n) ? Math.min(Math.max(n, 1), 64) : 16,
    });
  },
});

const solidColor = computed({
  get: () => props.modelValue.solid_color ?? '',
  set: (s: string) => {
    emitFull({
      ...props.modelValue,
      variant: 'solid',
      solid_color: s,
    });
  },
});
</script>

<template>
  <fieldset class="rounded-md border border-[color:var(--astro-glass-border)] p-3">
    <legend class="px-1 text-xs text-[color:var(--astro-text-secondary)]">
      {{ t('appearance.legend') }}
    </legend>
    <label class="mb-3 block">
      <span class="mb-1 block text-[11px] text-[color:var(--astro-text-secondary)]">{{
        t('appearance.skin')
      }}</span>
      <select
        class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2 text-sm"
        :value="modelValue.variant"
        @change="
          setVariant(($event.target as HTMLSelectElement).value as WidgetAppearance['variant'])
        "
      >
        <option value="glass">
          {{ t('appearance.glass') }}
        </option>
        <option value="solid">
          {{ t('appearance.solid') }}
        </option>
      </select>
    </label>
    <label
      v-if="modelValue.variant === 'glass'"
      class="block"
    >
      <span class="mb-1 block text-[11px] text-[color:var(--astro-text-secondary)]">
        {{ t('appearance.blurPx') }}
      </span>
      <input
        v-model.number="blurPx"
        type="number"
        min="1"
        max="64"
        class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2 text-sm"
      >
    </label>
    <label
      v-if="modelValue.variant === 'solid'"
      class="mt-2 block"
    >
      <span class="mb-1 block text-[11px] text-[color:var(--astro-text-secondary)]">
        {{ t('appearance.solidColor') }}
      </span>
      <div class="flex gap-2">
        <input
          v-model="solidColor"
          type="color"
          class="h-9 w-9 shrink-0 cursor-pointer rounded border border-[color:var(--astro-glass-border)] bg-transparent p-0"
        >
        <input
          v-model="solidColor"
          type="text"
          :placeholder="t('appearance.solidPlaceholder')"
          class="flex-1 rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2 text-sm"
        >
      </div>
    </label>
  </fieldset>
</template>
