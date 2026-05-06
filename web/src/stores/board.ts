import { defineStore } from 'pinia';
import { ref } from 'vue';

import { getRpc } from '@/api/jsonrpc';

export interface Board {
  id: number;
  name: string;
  grid_base_unit: number;
  wallpaper: string;
  theme: string;
  theme_custom: string;
  updated_at: string;
}

export const useBoardStore = defineStore('board', () => {
  const board = ref<Board | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchBoard(): Promise<void> {
    loading.value = true;
    error.value = null;
    try {
      const result = await getRpc().call<Board>('board.get');
      board.value = result;
    } catch (err) {
      error.value = formatError(err);
    } finally {
      loading.value = false;
    }
  }

  async function update(patch: Partial<Board>): Promise<Board> {
    const result = await getRpc().call<Board>('board.update', patch as Record<string, unknown>);
    board.value = result;
    return result;
  }

  return { board, loading, error, fetchBoard, update };
});

function formatError(err: unknown): string {
  if (err && typeof err === 'object' && 'message' in err) {
    return String((err as { message: unknown }).message);
  }
  return String(err);
}
