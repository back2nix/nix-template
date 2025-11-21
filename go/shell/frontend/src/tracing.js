/* shell/frontend/src/tracing.js */
import { WebTracerProvider } from '@opentelemetry/sdk-trace-web';
import { SimpleSpanProcessor, BatchSpanProcessor, ConsoleSpanExporter } from '@opentelemetry/sdk-trace-base';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch';
import { XMLHttpRequestInstrumentation } from '@opentelemetry/instrumentation-xml-http-request';
import { ZoneContextManager } from '@opentelemetry/context-zone';
import { Resource } from '@opentelemetry/resources';
import { SemanticResourceAttributes } from '@opentelemetry/semantic-conventions';

export function initTracing() {
  console.log('Initializing Tracing...');

  // 1. Экспортер (OTLP HTTP -> Collector)
  const exporter = new OTLPTraceExporter({
    url: 'http://localhost:4318/v1/traces',
  });

  // 2. Провайдер с Ресурсом
  const provider = new WebTracerProvider({
    resource: new Resource({
      [SemanticResourceAttributes.SERVICE_NAME]: 'shell-frontend',
      [SemanticResourceAttributes.SERVICE_VERSION]: '1.0.0',
      'deployment.environment': 'development',
    }),
  });

  // 3. Процессор
  // Включаем ConsoleSpanExporter, чтобы ты сразу видел спаны в консоли браузера (F12)
  // Если спаны печатаются в консоль, значит JS работает и проблема только в сети.
  provider.addSpanProcessor(new SimpleSpanProcessor(new ConsoleSpanExporter()));

  // Отправка в Collector
  provider.addSpanProcessor(new SimpleSpanProcessor(exporter));

  // 4. Context Manager
  provider.register({
    contextManager: new ZoneContextManager(),
  });

  // 5. Инструментация
  registerInstrumentations({
    tracerProvider: provider,
    instrumentations: [
      new FetchInstrumentation({
        propagateTraceHeaderCorsUrls: [
            /localhost:.+/, // Ловит 8080, 9002 и прочие порты на локалхосте
            /127\.0\.0\.1:.+/
        ],
        clearTimingResources: true,
      }),
      new XMLHttpRequestInstrumentation({
        propagateTraceHeaderCorsUrls: [
            /localhost:.+/,
            /127\.0\.0\.1:.+/
        ],
      }),
    ],
  });

  console.log('✅ Frontend Tracing Initialized');
  return provider.getTracer('shell-frontend');
}
