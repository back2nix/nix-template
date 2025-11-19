import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import federation from '@originjs/vite-plugin-federation'

export default defineConfig({
  plugins: [
    vue(),
    federation({
      name: 'shell_app',
      remotes: {
        // Указываем, где искать удаленный модуль
        // В реальном проде localhost меняется на DNS имя сервиса
        greeter_app: 'http://localhost:8081/assets/remoteEntry.js',
      },
      shared: ['vue']
    })
  ],
  build: {
    target: 'esnext'
  }
})
