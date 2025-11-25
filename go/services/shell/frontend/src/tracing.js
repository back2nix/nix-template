/* shell/frontend/src/tracing.js */
import { WebTracerProvider } from '@opentelemetry/sdk-trace-web';
import { SimpleSpanProcessor, BatchSpanProcessor, ConsoleSpanExporter } from '@opentelemetry/sdk-trace-base';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch';
import { XMLHttpRequestInstrumentation } from '@opentelemetry/instrumentation-xml-http-request';
import { UserInteractionInstrumentation } from '@opentelemetry/instrumentation-user-interaction';
import { ZoneContextManager } from '@opentelemetry/context-zone';
import { Resource } from '@opentelemetry/resources';
import { SemanticResourceAttributes } from '@opentelemetry/semantic-conventions';

export function initTracing() {
  console.log('Initializing Tracing...');

  // ИСПРАВЛЕНИЕ: Используем VITE_OTEL_ENDPOINT, переданный из justfile.
  // Если переменная не задана, пытаемся использовать Gateway (18080) как прокси к OTel.
  // Прямое подключение к 14318 тоже возможно, но Gateway надежнее из-за CORS.
  const collectorUrl = import.meta.env.VITE_OTEL_ENDPOINT || 'http://localhost:18080/v1/traces';

  console.log('OTEL Collector URL:', collectorUrl);

  const exporter = new OTLPTraceExporter({
    url: collectorUrl,
  });

  const provider = new WebTracerProvider({
    resource: new Resource({
      [SemanticResourceAttributes.SERVICE_NAME]: 'shell-frontend',
      [SemanticResourceAttributes.SERVICE_VERSION]: '1.0.0',
      'deployment.environment': import.meta.env.MODE || 'development',
    }),
  });

  // Включаем консольный вывод для отладки, если что-то идет не так
  if (import.meta.env.DEV) {
     provider.addSpanProcessor(new SimpleSpanProcessor(new ConsoleSpanExporter()));
  }

  provider.addSpanProcessor(new BatchSpanProcessor(exporter, {
    scheduledDelayMillis: 1000,
    maxExportBatchSize: 100,
  }));

  provider.register({
    contextManager: new ZoneContextManager(),
  });

  registerInstrumentations({
    tracerProvider: provider,
    instrumentations: [
      new UserInteractionInstrumentation({
        eventNames: ['click', 'submit'],
      }),
      new FetchInstrumentation({
        // Важно: не трассируем запросы к самому коллектору
        ignoreUrls: [collectorUrl, /.*\/v1\/traces/],
        propagateTraceHeaderCorsUrls: [
            /.*/ // Разрешаем передачу заголовков трассировки на все домены (включая API)
        ],
        clearTimingResources: true,
      }),
      new XMLHttpRequestInstrumentation({
        ignoreUrls: [collectorUrl, /.*\/v1\/traces/],
        propagateTraceHeaderCorsUrls: [
            /.*/
        ],
      }),
    ],
  });

  console.log('✅ Frontend Tracing Initialized via', collectorUrl);
  return provider.getTracer('shell-frontend');
}
