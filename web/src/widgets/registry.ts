import type { Component } from 'vue';

import AggregatedSearch from './AggregatedSearch.vue';
import BarChart from './BarChart.vue';
import BigNumber from './BigNumber.vue';
import Bullet from './Bullet.vue';
import ClockWidget from './ClockWidget.vue';
import DividerBlock from './DividerBlock.vue';
import Gauge from './Gauge.vue';
import HeatmapCalendar from './HeatmapCalendar.vue';
import LineChart from './LineChart.vue';
import LiquidFill from './LiquidFill.vue';
import ProgressRing from './ProgressRing.vue';
import RadialGauge3D from './RadialGauge3D.vue';
import SmartLink from './SmartLink.vue';
import Sparkline from './Sparkline.vue';
import StatusGrid from './StatusGrid.vue';
import TextBlock from './TextBlock.vue';
import Timeline from './Timeline.vue';
import WeatherWidget from './WeatherWidget.vue';

// Register widget renderers at build time.
export const widgetComponents: Record<string, Component> = {
  link: SmartLink,
  search: AggregatedSearch,
  gauge: Gauge,
  bignumber: BigNumber,
  line: LineChart,
  bar: BarChart,
  grid: StatusGrid,
  text: TextBlock,
  divider: DividerBlock,
  weather: WeatherWidget,
  clock: ClockWidget,
  liquid: LiquidFill,
  sparkline: Sparkline,
  bullet: Bullet,
  progress_ring: ProgressRing,
  heatmap: HeatmapCalendar,
  timeline: Timeline,
  radial3d: RadialGauge3D,
};

export type PaletteType =
  | 'link'
  | 'search'
  | 'gauge'
  | 'bignumber'
  | 'line'
  | 'bar'
  | 'grid'
  | 'text'
  | 'divider'
  | 'weather'
  | 'clock'
  | 'liquid'
  | 'sparkline'
  | 'bullet'
  | 'progress_ring'
  | 'heatmap'
  | 'timeline'
  | 'radial3d';

/**
 * Palette entry: localizable via i18n keys instead of hardcoded text so
 * en-US users get an English palette without a fork.
 */
export interface PaletteEntry {
  type: PaletteType;
  /** vue-i18n key under widget.palette.<type>.label */
  labelKey: string;
  /** vue-i18n key under widget.palette.<type>.description */
  descriptionKey: string;
  defaultW: number;
  defaultH: number;
  icon: string;
}

function entry(type: PaletteType, w: number, h: number, icon: string): PaletteEntry {
  return {
    type,
    labelKey: `widget.palette.${type}.label`,
    descriptionKey: `widget.palette.${type}.description`,
    defaultW: w,
    defaultH: h,
    icon,
  };
}

export const palette: PaletteEntry[] = [
  entry('link', 12, 8, 'mdi:link-variant'),
  entry('search', 60, 6, 'mdi:magnify'),
  entry('gauge', 18, 16, 'mdi:gauge'),
  entry('bignumber', 16, 10, 'mdi:numeric'),
  entry('line', 32, 16, 'mdi:chart-line'),
  entry('bar', 28, 16, 'mdi:chart-bar'),
  entry('grid', 32, 12, 'mdi:view-grid'),
  entry('text', 20, 8, 'mdi:text'),
  entry('divider', 28, 2, 'mdi:minus'),
  entry('weather', 16, 12, 'mdi:weather-partly-cloudy'),
  entry('clock', 22, 10, 'mdi:clock-outline'),
  entry('liquid', 16, 16, 'mdi:water'),
  entry('sparkline', 18, 6, 'mdi:chart-bell-curve'),
  entry('bullet', 24, 6, 'mdi:bullseye-arrow'),
  entry('progress_ring', 14, 14, 'mdi:progress-clock'),
  entry('heatmap', 32, 12, 'mdi:calendar-month'),
  entry('timeline', 28, 10, 'mdi:timeline-text'),
  entry('radial3d', 18, 16, 'mdi:gauge-full'),
];
