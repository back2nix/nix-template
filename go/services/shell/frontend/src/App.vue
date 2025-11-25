<script setup>
import { defineAsyncComponent, defineComponent, h, ref, onMounted, onUnmounted } from 'vue'

// Динамическая загрузка Landing
const LandingWidget = defineAsyncComponent({
  loader: () => import('landing_app/LandingWidget'),
  errorComponent: defineComponent({ render() { return h('div', '⚠️ Landing Service is offline') } }),
  timeout: 3000
})

// Динамическая загрузка Chat
const ChatWidget = defineAsyncComponent({
  loader: () => import('chat_app/ChatWidget'),
  errorComponent: defineComponent({ render() { return h('div', '⚠️ Chat Service is offline') } }),
  timeout: 3000
})

// Состояние навигации
const currentView = ref('landing') // 'landing' | 'chat'
const selectedChatId = ref(null)

// Обработчик события переключения на чат
const handleNavigateToChat = (event) => {
  console.log("Shell: navigating to chat", event.detail)
  selectedChatId.value = event.detail.id
  currentView.value = 'chat'
}

// Обработчик возврата назад
const handleBackToLanding = () => {
  selectedChatId.value = null
  currentView.value = 'landing'
}

onMounted(() => {
  window.addEventListener('navigate-to-chat', handleNavigateToChat)
})

onUnmounted(() => {
  window.removeEventListener('navigate-to-chat', handleNavigateToChat)
})
</script>

<template>
  <div class="shell-container">
    <header class="shell-header">
      <h1>🏢 Super Messenger</h1>
      <button v-if="currentView === 'chat'" @click="handleBackToLanding" class="back-btn">← Back</button>
    </header>

    <div class="widgets-area">
      <Suspense>
        <template #default>
          <div v-if="currentView === 'landing'">
            <LandingWidget />
          </div>
          <div v-else-if="currentView === 'chat'">
            <ChatWidget :chat-id="selectedChatId" />
          </div>
        </template>
        <template #fallback>
          <div class="loading">Loading Remote Module...</div>
        </template>
      </Suspense>
    </div>
  </div>
</template>

<style>
body { font-family: 'Segoe UI', sans-serif; background: #1e1e1e; color: #e0e0e0; margin: 0; }
.shell-container { max-width: 900px; margin: 0 auto; padding: 20px; }
.shell-header { display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid #333; padding-bottom: 20px; margin-bottom: 20px; }
.back-btn { padding: 8px 16px; background: #333; color: white; border: none; cursor: pointer; border-radius: 4px; }
.back-btn:hover { background: #444; }
.loading { color: #888; font-style: italic; }
</style>
