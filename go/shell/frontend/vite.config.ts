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
  },
  optimizeDeps: {
    include: [
      '@opentelemetry/api',
      '@opentelemetry/sdk-trace-web',
      '@opentelemetry/sdk-trace-base',
      '@opentelemetry/resources',
      '@opentelemetry/semantic-conventions',
      '@opentelemetry/exporter-trace-otlp-http',
      '@opentelemetry/instrumentation-fetch',
      '@opentelemetry/instrumentation-xml-http-request',
      '@opentelemetry/context-zone'
    ]
  },
  // Иногда нужно явно указать resolve alias, если пакеты двоятся
  resolve: {
    alias: {
      '@': '/src'
    }
  }
})
