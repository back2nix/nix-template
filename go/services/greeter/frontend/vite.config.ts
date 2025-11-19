import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import federation from '@originjs/vite-plugin-federation'

export default defineConfig({
  plugins: [
    vue(),
    federation({
      name: 'greeter_app', // Имя удаленного приложения
      filename: 'remoteEntry.js', // Точка входа
      exposes: {
        // Экспортируем наш компонент
        './GreeterWidget': './src/components/GreeterWidget.vue',
      },
      shared: ['vue']
    })
  ],
  build: {
    target: 'esnext' // Обязательно для top-level await
  },
  server: {
    port: 8081 // Важно для дева
  }
})
