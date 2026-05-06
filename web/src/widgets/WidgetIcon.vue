<script setup lang="ts">
import { Icon } from '@iconify/vue';
import { computed } from 'vue';

const props = defineProps<{
  type: '' | 'INTERNAL' | 'REMOTE' | 'ICONIFY';
  value: string;
  size?: string;
  fallback?: string;
}>();

const isIconify = computed(() => props.type === 'ICONIFY' && props.value);
const isImage = computed(
  () => (props.type === 'REMOTE' || props.type === 'INTERNAL') && props.value,
);

const dim = computed(() => props.size ?? '32px');

const remoteSrc = computed(() => {
  if (props.type === 'INTERNAL') {
    return `/uploads/${props.value}`;
  }
  return props.value;
});
</script>

<template>
  <span
    class="inline-flex items-center justify-center"
    :style="{ width: dim, height: dim }"
  >
    <Icon
      v-if="isIconify"
      :icon="value"
      :width="dim"
      :height="dim"
    />
    <img
      v-else-if="isImage"
      :src="remoteSrc"
      :alt="value"
      :style="{ width: dim, height: dim }"
      loading="lazy"
    >
    <Icon
      v-else
      :icon="fallback ?? 'mdi:application-outline'"
      :width="dim"
      :height="dim"
    />
  </span>
</template>
