import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    host: '0.0.0.0', // Listen on all interfaces
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://10.107.12.201:8080',
        changeOrigin: true,
        // Don't rewrite - server expects /api prefix
        agent: false, // Disable proxy agent to bypass system proxy
      },
    },
  },
})
