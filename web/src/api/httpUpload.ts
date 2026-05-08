/**
 * HTTP multipart uploads. `kind` must be registered server-side (upload.KindProfiles).
 */

export type HttpUploadKind = 'icon' | 'wallpaper';

/**
 * Optional Aero glass tint precomputed by the server when uploading a
 * wallpaper. Frontend stores it on board.theme_custom.glass_tint so first
 * paint no longer needs the client-side `extractWallpaperTint` round-trip.
 */
export interface UploadedTint {
  glass_bg: string;
  border: string;
  glow: string;
  highlight: string;
  average_hex: string;
}

export interface UploadedFileRef {
  name: string;
  url: string;
  tint?: UploadedTint;
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

export async function httpUpload(kind: HttpUploadKind, file: File): Promise<UploadedFileRef> {
  const form = new FormData();
  form.append('kind', kind);
  form.append('file', file);
  const res = await fetch('/api/upload', { method: 'POST', body: form });
  if (!res.ok) {
    throw new Error(await readApiError(res));
  }
  return (await res.json()) as UploadedFileRef;
}

export async function httpListUploads(): Promise<{ items: UploadedFileRef[] }> {
  const res = await fetch('/api/upload');
  if (!res.ok) {
    throw new Error(await readApiError(res));
  }
  return (await res.json()) as { items: UploadedFileRef[] };
}

export async function httpDeleteUpload(name: string): Promise<void> {
  const q = new URLSearchParams({ name });
  const res = await fetch(`/api/upload?${q.toString()}`, { method: 'DELETE' });
  if (!res.ok) {
    throw new Error(await readApiError(res));
  }
}
