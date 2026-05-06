/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,ts,tsx}'],
  theme: {
    extend: {
      colors: {
        astro: {
          bg: 'var(--astro-bg-base)',
          glass: 'var(--astro-glass-bg)',
          border: 'var(--astro-glass-border)',
          text: 'var(--astro-text-primary)',
          subtle: 'var(--astro-text-secondary)',
          ok: 'var(--astro-status-ok)',
          err: 'var(--astro-status-err)',
          unknown: 'var(--astro-status-unknown)',
        },
      },
      fontFamily: {
        mono: ['"JetBrains Mono"', '"Fira Code"', 'ui-monospace', 'monospace'],
      },
    },
  },
  plugins: [],
};
