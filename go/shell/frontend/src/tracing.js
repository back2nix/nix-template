/* shell/frontend/src/tracing.js */
import { WebTracerProvider } from '@opentelemetry/sdk-trace-web';
import { BatchSpanProcessor } from '@opentelemetry/sdk-trace-base';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch';
import { XMLHttpRequestInstrumentation } from '@opentelemetry/instrumentation-xml-http-request';
import { ZoneContextManager } from '@opentelemetry/context-zone';

// FIX: Используем namespace import для обхода ошибки Vite "Resource is not exported"
import * as resources from '@opentelemetry/resources';
// Пытаемся получить Resource из разных вариантов экспорта (ESM vs CJS)
const Resource = resources.Resource || resources.default?.Resource;

export function initTracing() {
  console.log('Initializing Tracing...');

  if (!Resource) {
    console.error('CRITICAL: Failed to load OpenTelemetry Resource class');
    return;
  }

  // 1. Экспортер
  // Убедитесь, что порт 4318 проброшен наружу из Docker-контейнера otel-collector
  // Если вы открываете сайт с хост-машины -> localhost:4318
  const exporter = new OTLPTraceExporter({
    url: 'http://localhost:4318/v1/traces',
  });

  // 2. Провайдер с Ресурсом
  const provider = new WebTracerProvider({
    resource: new Resource({
      // Используем строковые литералы вместо констант, чтобы избежать конфликтов версий
      'service.name': 'shell-frontend',
      'service.version': '1.0.0',
    }),
  });

  // 3. Процессор
  provider.addSpanProcessor(new BatchSpanProcessor(exporter));

  // 4. Context Manager
  provider.register({
    contextManager: new ZoneContextManager(),
  });

  // 5. Инструментация
  registerInstrumentations({
    instrumentations: [
      new FetchInstrumentation({
        propagateTraceHeaderCorsUrls: [
            /localhost:8080.+/, // Gateway
            /localhost:9002.+/  // Local backend
        ],
      }),
      new XMLHttpRequestInstrumentation({
        propagateTraceHeaderCorsUrls: [
            /localhost:8080.+/,
            /localhost:9002.+/
        ],
      }),
    ],
  });

  console.log('✅ Frontend Tracing Initialized');
  return provider.getTracer('shell-frontend');
}
