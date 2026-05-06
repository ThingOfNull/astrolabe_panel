import { createI18n } from 'vue-i18n';

import enUS from './locales/en-US.json';
import zhCN from './locales/zh-CN.json';

export type SupportedLocale = 'zh-CN' | 'en-US';

const STORAGE_KEY = 'astrolabe.locale';
const FALLBACK: SupportedLocale = 'zh-CN';

function detectLocale(): SupportedLocale {
  if (typeof window === 'undefined') {
    return FALLBACK;
  }
  const url = new URL(window.location.href);
  const fromQuery = url.searchParams.get('locale');
  if (fromQuery && isSupported(fromQuery)) {
    return fromQuery;
  }
  const fromStorage = window.localStorage.getItem(STORAGE_KEY);
  if (fromStorage && isSupported(fromStorage)) {
    return fromStorage;
  }
  const fromBrowser = window.navigator.language;
  if (fromBrowser.startsWith('zh')) {
    return 'zh-CN';
  }
  if (fromBrowser.startsWith('en')) {
    return 'en-US';
  }
  return FALLBACK;
}

function isSupported(locale: string): locale is SupportedLocale {
  return locale === 'zh-CN' || locale === 'en-US';
}

export const i18n = createI18n({
  legacy: false,
  locale: detectLocale(),
  fallbackLocale: FALLBACK,
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
  },
});

export function setLocale(locale: SupportedLocale): void {
  i18n.global.locale.value = locale;
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(STORAGE_KEY, locale);
  }
}
