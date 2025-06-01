/// <reference types="vitest" />
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  // process.cwd() 在 vite.config.js 执行时，通常是项目的根目录（在这里是 'frontend'）
  // 第三个参数 '' 表示加载所有环境变量，不仅仅是 VITE_ 开头的。
  const env = loadEnv(mode, process.cwd(), '');

  console.log('--- vite.config.ts Debug ---');
  console.log('Current mode:', mode);
  console.log('Current working directory (process.cwd()):', process.cwd());
  console.log('Variables loaded by loadEnv:', env);
  console.log('PORT value from loadEnv:', env.PORT); // 这里应该是你 .env 文件中的 PORT 值

  // 使用从 .env 文件加载的 PORT，如果未定义，则回退到 3000
  const backendPort = env.PORT || '3000';
  console.log(`Using backend port for proxy: ${backendPort}`);
  console.log('-----------------------------');

  return {
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
          rewrite: (path) => path // 保持原样，如果后端API路径包含了/api，则不需要rewrite
        }
      }
    },
    build: {
      outDir: 'dist',
      emptyOutDir: true
    },
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: ['./src/__tests__/setup.ts'],
      css: true,
      coverage: {
        provider: 'v8',
        reporter: ['text', 'json', 'html'],
        exclude: [
          'node_modules/',
          'src/__tests__/',
          'src/**/*.test.{ts,tsx}',
          'src/**/*.spec.{ts,tsx}',
          '**/*.d.ts',
        ],
        thresholds: {
          global: {
            branches: 70,
            functions: 70,
            lines: 70,
            statements: 70,
          },
        },
      },
    }
  }
})
