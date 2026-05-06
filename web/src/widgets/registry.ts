import type { Component } from 'vue';

import AggregatedSearch from './AggregatedSearch.vue';
import BarChart from './BarChart.vue';
import BigNumber from './BigNumber.vue';
import DividerBlock from './DividerBlock.vue';
import Gauge from './Gauge.vue';
import LineChart from './LineChart.vue';
import SmartLink from './SmartLink.vue';
import StatusGrid from './StatusGrid.vue';
import TextBlock from './TextBlock.vue';
import ClockWidget from './ClockWidget.vue';
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
  | 'clock';

export interface PaletteEntry {
  type: PaletteType;
  label: string;
  description: string;
  defaultW: number;
  defaultH: number;
  icon: string;
}

export const palette: PaletteEntry[] = [
  {
    type: 'link',
    label: '智能链接',
    description: '内网服务入口，支持自动探活',
    defaultW: 12,
    defaultH: 8,
    icon: 'mdi:link-variant',
  },
  {
    type: 'search',
    label: '聚合搜索',
    description: '多搜索引擎切换，按 Ctrl+K 聚焦',
    defaultW: 60,
    defaultH: 6,
    icon: 'mdi:magnify',
  },
  {
    type: 'gauge',
    label: '仪表盘',
    description: '展示 Scalar 数据；适合百分比 / 进度',
    defaultW: 18,
    defaultH: 16,
    icon: 'mdi:gauge',
  },
  {
    type: 'bignumber',
    label: '大数字',
    description: '关键 KPI；等宽字体瞬时数值',
    defaultW: 16,
    defaultH: 10,
    icon: 'mdi:numeric',
  },
  {
    type: 'line',
    label: '折线图',
    description: '展示 TimeSeries 数据的历史趋势',
    defaultW: 32,
    defaultH: 16,
    icon: 'mdi:chart-line',
  },
  {
    type: 'bar',
    label: '柱状对比',
    description: '展示 Categorical 数据的横向对比',
    defaultW: 28,
    defaultH: 16,
    icon: 'mdi:chart-bar',
  },
  {
    type: 'grid',
    label: '状态矩阵',
    description: '多实体红 / 绿 / 灰状态格',
    defaultW: 32,
    defaultH: 12,
    icon: 'mdi:view-grid',
  },
  {
    type: 'text',
    label: '文本',
    description: '多行说明或标注；不参与指标数据源',
    defaultW: 20,
    defaultH: 8,
    icon: 'mdi:text',
  },
  {
    type: 'divider',
    label: '分割线',
    description: '水平或竖直分隔线',
    defaultW: 28,
    defaultH: 2,
    icon: 'mdi:minus',
  },
  {
    type: 'weather',
    label: '天气',
    description: '魅族数据源；手动选择城市',
    defaultW: 16,
    defaultH: 12,
    icon: 'mdi:weather-partly-cloudy',
  },
  {
    type: 'clock',
    label: '时钟',
    description: '数字 / 翻页风格；支持时区与日期',
    defaultW: 22,
    defaultH: 10,
    icon: 'mdi:clock-outline',
  },
];
