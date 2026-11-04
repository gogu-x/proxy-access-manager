import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    host: "127.0.0.1",
    watch: { usePolling: true, interval: 1000 },
    proxy: {
      '/api': 'http://127.0.0.1:10001',
      '/health': 'http://127.0.0.1:10001',
      '/sub': 'http://127.0.0.1:10001',
    },
  },
})
