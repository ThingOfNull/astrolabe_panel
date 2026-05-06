// Coords are grid multipliers; px = multiplier * board.grid_base_unit.
export interface Rect {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface Board {
  id: number;
  name: string;
  grid_base_unit: number;
  wallpaper: string;
  theme: string;
  theme_custom: string;
  updated_at: string;
}

export interface Widget extends Rect {
  id: number;
  board_id: number;
  type: 'link' | 'search' | string;
  z_index: number;
  icon_type: '' | 'INTERNAL' | 'REMOTE' | 'ICONIFY';
  icon_value: string;
  data_source_id: number | null;
  metric_query: unknown;
  config: unknown;
  created_at?: string;
  updated_at?: string;
}

export type CanvasMode = 'view' | 'edit';

// Logical design size in grid units. View mode uses px = mult * base; edit mode may scale preview.
export const DESIGN_GRID_WIDTH = 200;
export const DESIGN_GRID_HEIGHT = 120;
