import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import federation from '@originjs/vite-plugin-federation'
import path from 'path'

// RUNTIME CONFIG: В k8s эти значения будут переопределены через window.__RUNTIME_CONFIG__
const LANDING_REMOTE = process.env.VITE_LANDING_REMOTE_URL || 'http://localhost:18080/api/landing/remoteEntry.js'
const CHAT_REMOTE = process.env.VITE_CHAT_REMOTE_URL || 'http://localhost:18080/api/chat/remoteEntry.js'

export default defineConfig({
  define: {
    'process.env': {}
  },
  plugins: [
    vue(),
    federation({
      name: 'shell_app',
      remotes: {
        landing_app: {
          external: `Promise.resolve(window.__RUNTIME_CONFIG__?.LANDING_REMOTE_URL || '${LANDING_REMOTE}')`,
          externalType: 'promise'
        },
        chat_app: {
          external: `Promise.resolve(window.__RUNTIME_CONFIG__?.CHAT_REMOTE_URL || '${CHAT_REMOTE}')`,
          externalType: 'promise'
        }
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
    commonjsOptions: {
      transformMixedEsModules: true,
    }
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
  resolve: {
    alias: {
      '@': '/src',
      '@opentelemetry/api/package.json': path.resolve(__dirname, 'node_modules/@opentelemetry/api/package.json')
    }
  }
})
