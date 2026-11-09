import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'node:path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@contracts': path.resolve(__dirname, '../../contracts')
    }
  },
  server: {
    port: 5174,
    fs: {
      allow: [path.resolve(__dirname, '../..')]
    },
    proxy: {
      '/api': 'http://127.0.0.1:8080',
      '/v1': {
        target: 'http://127.0.0.1:8080',
        ws: true
      }
    }
  }
})
