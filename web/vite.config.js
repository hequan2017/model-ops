import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 开发模式代理到 Go 后端；生产构建后由 Go 直接托管 dist
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': { target: 'http://127.0.0.1:8080', changeOrigin: true },
    },
  },
  build: { chunkSizeWarningLimit: 1500 },
})
