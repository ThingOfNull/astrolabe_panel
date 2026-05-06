/**
 * Board config export/import over HTTP multipart (same style as /api/upload).
 */

export interface ConfigImportSummary {
  board_updated: boolean;
  data_sources_added: number;
  widgets_added: number;
  widgets_skipped?: number[];
}

async function readApiError(res: Response): Promise<string> {
  try {
    const j = (await res.json()) as { error?: unknown };
    if (j && typeof j.error === 'string' && j.error !== '') {
      return j.error;
    }
  } catch {
    // ignore
  }
  return res.statusText || `HTTP ${res.status}`;
}

export async function httpFetchConfigExport(): Promise<unknown> {
  const res = await fetch('/api/config/export');
  if (!res.ok) {
    throw new Error(await readApiError(res));
  }
  return res.json();
}

export async function downloadConfigBundleFile(): Promise<void> {
  const data = await httpFetchConfigExport();
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `astrolabe-config-${Date.now()}.json`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

export async function httpImportConfigFile(file: File): Promise<ConfigImportSummary> {
  const form = new FormData();
  form.append('file', file);
  const res = await fetch('/api/config/import', { method: 'POST', body: form });
  if (!res.ok) {
    throw new Error(await readApiError(res));
  }
  return (await res.json()) as ConfigImportSummary;
}
