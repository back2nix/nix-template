<script setup>
import { defineAsyncComponent, defineComponent, h } from 'vue'

// Динамически загружаем компонент с удаленного сервера
// Если сервер лежит, показываем ошибку
const GreeterWidget = defineAsyncComponent({
  loader: () => import('greeter_app/GreeterWidget'),
  errorComponent: defineComponent({
    render() { return h('div', '⚠️ Greeter Service is offline') }
  }),
  timeout: 3000
})
</script>

<template>
  <div class="shell-container">
    <h1>🏢 Main Corporate Portal (Shell)</h1>
    <p>Host running on port 8080</p>

    <hr />

    <div class="widgets-area">
      <!-- Вставляем виджет из микросервиса -->
      <Suspense>
        <GreeterWidget />
        <template #fallback>
          <div>Loading Remote Module...</div>
        </template>
      </Suspense>
    </div>
  </div>
</template>

<style>
body { font-family: sans-serif; background: #222; color: white; }
.shell-container { max-width: 800px; margin: 0 auto; padding: 20px; }
.widgets-area { display: grid; gap: 20px; margin-top: 30px; }
</style>
