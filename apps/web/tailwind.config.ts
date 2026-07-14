import type { Config } from 'tailwindcss';

const config: Config = {
  content: ['./index.html', './src/**/*.{ts,tsx}', '../../packages/ui/src/**/*.{ts,tsx}'],
  // Tailwind's `dark:` variants respond to the user's explicit theme
  // choice (persisted in localStorage → written to <html data-theme="…">
  // by the pre-paint script in index.html), not the OS's
  // prefers-color-scheme media query. Keeps the two in sync when the
  // user overrides their system preference.
  darkMode: ['selector', "[data-theme='dark']"],
  theme: {
    extend: {
      colors: {
        // Civic palette. `civic.500` is anchored on the brand's Civic
        // Blue (`#2563EB`) — every `bg-civic-500`, `text-civic-500`,
        // `border-civic-500` platform-wide picks up the exact brand
        // value in one edit. The rest of the ladder keeps the
        // original Tailwind-derived values so existing usages of
        // `civic-600`/`700`/`900` don't drift visually. `civic.950`
        // is a new addition for the brand's Deep Navy — for
        // hierarchy elements (headings on light bg, backgrounds
        // when we want warmer-than-slate-950).
        civic: {
          50: '#eff6ff',
          100: '#dbeafe',
          200: '#bfdbfe',
          500: '#2563EB', // Civic Blue — brand primary (was Tailwind blue-500 #3b82f6)
          600: '#2563eb',
          700: '#1d4ed8',
          900: '#1e3a8a',
          950: '#0F2747', // Deep Navy — brand primary hierarchy tone
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
};

export default config;
