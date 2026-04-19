import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const backend = process.env.VITE_DEV_PROXY || 'http://127.0.0.1:3837'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: '../web/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/download': { target: backend, changeOrigin: true },
      '/playlist': { target: backend, changeOrigin: true },
      '/channel': { target: backend, changeOrigin: true },
      '/video': { target: backend, changeOrigin: true },
      '/url': { target: backend, changeOrigin: true },
      '/health': { target: backend, changeOrigin: true },
      '/swagger': { target: backend, changeOrigin: true },
    },
  },
})
