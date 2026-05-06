import { copyFileSync, existsSync, mkdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath, URL } from 'node:url';

import vue from '@vitejs/plugin-vue';
import { defineConfig } from 'vite';

const webRoot = fileURLToPath(new URL('.', import.meta.url));
const repoIco = fileURLToPath(new URL('../imgs/logo.ico', import.meta.url));

/**
 * Copies /imgs/logo.ico into public as favicon.ico for dev and embed builds (single source).
 */
function faviconFromImgs() {
  return {
    name: 'favicon-from-imgs',
    buildStart() {
      if (!existsSync(repoIco)) {
        // eslint-disable-next-line no-console -- build diagnostic
        console.warn(`[favicon-from-imgs] skipped: missing ${repoIco}`);
        return;
      }
      const outDir = path.join(webRoot, 'public');
      mkdirSync(outDir, { recursive: true });
      copyFileSync(repoIco, path.join(outDir, 'favicon.ico'));
    },
  };
}

export default defineConfig({
  plugins: [vue(), faviconFromImgs()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    outDir: fileURLToPath(new URL('../internal/embed/dist', import.meta.url)),
    emptyOutDir: true,
    sourcemap: false,
    target: 'es2022',
  },
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      '/healthz': 'http://127.0.0.1:8080',
      '/api': 'http://127.0.0.1:8080',
      '/uploads': 'http://127.0.0.1:8080',
      '/ws': {
        target: 'ws://127.0.0.1:8080',
        ws: true,
        changeOrigin: true,
      },
    },
  },
});
