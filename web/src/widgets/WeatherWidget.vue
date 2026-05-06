<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import type { Widget } from '@/canvas/types';

import { defaultWeatherWidgetConfig, type WeatherWidgetConfig } from './types';

const DASH = '--';

const { t } = useI18n();

const props = defineProps<{
  widget: Widget;
}>();

const DEFAULT_W = 16;
const DEFAULT_H = 12;
const scale = computed(() => {
  const rw = props.widget.w / DEFAULT_W;
  const rh = props.widget.h / DEFAULT_H;
  return Math.max(0.85, Math.min(rw, rh));
});

const cfg = computed<WeatherWidgetConfig>(() => {
  const c = (props.widget.config ?? {}) as Partial<WeatherWidgetConfig>;
  const base = defaultWeatherWidgetConfig();
  const id = typeof c.city_id === 'number' && c.city_id > 0 ? c.city_id : base.city_id;
  return {
    ...base,
    ...c,
    city_id: id,
  };
});

const cityName = ref(DASH);
const temp = ref(DASH);
const condition = ref(DASH);
const wind = ref(DASH);
const loading = ref(false);

function setDash(): void {
  cityName.value = cfg.value.city_label?.trim() || DASH;
  temp.value = DASH;
  condition.value = DASH;
  wind.value = DASH;
}

interface MeizuWeatherPayload {
  code?: string | number;
  value?: Array<{
    city?: string;
    realtime?: {
      temp?: string;
      weather?: string;
      wD?: string;
      wS?: string;
    };
  }>;
}

let timer: number | null = null;
const POLL_MS = 20 * 60 * 1000;

async function fetchWeather(): Promise<void> {
  const id = cfg.value.city_id;
  if (!id || id <= 0) {
    setDash();
    return;
  }
  loading.value = true;
  try {
    const res = await fetch(`/api/weather?city_id=${encodeURIComponent(String(id))}`);
    if (!res.ok) throw new Error('http');
    const j = (await res.json()) as MeizuWeatherPayload;
    const ok = j.code === '200' || j.code === 200;
    const block = ok ? j.value?.[0] : undefined;
    const rt = block?.realtime;
    if (!rt || !block) throw new Error('empty');

    cityName.value = block.city?.trim() || cfg.value.city_label?.trim() || DASH;
    temp.value = rt.temp?.trim() || DASH;
    condition.value = rt.weather?.trim() || DASH;
    const wParts = [rt.wD?.trim(), rt.wS?.trim()].filter(Boolean);
    wind.value = wParts.join(' ') || DASH;
  } catch {
    setDash();
  } finally {
    loading.value = false;
  }
}

function restartTimer(): void {
  if (timer !== null) {
    clearInterval(timer);
    timer = null;
  }
  timer = window.setInterval(() => {
    void fetchWeather();
  }, POLL_MS);
}

onMounted(() => {
  setDash();
  void fetchWeather();
  restartTimer();
});

onBeforeUnmount(() => {
  if (timer !== null) clearInterval(timer);
});

watch(
  () => cfg.value.city_id,
  () => {
    setDash();
    void fetchWeather();
    restartTimer();
  },
);

const titleSize = computed(() => `${Math.round(13 * scale.value)}px`);
const tempSize = computed(() => `${Math.round(36 * scale.value)}px`);
const metaSize = computed(() => `${Math.round(11 * scale.value)}px`);
const subTempSize = computed(() => `${Math.round(18 * scale.value)}px`);
const gap = computed(() => `${Math.round(10 * scale.value)}px`);
const pad = computed(() => `${Math.round(14 * scale.value)}px`);
</script>

<template>
  <div
    class="weather-widget relative flex h-full min-h-0 min-w-0 flex-col overflow-hidden rounded-[inherit] text-[color:var(--astro-text-primary)]"
    :style="{ padding: pad, gap }"
  >
    <div
      class="pointer-events-none absolute -right-[20%] -top-[30%] h-[85%] w-[85%] rounded-full opacity-[0.14]"
      style="background: radial-gradient(circle at 30% 30%, var(--astro-accent), transparent 70%)"
    />
    <div class="relative z-[1] flex items-start justify-between gap-2">
      <div
        class="min-w-0 font-medium leading-tight text-[color:var(--astro-text-primary)]"
        :style="{ fontSize: titleSize }"
      >
        {{ loading && cityName === DASH ? t('weather.loading') : cityName }}
      </div>
      <div
        v-if="condition !== DASH"
        class="shrink-0 rounded-full border border-[color:var(--astro-glass-border)] bg-[color:var(--astro-glass-bg)] px-2 py-0.5 text-[color:var(--astro-text-secondary)]"
        :style="{ fontSize: metaSize }"
      >
        {{ condition }}
      </div>
    </div>

    <div class="relative z-[1] flex flex-1 items-end gap-2">
      <div
        class="tabular-nums leading-none tracking-tight text-[color:var(--astro-text-primary)] drop-shadow-[0_1px_12px_color-mix(in_srgb,var(--astro-accent)_35%,transparent)]"
        :style="{ fontSize: tempSize }"
      >
        {{ temp }}
      </div>
      <span
        v-if="temp !== DASH"
        class="pb-1 opacity-65"
        :style="{ fontSize: subTempSize }"
      >
        °C
      </span>
    </div>

    <div
      class="relative z-[1] flex min-h-0 items-center justify-between gap-3 border-t border-[color:var(--astro-glass-border)]/70 pt-2 text-[color:var(--astro-text-secondary)]"
      :style="{ fontSize: metaSize }"
    >
      <span class="shrink-0">{{ t('weather.wind') }}</span>
      <span class="min-w-0 truncate text-right font-medium text-[color:var(--astro-text-primary)]">
        {{ wind }}
      </span>
    </div>
  </div>
</template>
