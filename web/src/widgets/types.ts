// Typed slices of widget config JSON; unknown keys stay when round-tripping.

import type { Shape } from '@/api/types';

/** Optional chrome style; stored as JSON on server. */
export interface WidgetAppearance {
  variant: 'glass' | 'solid';
  blur_px?: number;
  solid_color?: string;
}

export interface TextStyle {
  font_size?: string;
  color?: string;
  font_weight?: string;
}

export interface SmartLinkConfig {
  title: string;
  url: string;
  /** Default: icon left, text right. */
  layout?: 'horizontal' | 'vertical';
  /**
   * Preferred way to express which parts to render. Replaces the legacy
   * triplet of show_icon/show_title/show_url booleans, which permitted the
   * "nothing visible" combination. The legacy fields remain readable for
   * backward compatibility (see resolveDisplayMode in SmartLink.vue).
   */
  display_mode?: 'icon_only' | 'title_only' | 'title_url' | 'url_only';
  show_icon?: boolean;
  show_title?: boolean;
  show_url?: boolean;
  title_style: TextStyle;
  url_style: TextStyle;
  open_in_new_tab?: boolean;
  probe?: {
    enabled?: boolean;
    type?: 'http' | 'tcp';
    host?: string;
    interval_sec?: number;
    timeout_sec?: number;
  };
}

export interface TextWidgetConfig {
  content: string;
  font_size?: string;
  font_weight?: string;
  color?: string;
  text_align?: 'left' | 'center' | 'right';
  /** Rows vs column text (vertical uses writing-mode). */
  orientation?: 'horizontal' | 'vertical';
}

export interface DividerWidgetConfig {
  orientation: 'horizontal' | 'vertical';
  thickness?: number;
  color?: string;
  line_style?: 'solid' | 'dashed' | 'dotted';
}

/** Weather widget city id; see meizuWeatherCities.json */
export interface WeatherWidgetConfig {
  city_id: number;
  /** UI label cache; API keys off city_id only. */
  city_label?: string;
}

/** Clock widget: no datasource; browser or IANA tz. */
export interface ClockWidgetConfig {
  /** digital | flip (split-flap style) */
  variant?: 'digital' | 'flip';
  show_seconds?: boolean;
  show_date?: boolean;
  use_24h?: boolean;
  /** IANA zone; empty = browser local */
  timezone?: string;
}

export interface SearchEngine {
  id: string;
  label: string;
  url: string;
  icon?: { type: 'INTERNAL' | 'REMOTE' | 'ICONIFY'; value: string };
}

export interface AggregatedSearchConfig {
  engines: SearchEngine[];
  default_engine_id?: string;
}

export const DEFAULT_SEARCH_ENGINES: SearchEngine[] = [
  {
    id: 'google',
    label: 'Google',
    url: 'https://www.google.com/search?q={q}',
    icon: { type: 'ICONIFY', value: 'logos:google-icon' },
  },
  {
    id: 'bing',
    label: 'Bing',
    url: 'https://www.bing.com/search?q={q}',
    icon: { type: 'ICONIFY', value: 'logos:bing' },
  },
  {
    id: 'duckduckgo',
    label: 'DuckDuckGo',
    url: 'https://duckduckgo.com/?q={q}',
    icon: { type: 'ICONIFY', value: 'logos:duckduckgo' },
  },
  {
    id: 'baidu',
    label: 'Baidu',
    url: 'https://www.baidu.com/s?wd={q}',
    icon: { type: 'ICONIFY', value: 'simple-icons:baidu' },
  },
];

export interface GaugeConfig {
  title?: string;
  min?: number;
  max?: number;
  unit?: string;
  thresholds?: { value: number; color: string }[];
}

export interface BigNumberConfig {
  title?: string;
  unit?: string;
  precision?: number;
  thresholds?: { value: number; color: string }[];
}

export interface LineConfig {
  title?: string;
  smooth?: boolean;
  area_fill?: boolean;
  thresholds?: { value: number; color: string }[];
}

export interface BarConfig {
  title?: string;
  horizontal?: boolean;
  top_n?: number;
  thresholds?: { value: number; color: string }[];
}

export interface StatusGridConfig {
  title?: string;
  cell_min_px?: number;
}

// widget.type allowed metric shapes (server has same map).
export const ACCEPTED_SHAPES: Record<string, Shape[]> = {
  gauge: ['Scalar'],
  bignumber: ['Scalar'],
  line: ['TimeSeries'],
  bar: ['Categorical'],
  grid: ['EntityList'],
  liquid: ['Scalar'],
  radial3d: ['Scalar'],
  heatmap: ['TimeSeries'],
  sparkline: ['TimeSeries'],
  bullet: ['Scalar'],
  progress_ring: ['Scalar'],
  timeline: ['EntityList'],
};

export function defaultAppearance(): WidgetAppearance {
  return { variant: 'glass', blur_px: 16 };
}

export function defaultLinkConfig(): SmartLinkConfig {
  return {
    // Empty title: rendering layer falls back to t('widget.default_title.link')
    // so the stored config stays locale-neutral.
    title: '',
    url: 'https://example.com',
    layout: 'horizontal',
    display_mode: 'title_url',
    // Legacy show_* still defaulted on so older code paths and any tooling
    // that inspects the config keeps seeing the same shape.
    show_icon: true,
    show_title: true,
    show_url: true,
    title_style: {},
    url_style: {},
    open_in_new_tab: true,
    probe: { enabled: true, type: 'http', interval_sec: 30, timeout_sec: 4 },
  };
}

export function defaultSearchConfig(): AggregatedSearchConfig {
  return {
    engines: DEFAULT_SEARCH_ENGINES,
    default_engine_id: 'google',
  };
}

export function defaultGaugeConfig(): GaugeConfig {
  return { title: '', min: 0, max: 100, unit: '%' };
}

export function defaultBigNumberConfig(): BigNumberConfig {
  return { title: '', precision: 1 };
}

export function defaultLineConfig(): LineConfig {
  return { title: '', smooth: true, area_fill: true };
}

export function defaultBarConfig(): BarConfig {
  return { title: '', horizontal: true, top_n: 10 };
}

export function defaultStatusGridConfig(): StatusGridConfig {
  return { title: '', cell_min_px: 80 };
}

export function defaultTextWidgetConfig(): TextWidgetConfig {
  return {
    content: '',
    font_size: '14px',
    font_weight: '400',
    color: '',
    text_align: 'left',
    orientation: 'horizontal',
  };
}

export function defaultDividerWidgetConfig(): DividerWidgetConfig {
  return {
    orientation: 'horizontal',
    thickness: 2,
    color: '',
    line_style: 'solid',
  };
}

export function defaultWeatherWidgetConfig(): WeatherWidgetConfig {
  return {
    city_id: 101010100,
    // city_label is a UI display cache only; leaving it empty lets locale
    // lookup resolve to the localized name on first render.
    city_label: '',
  };
}

export function defaultClockWidgetConfig(): ClockWidgetConfig {
  return {
    variant: 'digital',
    show_seconds: true,
    show_date: true,
    use_24h: true,
    timezone: '',
  };
}
