<script setup>
import { ref } from 'vue'

// Заглушка данных, которые в будущем придут с бекенда Landing
const chats = ref([
  { id: 1, name: "Support Agent", desc: "Technical help" },
  { id: 2, name: "Sales Rep", desc: "Pricing questions" },
  { id: 3, name: "Community Manager", desc: "General talk" }
])

const openChat = (chat) => {
  // Отправляем событие в Shell
  const event = new CustomEvent('navigate-to-chat', { detail: chat });
  window.dispatchEvent(event);
}
</script>

<template>
  <div class="landing-box">
    <h2>👋 Welcome to Our Service</h2>
    <p>Select a person to start chatting:</p>

    <div class="chat-list">
      <div v-for="chat in chats" :key="chat.id" class="chat-card" @click="openChat(chat)">
        <div class="avatar">👤</div>
        <div class="info">
          <h3>{{ chat.name }}</h3>
          <span>{{ chat.desc }}</span>
        </div>
        <div class="arrow">→</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.landing-box { padding: 20px; background: #2a2a2a; border-radius: 10px; }
.chat-list { display: flex; flex-direction: column; gap: 10px; margin-top: 20px; }
.chat-card {
  display: flex; align-items: center; background: #333; padding: 15px;
  border-radius: 8px; cursor: pointer; transition: background 0.2s;
}
.chat-card:hover { background: #444; }
.avatar { font-size: 24px; margin-right: 15px; }
.info h3 { margin: 0; font-size: 16px; color: white; }
.info span { font-size: 12px; color: #aaa; }
.arrow { margin-left: auto; color: #666; }
</style>
