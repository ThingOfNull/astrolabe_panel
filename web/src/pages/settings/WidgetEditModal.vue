<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import type { MetricNode, MetricQuery, Shape } from '@/api/types';
import type { Widget } from '@/canvas/types';
import { useDataSourceStore } from '@/stores/datasources';
import {
  ACCEPTED_SHAPES,
  DEFAULT_SEARCH_ENGINES,
  defaultAppearance,
  defaultBarConfig,
  defaultBigNumberConfig,
  defaultDividerWidgetConfig,
  defaultGaugeConfig,
  defaultLineConfig,
  defaultLinkConfig,
  defaultSearchConfig,
  defaultStatusGridConfig,
  defaultTextWidgetConfig,
  defaultWeatherWidgetConfig,
  defaultClockWidgetConfig,
  type AggregatedSearchConfig,
  type BarConfig,
  type BigNumberConfig,
  type DividerWidgetConfig,
  type GaugeConfig,
  type LineConfig,
  type SearchEngine,
  type SmartLinkConfig,
  type StatusGridConfig,
  type TextWidgetConfig,
  type WeatherWidgetConfig,
  type ClockWidgetConfig,
  type WidgetAppearance,
} from '@/widgets/types';
import { parseWidgetAppearance } from '@/widgets/widgetAppearance';

import AppearanceForm from './AppearanceForm.vue';
import CityPicker from './CityPicker.vue';
import IconPicker from './IconPicker.vue';
import MetricTreeView from './MetricTreeView.vue';
import ThresholdsEditor from './ThresholdsEditor.vue';

const props = defineProps<{
  widget: Widget | null;
  open: boolean;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'submit', payload: { id: number; patch: Partial<Widget> }): void;
}>();

type Tab = 'basic' | 'data' | 'style';
const tab = ref<Tab>('basic');

const { t } = useI18n();
const dsStore = useDataSourceStore();

const linkForm = ref<SmartLinkConfig>(defaultLinkConfig());
const appearanceForm = ref<WidgetAppearance>(defaultAppearance());
const searchForm = ref<AggregatedSearchConfig>(defaultSearchConfig());
const gaugeForm = ref<GaugeConfig>(defaultGaugeConfig());
const bignumberForm = ref<BigNumberConfig>(defaultBigNumberConfig());
const lineForm = ref<LineConfig>(defaultLineConfig());
const barForm = ref<BarConfig>(defaultBarConfig());
const gridForm = ref<StatusGridConfig>(defaultStatusGridConfig());
const textForm = ref<TextWidgetConfig>(defaultTextWidgetConfig());
const dividerForm = ref<DividerWidgetConfig>(defaultDividerWidgetConfig());
const weatherForm = ref<WeatherWidgetConfig>(defaultWeatherWidgetConfig());
const clockForm = ref<ClockWidgetConfig>(defaultClockWidgetConfig());
const iconType = ref<'ICONIFY' | 'REMOTE' | 'INTERNAL'>('ICONIFY');
const iconValue = ref<string>('mdi:link-variant');
const iconModel = computed({
  get: () => ({ icon_type: iconType.value, icon_value: iconValue.value }),
  set: (v) => {
    iconType.value = v.icon_type;
    iconValue.value = v.icon_value;
  },
});

const dataDsId = ref<number | null>(null);
const dataMetricQuery = ref<MetricQuery | null>(null);

const isLink = computed(() => props.widget?.type === 'link');
const isSearch = computed(() => props.widget?.type === 'search');
const isGauge = computed(() => props.widget?.type === 'gauge');
const isBigNumber = computed(() => props.widget?.type === 'bignumber');
const isLine = computed(() => props.widget?.type === 'line');
const isBar = computed(() => props.widget?.type === 'bar');
const isGrid = computed(() => props.widget?.type === 'grid');
const isText = computed(() => props.widget?.type === 'text');
const isDivider = computed(() => props.widget?.type === 'divider');
const isWeather = computed(() => props.widget?.type === 'weather');
const isClock = computed(() => props.widget?.type === 'clock');
const isDataWidget = computed(() =>
  isGauge.value || isBigNumber.value || isLine.value || isBar.value || isGrid.value,
);

const acceptedShape = computed<Shape | undefined>(() => {
  if (!props.widget) return undefined;
  return ACCEPTED_SHAPES[props.widget.type]?.[0];
});

onMounted(async () => {
  if (dsStore.items.length === 0) {
    await dsStore.fetchAll();
  }
});

watch(
  () => [props.widget?.id, props.open] as const,
  () => {
    if (!props.widget) return;
    appearanceForm.value = parseWidgetAppearance(props.widget.config);
    tab.value = 'basic';
    iconType.value =
      props.widget.icon_type === 'REMOTE'
        ? 'REMOTE'
        : props.widget.icon_type === 'INTERNAL'
          ? 'INTERNAL'
          : 'ICONIFY';
    iconValue.value = props.widget.icon_value || iconDefault(props.widget.type);
    if (props.widget.type === 'link') {
      const cfg = (props.widget.config ?? {}) as Partial<SmartLinkConfig>;
      const base = defaultLinkConfig();
      const merged: SmartLinkConfig = { ...base, ...cfg };
      merged.probe = { ...base.probe, ...(cfg.probe ?? {}) };
      merged.title_style = { ...base.title_style, ...(cfg.title_style ?? {}) };
      merged.url_style = { ...base.url_style, ...(cfg.url_style ?? {}) };
      linkForm.value = merged;
    }
    if (props.widget.type === 'search') {
      const cfg = (props.widget.config ?? {}) as Partial<AggregatedSearchConfig>;
      searchForm.value = {
        engines: cfg.engines && cfg.engines.length > 0 ? cfg.engines : DEFAULT_SEARCH_ENGINES,
        default_engine_id: cfg.default_engine_id ?? 'google',
      };
    }
    if (props.widget.type === 'gauge') {
      gaugeForm.value = { ...defaultGaugeConfig(), ...(props.widget.config as Partial<GaugeConfig>) };
    }
    if (props.widget.type === 'bignumber') {
      bignumberForm.value = {
        ...defaultBigNumberConfig(),
        ...(props.widget.config as Partial<BigNumberConfig>),
      };
    }
    if (props.widget.type === 'line') {
      lineForm.value = { ...defaultLineConfig(), ...(props.widget.config as Partial<LineConfig>) };
    }
    if (props.widget.type === 'bar') {
      barForm.value = { ...defaultBarConfig(), ...(props.widget.config as Partial<BarConfig>) };
    }
    if (props.widget.type === 'grid') {
      gridForm.value = {
        ...defaultStatusGridConfig(),
        ...(props.widget.config as Partial<StatusGridConfig>),
      };
    }
    if (props.widget.type === 'text') {
      textForm.value = {
        ...defaultTextWidgetConfig(),
        ...(props.widget.config as Partial<TextWidgetConfig>),
      };
    }
    if (props.widget.type === 'divider') {
      const c = (props.widget.config ?? {}) as Partial<DividerWidgetConfig>;
      const base = defaultDividerWidgetConfig();
      dividerForm.value = {
        ...base,
        ...c,
        orientation:
          c.orientation === 'vertical' || c.orientation === 'horizontal'
            ? c.orientation
            : base.orientation,
      };
    }
    if (props.widget.type === 'weather') {
      const c = (props.widget.config ?? {}) as Partial<WeatherWidgetConfig>;
      weatherForm.value = {
        ...defaultWeatherWidgetConfig(),
        ...c,
        city_id:
          typeof c.city_id === 'number' && c.city_id > 0
            ? c.city_id
            : defaultWeatherWidgetConfig().city_id,
      };
    }
    if (props.widget.type === 'clock') {
      const c = (props.widget.config ?? {}) as Partial<ClockWidgetConfig>;
      const base = defaultClockWidgetConfig();
      clockForm.value = {
        ...base,
        ...c,
        variant: c.variant === 'flip' ? 'flip' : 'digital',
        show_seconds: c.show_seconds !== false,
        show_date: c.show_date !== false,
        use_24h: c.use_24h !== false,
        timezone: typeof c.timezone === 'string' ? c.timezone : '',
      };
    }
    dataDsId.value = props.widget.data_source_id ?? null;
    dataMetricQuery.value =
      (props.widget.metric_query as MetricQuery | null) ?? null;
    if (isDataWidget.value && dataDsId.value) {
      void dsStore.discover(dataDsId.value).catch(() => undefined);
    }
  },
  { immediate: true },
);

function iconDefault(type: string): string {
  if (type === 'gauge') return 'mdi:gauge';
  if (type === 'bignumber') return 'mdi:numeric';
  if (type === 'line') return 'mdi:chart-line';
  if (type === 'bar') return 'mdi:chart-bar';
  if (type === 'grid') return 'mdi:view-grid';
  if (type === 'search') return 'mdi:magnify';
  if (type === 'text') return 'mdi:text';
  if (type === 'divider') return 'mdi:minus';
  if (type === 'weather') return 'mdi:weather-partly-cloudy';
  if (type === 'clock') return 'mdi:clock-outline';
  return 'mdi:link-variant';
}

function close(): void {
  emit('close');
}

function validateSmartLink(cfg: SmartLinkConfig): string | null {
  // display_mode (preferred) always represents at least one visible element.
  if (
    cfg.display_mode === 'icon_only' ||
    cfg.display_mode === 'title_only' ||
    cfg.display_mode === 'title_url' ||
    cfg.display_mode === 'url_only'
  ) {
    if (cfg.display_mode === 'title_only' && !String(cfg.title ?? '').trim()) {
      return t('widgetEdit.validationEmpty');
    }
    if (cfg.display_mode === 'url_only' && !String(cfg.url ?? '').trim()) {
      return t('widgetEdit.validationEmpty');
    }
    return null;
  }
  // Legacy triplet path.
  const showI = cfg.show_icon !== false;
  const showT = cfg.show_title !== false;
  const showU = cfg.show_url !== false;
  if (!showI && !showT && !showU) {
    return t('widgetEdit.validationLinkVisibility');
  }
  const titleFilled = String(cfg.title ?? '').trim().length > 0;
  const urlFilled = String(cfg.url ?? '').trim().length > 0;
  const hasVisible = showI || (showT && titleFilled) || (showU && urlFilled);
  if (!hasVisible) {
    return t('widgetEdit.validationEmpty');
  }
  return null;
}

function attachAppearanceCfg<T extends Record<string, unknown>>(cfg: T): T {
  const ap = appearanceForm.value;
  const trimmed: Partial<WidgetAppearance> = {
    variant: ap.variant,
  };
  if (ap.variant === 'glass' && ap.blur_px != null && ap.blur_px > 0) {
    trimmed.blur_px = Math.min(ap.blur_px, 64);
  }
  if (ap.variant === 'solid') {
    if (typeof ap.solid_color === 'string' && ap.solid_color.trim() !== '') {
      trimmed.solid_color = ap.solid_color.trim();
    } else {
      trimmed.solid_color = 'rgba(15, 23, 42, 0.85)';
    }
  }
  const next = { ...cfg };
  Object.assign(next, { appearance: trimmed });
  return next as T;
}

function submit(): void {
  if (!props.widget) return;
  if (isLink.value) {
    const err = validateSmartLink(linkForm.value);
    if (err !== null) {
      window.alert(err);
      return;
    }
  }
  const patch: Partial<Widget> = {
    icon_type: iconType.value,
    icon_value: iconValue.value,
  };
  if (isLink.value) {
    patch.config = attachAppearanceCfg(
      JSON.parse(JSON.stringify(linkForm.value)) as Record<string, unknown>,
    );
  } else if (isSearch.value) {
    patch.config = attachAppearanceCfg(
      JSON.parse(JSON.stringify(searchForm.value)) as Record<string, unknown>,
    );
  } else if (isGauge.value) {
    patch.config = attachAppearanceCfg(
      JSON.parse(JSON.stringify(gaugeForm.value)) as Record<string, unknown>,
    );
  } else if (isBigNumber.value) {
    patch.config = attachAppearanceCfg(
      JSON.parse(JSON.stringify(bignumberForm.value)) as Record<string, unknown>,
    );
  } else if (isLine.value) {
    patch.config = attachAppearanceCfg(
      JSON.parse(JSON.stringify(lineForm.value)) as Record<string, unknown>,
    );
  } else if (isBar.value) {
    patch.config = attachAppearanceCfg(
      JSON.parse(JSON.stringify(barForm.value)) as Record<string, unknown>,
    );
  } else if (isGrid.value) {
    patch.config = attachAppearanceCfg(
      JSON.parse(JSON.stringify(gridForm.value)) as Record<string, unknown>,
    );
  } else if (isText.value) {
    patch.config = attachAppearanceCfg(
      JSON.parse(JSON.stringify(textForm.value)) as Record<string, unknown>,
    );
  } else if (isDivider.value) {
    patch.config = attachAppearanceCfg(
      JSON.parse(JSON.stringify(dividerForm.value)) as Record<string, unknown>,
    );
  } else if (isWeather.value) {
    patch.config = attachAppearanceCfg(
      JSON.parse(JSON.stringify(weatherForm.value)) as Record<string, unknown>,
    );
  } else if (isClock.value) {
    const tz = clockForm.value.timezone?.trim() ?? '';
    patch.config = attachAppearanceCfg({
      variant: clockForm.value.variant ?? 'digital',
      show_seconds: clockForm.value.show_seconds !== false,
      show_date: clockForm.value.show_date !== false,
      use_24h: clockForm.value.use_24h !== false,
      timezone: tz,
    });
  }
  if (isDataWidget.value) {
    patch.data_source_id = dataDsId.value;
    patch.metric_query = dataMetricQuery.value;
  }
  emit('submit', { id: props.widget.id, patch });
}

async function onDataSourceChange(id: number): Promise<void> {
  dataDsId.value = id;
  if (id > 0) {
    try {
      await dsStore.discover(id);
    } catch {
      // Errors surface via store.error; keep modal usable
    }
  }
}

function onMetricSelect(payload: { node: MetricNode; shape: Shape }): void {
  dataMetricQuery.value = {
    path: payload.node.path,
    shape: payload.shape,
    window_sec: payload.shape === 'TimeSeries' ? 1800 : undefined,
  };
}

function addEngine(): void {
  searchForm.value.engines.push({
    id: `engine-${Date.now()}`,
    label: t('widgetEdit.engineLabelNew'),
    url: 'https://example.com/?q={q}',
    icon: { type: 'ICONIFY', value: 'mdi:magnify' },
  });
}

function removeEngine(id: string): void {
  searchForm.value.engines = searchForm.value.engines.filter((e: SearchEngine) => e.id !== id);
}
</script>

<template>
  <div
    v-if="open && widget"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    @click.self="close"
  >
    <div class="astro-glass max-h-[80vh] w-[640px] overflow-hidden flex flex-col">
      <header class="flex items-center justify-between border-b border-[color:var(--astro-glass-border)] px-5 py-3">
        <h3 class="text-base font-semibold">
          {{ t('widgetEdit.title', { id: widget.id, type: widget.type }) }}
        </h3>
        <button
          type="button"
          class="text-sm text-[color:var(--astro-text-secondary)] hover:text-[color:var(--astro-text-primary)]"
          @click="close"
        >
          {{ t('widgetEdit.close') }}
        </button>
      </header>

      <nav class="flex gap-1 border-b border-[color:var(--astro-glass-border)] px-5 pt-3">
        <button
          type="button"
          class="rounded-t-md px-3 py-1 text-sm"
          :class="tab === 'basic' ? 'bg-white/10' : 'hover:bg-white/5'"
          @click="tab = 'basic'"
        >
          {{ t('widgetEdit.tabBasic') }}
        </button>
        <button
          v-if="isDataWidget"
          type="button"
          class="rounded-t-md px-3 py-1 text-sm"
          :class="tab === 'data' ? 'bg-white/10' : 'hover:bg-white/5'"
          @click="tab = 'data'"
        >
          {{ t('widgetEdit.tabData') }}
        </button>
        <button
          type="button"
          class="rounded-t-md px-3 py-1 text-sm"
          :class="tab === 'style' ? 'bg-white/10' : 'hover:bg-white/5'"
          @click="tab = 'style'"
        >
          {{ t('widgetEdit.tabStyle') }}
        </button>
      </nav>

      <section class="flex-1 overflow-y-auto px-5 py-4">
        <div
          v-if="tab === 'basic' && isLink"
          class="space-y-4 text-sm"
        >
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.titleField') }}</span>
            <input
              v-model="linkForm.title"
              type="text"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
            >
          </label>
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.urlField') }}</span>
            <input
              v-model="linkForm.url"
              type="url"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
            >
          </label>
          <label class="flex items-center gap-2">
            <input
              v-model="linkForm.open_in_new_tab"
              type="checkbox"
            >
            <span>{{ t('widgetEdit.openNewTab') }}</span>
          </label>
          <fieldset class="rounded-md border border-[color:var(--astro-glass-border)] p-3">
            <legend class="px-1 text-xs text-[color:var(--astro-text-secondary)]">
              {{ t('widgetEdit.probe') }}
            </legend>
            <label class="flex items-center gap-2 text-sm">
              <input
                v-model="linkForm.probe!.enabled"
                type="checkbox"
              >
              <span>{{ t('widgetEdit.probeEnable') }}</span>
            </label>
            <div class="mt-2 grid grid-cols-2 gap-3">
              <label>
                <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.probeType') }}</span>
                <select
                  v-model="linkForm.probe!.type"
                  class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1"
                >
                  <option value="http">HTTP</option>
                  <option value="tcp">TCP</option>
                </select>
              </label>
              <label v-if="linkForm.probe!.type === 'tcp'">
                <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">host:port</span>
                <input
                  v-model="linkForm.probe!.host"
                  type="text"
                  placeholder="nas.local:5000"
                  class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1"
                >
              </label>
              <label>
                <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.probeInterval') }}</span>
                <input
                  v-model.number="linkForm.probe!.interval_sec"
                  type="number"
                  min="5"
                  class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1"
                >
              </label>
              <label>
                <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.probeTimeout') }}</span>
                <input
                  v-model.number="linkForm.probe!.timeout_sec"
                  type="number"
                  min="1"
                  class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1"
                >
              </label>
            </div>
          </fieldset>
        </div>

        <div
          v-if="tab === 'basic' && isSearch"
          class="space-y-3 text-sm"
        >
          <p class="text-xs text-[color:var(--astro-text-secondary)]">
            <i18n-t
              keypath="widgetEdit.engineIntro"
              tag="span"
            >
              <template #placeholder>
                <code>{q}</code>
              </template>
            </i18n-t>
          </p>
          <div class="space-y-2">
            <div
              v-for="engine in searchForm.engines"
              :key="engine.id"
              class="grid grid-cols-12 items-center gap-2 rounded border border-[color:var(--astro-glass-border)] p-2"
            >
              <input
                v-model="engine.label"
                type="text"
                class="col-span-3 rounded border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1 text-xs"
              >
              <input
                v-model="engine.url"
                type="url"
                class="col-span-7 rounded border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1 text-xs"
              >
              <button
                type="button"
                class="col-span-2 rounded bg-red-600/70 px-2 py-1 text-xs text-white hover:bg-red-600"
                @click="removeEngine(engine.id)"
              >
                {{ t('widgetEdit.engineDelete') }}
              </button>
            </div>
          </div>
          <button
            type="button"
            class="rounded border border-[color:var(--astro-glass-border)] px-3 py-1 text-xs hover:bg-white/5"
            @click="addEngine"
          >
            {{ t('widgetEdit.engineAdd') }}
          </button>
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.engineDefault') }}</span>
            <input
              v-model="searchForm.default_engine_id"
              type="text"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
            >
          </label>
        </div>

        <div
          v-if="tab === 'basic' && isGauge"
          class="space-y-3 text-sm"
        >
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.titleField') }}</span>
            <input
              v-model="gaugeForm.title"
              type="text"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
            >
          </label>
          <div class="grid grid-cols-3 gap-2">
            <label>
              <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.gaugeMin') }}</span>
              <input
                v-model.number="gaugeForm.min"
                type="number"
                class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1"
              >
            </label>
            <label>
              <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.gaugeMax') }}</span>
              <input
                v-model.number="gaugeForm.max"
                type="number"
                class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1"
              >
            </label>
            <label>
              <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.unit') }}</span>
              <input
                v-model="gaugeForm.unit"
                type="text"
                class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1"
              >
            </label>
          </div>
          <ThresholdsEditor v-model="gaugeForm.thresholds" />
        </div>

        <div
          v-if="tab === 'basic' && isBigNumber"
          class="space-y-3 text-sm"
        >
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.titleField') }}</span>
            <input
              v-model="bignumberForm.title"
              type="text"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
            >
          </label>
          <div class="grid grid-cols-2 gap-2">
            <label>
              <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.unit') }}</span>
              <input
                v-model="bignumberForm.unit"
                type="text"
                class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1"
              >
            </label>
            <label>
              <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.decimals') }}</span>
              <input
                v-model.number="bignumberForm.precision"
                type="number"
                min="0"
                max="6"
                class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1"
              >
            </label>
          </div>
          <ThresholdsEditor v-model="bignumberForm.thresholds" />
        </div>

        <div
          v-if="tab === 'basic' && isLine"
          class="space-y-3 text-sm"
        >
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.titleField') }}</span>
            <input
              v-model="lineForm.title"
              type="text"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
            >
          </label>
          <label class="flex items-center gap-2">
            <input
              v-model="lineForm.smooth"
              type="checkbox"
            >
            <span class="text-xs">{{ t('widgetEdit.smooth') }}</span>
          </label>
          <label class="flex items-center gap-2">
            <input
              v-model="lineForm.area_fill"
              type="checkbox"
            >
            <span class="text-xs">{{ t('widgetEdit.areaFill') }}</span>
          </label>
          <ThresholdsEditor v-model="lineForm.thresholds" />
        </div>

        <div
          v-if="tab === 'basic' && isBar"
          class="space-y-3 text-sm"
        >
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.titleField') }}</span>
            <input
              v-model="barForm.title"
              type="text"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
            >
          </label>
          <label class="flex items-center gap-2">
            <input
              v-model="barForm.horizontal"
              type="checkbox"
            >
            <span class="text-xs">{{ t('widgetEdit.horizontal') }}</span>
          </label>
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.topN') }}</span>
            <input
              v-model.number="barForm.top_n"
              type="number"
              min="1"
              max="50"
              class="w-32 rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1"
            >
          </label>
          <ThresholdsEditor v-model="barForm.thresholds" />
        </div>

        <div
          v-if="tab === 'basic' && isGrid"
          class="space-y-3 text-sm"
        >
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.titleField') }}</span>
            <input
              v-model="gridForm.title"
              type="text"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
            >
          </label>
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.cellMinPx') }}</span>
            <input
              v-model.number="gridForm.cell_min_px"
              type="number"
              min="40"
              max="200"
              class="w-32 rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1"
            >
          </label>
        </div>

        <div
          v-if="tab === 'basic' && isWeather"
          class="space-y-3 text-sm"
        >
          <div>
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">
              {{ t('widgetEdit.weatherCity') }}
            </span>
            <CityPicker
              :model-value="weatherForm.city_id > 0 ? weatherForm.city_id : null"
              @update:model-value="(id) => {
                weatherForm.city_id = id ?? defaultWeatherWidgetConfig().city_id;
              }"
              @pick-label="(label) => { weatherForm.city_label = label; }"
            />
            <p class="mt-2 text-[11px] text-[color:var(--astro-text-secondary)]">
              {{ t('widgetEdit.weatherFooter') }}
            </p>
          </div>
        </div>

        <div
          v-if="tab === 'basic' && isClock"
          class="space-y-3 text-sm"
        >
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.clockVariant') }}</span>
            <select
              v-model="clockForm.variant"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
            >
              <option value="digital">
                {{ t('widgetEdit.clockDigital') }}
              </option>
              <option value="flip">
                {{ t('widgetEdit.clockFlip') }}
              </option>
            </select>
          </label>
          <label class="flex items-center gap-2">
            <input
              v-model="clockForm.show_seconds"
              type="checkbox"
            >
            <span>{{ t('widgetEdit.clockShowSec') }}</span>
          </label>
          <label class="flex items-center gap-2">
            <input
              v-model="clockForm.show_date"
              type="checkbox"
            >
            <span>{{ t('widgetEdit.clockShowDate') }}</span>
          </label>
          <label class="flex items-center gap-2">
            <input
              v-model="clockForm.use_24h"
              type="checkbox"
            >
            <span>{{ t('widgetEdit.clock24h') }}</span>
          </label>
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">
              {{ t('widgetEdit.clockTimezone') }}
            </span>
            <input
              v-model.trim="clockForm.timezone"
              type="text"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
              :placeholder="t('widgetEdit.clockTimezonePlaceholder')"
              autocomplete="off"
            >
          </label>
        </div>

        <div
          v-if="tab === 'basic' && isText"
          class="space-y-3 text-sm"
        >
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.textContent') }}</span>
            <textarea
              v-model="textForm.content"
              rows="6"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2 font-mono text-xs"
            />
          </label>
          <div class="grid grid-cols-2 gap-2">
            <label>
              <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.textFontSize') }}</span>
              <input
                v-model="textForm.font_size"
                type="text"
                class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1"
              >
            </label>
            <label>
              <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.textFontWeight') }}</span>
              <input
                v-model="textForm.font_weight"
                type="text"
                class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1"
              >
            </label>
          </div>
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.textColor') }}</span>
            <div class="flex gap-2">
              <input
                v-model="textForm.color"
                type="color"
                class="h-9 w-9 shrink-0 cursor-pointer rounded border border-[color:var(--astro-glass-border)] bg-transparent p-0"
              >
              <input
                v-model="textForm.color"
                type="text"
                class="flex-1 rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
              >
            </div>
          </label>
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.textAlign') }}</span>
            <select
              v-model="textForm.text_align"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
            >
              <option value="left">
                {{ t('widgetEdit.alignLeft') }}
              </option>
              <option value="center">
                {{ t('widgetEdit.alignCenter') }}
              </option>
              <option value="right">
                {{ t('widgetEdit.alignRight') }}
              </option>
            </select>
          </label>
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.textOrientation') }}</span>
            <select
              v-model="textForm.orientation"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
            >
              <option value="horizontal">
                {{ t('widgetEdit.orientHoriz') }}
              </option>
              <option value="vertical">
                {{ t('widgetEdit.orientVert') }}
              </option>
            </select>
          </label>
        </div>

        <div
          v-if="tab === 'basic' && isDivider"
          class="space-y-3 text-sm"
        >
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.dividerOrientation') }}</span>
            <select
              v-model="dividerForm.orientation"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
            >
              <option value="horizontal">
                {{ t('widgetEdit.dividerH') }}
              </option>
              <option value="vertical">
                {{ t('widgetEdit.dividerV') }}
              </option>
            </select>
          </label>
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.dividerThickness') }}</span>
            <input
              v-model.number="dividerForm.thickness"
              type="number"
              min="1"
              max="16"
              class="w-32 rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1"
            >
          </label>
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.dividerStyle') }}</span>
            <select
              v-model="dividerForm.line_style"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
            >
              <option value="solid">
                {{ t('widgetEdit.lineSolid') }}
              </option>
              <option value="dashed">
                {{ t('widgetEdit.lineDashed') }}
              </option>
              <option value="dotted">
                {{ t('widgetEdit.lineDotted') }}
              </option>
            </select>
          </label>
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.dividerColor') }}</span>
            <div class="flex gap-2">
              <input
                v-model="dividerForm.color"
                type="color"
                class="h-9 w-9 shrink-0 cursor-pointer rounded border border-[color:var(--astro-glass-border)] bg-transparent p-0"
              >
              <input
                v-model="dividerForm.color"
                type="text"
                class="flex-1 rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
              >
            </div>
          </label>
        </div>

        <div
          v-if="tab === 'data' && isDataWidget"
          class="space-y-3 text-sm"
        >
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.dataSource') }}</span>
            <select
              :value="dataDsId ?? ''"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
              @change="onDataSourceChange(Number(($event.target as HTMLSelectElement).value))"
            >
              <option value="">{{ t('widgetEdit.dataSourcePlaceholder') }}</option>
              <option
                v-for="d in dsStore.items"
                :key="d.id"
                :value="d.id"
              >
                {{ d.name }} ({{ d.type }})
              </option>
            </select>
          </label>
          <p
            v-if="dsStore.items.length === 0"
            class="text-xs text-amber-300"
          >
            {{ t('widgetEdit.dataNoSource') }}
          </p>
          <p class="text-xs text-[color:var(--astro-text-secondary)]">
            {{ t('widgetEdit.dataShapeNote') }}<span class="font-mono">{{ acceptedShape }}</span>
          </p>
          <div
            v-if="dataDsId && dsStore.trees[dataDsId]"
            class="max-h-64 overflow-y-auto rounded border border-[color:var(--astro-glass-border)] p-2"
          >
            <MetricTreeView
              :nodes="dsStore.trees[dataDsId].roots"
              :filter-shape="acceptedShape"
              :selected-path="dataMetricQuery?.path"
              @select="onMetricSelect"
            />
          </div>
          <div
            v-if="dataMetricQuery"
            class="rounded bg-white/5 px-3 py-2 text-xs"
          >
            {{ t('widgetEdit.dataSelected') }}<span class="font-mono">{{ dataMetricQuery.path }}</span>
            <span class="ml-2 text-[color:var(--astro-text-secondary)]">{{ dataMetricQuery.shape }}</span>
          </div>
        </div>

        <div
          v-if="tab === 'style'"
          class="space-y-3 text-sm"
        >
          <AppearanceForm v-model="appearanceForm" />
          <IconPicker v-model="iconModel" />
          <template v-if="isLink">
            <label class="block">
              <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.linkLayout') }}</span>
              <select
                v-model="linkForm.layout"
                class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
              >
                <option value="horizontal">
                  {{ t('widgetEdit.linkLayoutHoriz') }}
                </option>
                <option value="vertical">
                  {{ t('widgetEdit.linkLayoutVert') }}
                </option>
              </select>
            </label>
            <fieldset class="rounded-md border border-[color:var(--astro-glass-border)] p-3">
              <legend class="px-1 text-xs text-[color:var(--astro-text-secondary)]">
                {{ t('widgetEdit.linkVisible') }}
              </legend>
              <label class="block">
                <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.linkDisplayMode') }}</span>
                <select
                  v-model="linkForm.display_mode"
                  class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1.5"
                >
                  <option value="icon_only">{{ t('widgetEdit.linkDisplayMode_icon_only') }}</option>
                  <option value="title_only">{{ t('widgetEdit.linkDisplayMode_title_only') }}</option>
                  <option value="title_url">{{ t('widgetEdit.linkDisplayMode_title_url') }}</option>
                  <option value="url_only">{{ t('widgetEdit.linkDisplayMode_url_only') }}</option>
                </select>
              </label>
              <p class="mt-2 text-[11px] text-[color:var(--astro-text-secondary)]">
                {{ t('widgetEdit.linkVisibleHint') }}
              </p>
            </fieldset>
            <p class="text-xs text-[color:var(--astro-text-secondary)]">
              {{ t('widgetEdit.linkStyleHint') }}
            </p>
            <fieldset class="rounded-md border border-[color:var(--astro-glass-border)] p-3">
              <legend class="px-1 text-xs text-[color:var(--astro-text-secondary)]">
                {{ t('widgetEdit.linkTitleLegend') }}
              </legend>
              <div class="grid grid-cols-3 gap-2">
                <label class="col-span-1">
                  <span class="mb-1 block text-[11px] text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.fontSize') }}</span>
                  <input
                    v-model="linkForm.title_style.font_size"
                    type="text"
                    :placeholder="t('widgetEdit.fontSizePlaceholder')"
                    class="w-full rounded border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1 text-xs"
                  >
                </label>
                <label class="col-span-1">
                  <span class="mb-1 block text-[11px] text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.fontWeight') }}</span>
                  <input
                    v-model="linkForm.title_style.font_weight"
                    type="text"
                    :placeholder="t('widgetEdit.fontWeightPlaceholder')"
                    class="w-full rounded border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1 text-xs"
                  >
                </label>
                <label class="col-span-1">
                  <span class="mb-1 block text-[11px] text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.color') }}</span>
                  <div class="flex gap-1">
                    <input
                      v-model="linkForm.title_style.color"
                      type="color"
                      class="h-7 w-7 shrink-0 cursor-pointer rounded border border-[color:var(--astro-glass-border)] bg-transparent p-0"
                    >
                    <input
                      v-model="linkForm.title_style.color"
                      type="text"
                      :placeholder="t('widgetEdit.colorPlaceholderTitle')"
                      class="flex-1 rounded border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1 text-xs"
                    >
                  </div>
                </label>
              </div>
            </fieldset>
            <fieldset class="rounded-md border border-[color:var(--astro-glass-border)] p-3">
              <legend class="px-1 text-xs text-[color:var(--astro-text-secondary)]">
                {{ t('widgetEdit.linkUrlLegend') }}
              </legend>
              <div class="grid grid-cols-3 gap-2">
                <label class="col-span-1">
                  <span class="mb-1 block text-[11px] text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.fontSize') }}</span>
                  <input
                    v-model="linkForm.url_style.font_size"
                    type="text"
                    :placeholder="t('widgetEdit.fontSizePlaceholder')"
                    class="w-full rounded border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1 text-xs"
                  >
                </label>
                <label class="col-span-1">
                  <span class="mb-1 block text-[11px] text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.fontWeight') }}</span>
                  <input
                    v-model="linkForm.url_style.font_weight"
                    type="text"
                    :placeholder="t('widgetEdit.fontWeightPlaceholder')"
                    class="w-full rounded border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1 text-xs"
                  >
                </label>
                <label class="col-span-1">
                  <span class="mb-1 block text-[11px] text-[color:var(--astro-text-secondary)]">{{ t('widgetEdit.color') }}</span>
                  <div class="flex gap-1">
                    <input
                      v-model="linkForm.url_style.color"
                      type="color"
                      class="h-7 w-7 shrink-0 cursor-pointer rounded border border-[color:var(--astro-glass-border)] bg-transparent p-0"
                    >
                    <input
                      v-model="linkForm.url_style.color"
                      type="text"
                      :placeholder="t('widgetEdit.colorPlaceholderUrl')"
                      class="flex-1 rounded border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1 text-xs"
                    >
                  </div>
                </label>
              </div>
            </fieldset>
          </template>
        </div>
      </section>

      <footer class="flex justify-end gap-2 border-t border-[color:var(--astro-glass-border)] px-5 py-3">
        <button
          type="button"
          class="astro-btn-icon rounded-md border border-[color:var(--astro-glass-border)] px-4 py-1.5 text-sm hover:bg-white/5 hover:shadow-md"
          @click="close"
        >
          {{ t('widgetEdit.cancel') }}
        </button>
        <button
          type="button"
          class="astro-btn-icon rounded-md bg-[color:var(--astro-accent)] px-4 py-1.5 text-sm text-black hover:brightness-110 active:brightness-95"
          @click="submit"
        >
          {{ t('widgetEdit.save') }}
        </button>
      </footer>
    </div>
  </div>
</template>
