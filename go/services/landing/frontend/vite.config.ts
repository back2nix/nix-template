import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import federation from '@originjs/vite-plugin-federation'

const BASE_PATH = process.env.VITE_BASE_PATH || '/api/landing/'

export default defineConfig({
  base: BASE_PATH,
  plugins: [
    vue(),
    federation({
      name: 'landing_app',
      filename: 'remoteEntry.js',
      exposes: {
        './LandingWidget': './src/components/LandingWidget.vue',
      },
      shared: ['vue']
    })
  ],
  build: {
    target: 'esnext',
    emptyOutDir: true,
    // ИСПРАВЛЕНИЕ: remoteEntry.js должен быть в корне dist для Module Federation
    assetsDir: ''
  },
  server: {
    port: 8081
  }
})
