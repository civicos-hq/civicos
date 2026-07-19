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
        // Civic palette — GREEN experiment. Anchored on #234F2C
        // forest green while we trial the green-and-white palette
        // across the citizen app (homepage, auth, dashboard chrome).
        // Every `text-civic-700`, `bg-civic-50/40`, `border-civic-200`
        // used across detail pages / list pages / shared components
        // picks up the new tone in one edit. Ladder still spans
        // light → dark so existing `dark:civic-200`, `civic-500/40`
        // etc. compositions keep their contrast semantics. To revert
        // to the original navy palette, restore the previous hex
        // values (in git history).
        civic: {
          50: '#f0f7ef', // near-white mint tint (was #eff6ff)
          100: '#d7ecd6', // light mint (was #dbeafe)
          200: '#b8d9b6', // light green — border tone (was #bfdbfe)
          500: '#234F2C', // Forest Green — brand primary (was Civic Blue #2563EB)
          600: '#1e4527', // slightly darker for hover (was #2563eb)
          700: '#1b3d22', // deeper for text on light bg (was #1d4ed8)
          900: '#142e1a', // deep (was #1e3a8a)
          950: '#0a1f11', // deepest — hierarchy (was Deep Navy #0F2747)
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
