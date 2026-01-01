import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  server: {
    host: '0.0.0.0',
    port: 3000,
    proxy: {
      '/api': {
        // target: 'http://192.168.110.63:8475',
        target: 'http://127.0.0.1:8475',
        changeOrigin: true,
        ws: true  // 启用 WebSocket 代理
      }
    }
  }
})
