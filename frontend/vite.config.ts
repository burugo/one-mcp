import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// 读取与后端相同的PORT环境变量，没有则默认为3000
const backendPort = process.env.PORT || 3000

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src')
    }
  },
  server: {
    port: 5173, // 明确指定开发服务器端口
    strictPort: true, // 如果端口被占用，则报错而不是尝试下一个端口
    proxy: {
      '^/api/.*': {
        target: `http://localhost:${backendPort}`,
        changeOrigin: true,
        secure: false,
        rewrite: (path) => path
      }
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
})
