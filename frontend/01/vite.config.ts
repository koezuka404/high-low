import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const proxyTarget = process.env.VITE_PROXY_TARGET ?? 'http://localhost:8080'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: proxyTarget,
        changeOrigin: true,
      },
      '/login': {
        target: proxyTarget,
        changeOrigin: true,
      },
      '/signup': {
        target: proxyTarget,
        changeOrigin: true,
      },
      '/logout': {
        target: proxyTarget,
        changeOrigin: true,
      },
    },
  },
})
