import { createApp } from 'vue'
import App from './App.vue'
import { initTracing } from './tracing' // <-- Импорт

// 1. Запускаем трейсинг
initTracing();

// 2. Создаем приложение
createApp(App).mount('#app')
