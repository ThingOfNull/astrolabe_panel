import { createPinia } from 'pinia';
import { createApp } from 'vue';

import { getRpc } from './api/jsonrpc';
import App from './App.vue';
import { i18n } from './i18n';
import { router } from './router';
import './styles/main.css';

// Bundled Inter + JetBrains Mono for offline / LAN (stable digit width).
import '@fontsource/inter/400.css';
import '@fontsource/inter/500.css';
import '@fontsource/inter/600.css';
import '@fontsource/jetbrains-mono/400.css';
import '@fontsource/jetbrains-mono/500.css';

const app = createApp(App);
app.use(createPinia());
app.use(router);
app.use(i18n);
app.mount('#app');

// Playwright test hook: exposes same RPC as production build (not gated on DEV).
(window as unknown as { __ASTROLABE_TEST_RPC?: typeof getRpc }).__ASTROLABE_TEST_RPC = ((
  method: string,
  params?: unknown,
) => getRpc().call(method, params as Record<string, unknown> | undefined)) as never;
