// TS shapes mirroring internal adapter types.
export type Shape = 'Scalar' | 'TimeSeries' | 'Categorical' | 'EntityList';

export interface MetricNode {
  path: string;
  label: string;
  unit?: string;
  shapes: Shape[];
  leaf: boolean;
  children?: MetricNode[];
}

export interface MetricTree {
  roots: MetricNode[];
}

export interface MetricQuery {
  path: string;
  shape: Shape;
  window_sec?: number;
  points?: number;
  dim?: string;
}

export interface ScalarPayload {
  value: number;
  unit?: string;
  ts: number;
}

export type TimeSeriesPoint = [number, number];

export interface TimeSeriesSeries {
  name: string;
  points: TimeSeriesPoint[];
}

export interface TimeSeriesPayload {
  unit?: string;
  series: TimeSeriesSeries[];
}

export interface CategoricalItem {
  label: string;
  value: number;
}

export interface CategoricalPayload {
  unit?: string;
  items: CategoricalItem[];
}

export interface EntityListItem {
  id: string;
  label: string;
  status: 'ok' | 'warn' | 'down' | 'unknown';
  extra?: Record<string, unknown>;
}

export interface EntityListPayload {
  items: EntityListItem[];
}

export interface DataPayload {
  shape: Shape;
  scalar?: ScalarPayload;
  time_series?: TimeSeriesPayload;
  categorical?: CategoricalPayload;
  entity_list?: EntityListPayload;
}

export interface DataSourceView {
  id: number;
  name: string;
  type: string;
  endpoint: string;
  auth: unknown;
  extra: unknown;
  last_health: 'ok' | 'error' | 'unknown' | '';
  last_health_at: string;
  created_at: string;
  updated_at: string;
}

export interface MetricFetchResponse {
  data_source_id: number;
  path: string;
  shape: Shape;
  payload: DataPayload;
  cached_at: number;
}

export interface MetricBatchItem {
  data_source_id: number;
  path: string;
  shape: Shape;
  payload?: DataPayload;
  error?: string;
}
