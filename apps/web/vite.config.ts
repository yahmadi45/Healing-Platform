import { loadEnv } from 'vite';
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, '.', '');
  const target = env.SENTINELOPS_API_TARGET || 'http://localhost:8080';

  return {
    plugins: [react()],
    server: {
      proxy: {
        '/api': { target, changeOrigin: true },
        '/healthz': { target, changeOrigin: true },
        '/readyz': { target, changeOrigin: true },
        '/metrics': { target, changeOrigin: true },
      },
    },
    test: {
      environment: 'jsdom',
      setupFiles: './src/test/setup.ts',
      restoreMocks: true,
    },
  };
});
