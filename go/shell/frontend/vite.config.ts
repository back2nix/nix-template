import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import federation from '@originjs/vite-plugin-federation'

export default defineConfig({
  // Полифилл для process.env, который используется внутри OTel библиотек
  define: {
    'process.env': {}
  },
  plugins: [
    vue(),
    federation({
      name: 'shell_app',
      remotes: {
        greeter_app: 'http://localhost:8081/assets/remoteEntry.js',
      },
      shared: ['vue']
    })
  ],
  build: {
    target: 'esnext',
    commonjsOptions: {
      // Помогает Rollup правильно обрабатывать смешанные CJS/ESM модули OTel
      transformMixedEsModules: true,
    }
  },
  optimizeDeps: {
    // Принудительно включаем пакеты OTel в пре-бандлинг
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
  resolve: {
    alias: {
      '@': '/src'
    }
  }
})
