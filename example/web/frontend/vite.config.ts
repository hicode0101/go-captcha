import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      // 开发模式下把 API 请求代理到 Go 后端
      '/api': 'http://localhost:8090',
    },
  },
})
