import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Admin console — separate origin from the citizen surface so its
// security posture can diverge (stricter CSP, separate cookies, etc).
// Runs on :5174 alongside the citizen app on :5173.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5174,
    strictPort: true,
  },
});
