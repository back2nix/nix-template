import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import federation from '@originjs/vite-plugin-federation'
import path from 'path'

const BASE_PATH = process.env.VITE_BASE_PATH || '/api/chat/'
// ИСПРАВЛЕНИЕ: Используем порт, соответствующий .env (18082), чтобы избежать путаницы при dev запуске
const PORT = Number(process.env.CHAT_SERVER_HTTP_PORT) || 18082

export default defineConfig({
  base: BASE_PATH,
  plugins: [
    vue(),
    federation({
      name: 'chat_app',
      filename: 'remoteEntry.js',
      exposes: {
        './ChatWidget': './src/components/ChatWidget.vue',
      },
      shared: {
        vue: {},
        '@opentelemetry/api': {
          singleton: true,
          requiredVersion: false
        }
      }
    })
  ],
  build: {
    target: 'esnext',
    emptyOutDir: true,
    assetsDir: ''
  },
  server: {
    port: PORT
  },
  resolve: {
    alias: {
      '@opentelemetry/api/package.json': path.resolve(__dirname, 'node_modules/@opentelemetry/api/package.json')
    }
  }
})
