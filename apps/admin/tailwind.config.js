/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'ui-monospace', 'monospace'],
      },
      colors: {
        // Admin console reads as institutional rather than citizen-facing.
        // Deep slate + civic blue accent — same accent as the citizen app
        // for family, but muted and utilitarian.
        civic: {
          50: '#f0f7ff',
          100: '#e0efff',
          500: '#1f6ed4',
          600: '#1858ad',
          700: '#174d95',
          900: '#0e2f5f',
        },
      },
    },
  },
  plugins: [],
};
