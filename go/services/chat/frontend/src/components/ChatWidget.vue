<script setup>
import { ref, defineProps, onMounted, onUnmounted } from 'vue'
import { trace, context, propagation } from '@opentelemetry/api';

const props = defineProps({
  chatId: { type: Number, default: 0 }
})

const messages = ref([
  { id: 1, text: "System: Connecting...", sender: "them" }
])
const input = ref("")
let socket = null
let reconnectTimer = null

const tracer = trace.getTracer('chat-widget');

// ИСПРАВЛЕНИЕ: Читаем из VITE_GATEWAY_URL, которую пробрасывает justfile
const GATEWAY_URL = import.meta.env.VITE_GATEWAY_URL || 'http://localhost:18080';
const WS_GATEWAY_URL = GATEWAY_URL.replace(/^http/, 'ws');

console.log('[ChatWidget] Gateway URL:', GATEWAY_URL);
console.log('[ChatWidget] WebSocket URL:', WS_GATEWAY_URL);

const connectWebSocket = () => {
  if (socket) socket.close();

  socket = new WebSocket(`${WS_GATEWAY_URL}/ws`);

  socket.onopen = () => {
    console.log("[ChatWidget] WS Connected to Notification Service");
    messages.value.push({ id: Date.now(), text: "System: Connected!", sender: "them" });
  };

  socket.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      console.log("[ChatWidget] WS Received:", data);

      let msgText = "";
      if (data.payload && data.payload.msg) {
        msgText = data.payload.msg;
      } else if (data.text) {
        msgText = data.text;
      } else if (data.message) {
        msgText = data.message;
      } else {
        msgText = JSON.stringify(data);
      }

      if (msgText) {
         messages.value.push({
            id: Date.now(),
            text: "Notification: " + msgText,
            sender: "them"
         });
      }
    } catch (e) {
      console.error("[ChatWidget] Failed to parse message", e, event.data);
    }
  };

  socket.onclose = (event) => {
    console.log("[ChatWidget] WS Closed", event);
    messages.value.push({ id: Date.now(), text: "System: Disconnected, reconnecting...", sender: "them" });
    reconnectTimer = setTimeout(connectWebSocket, 3000);
  };

  socket.onerror = (err) => {
    console.error("[ChatWidget] WS Error", err);
    socket.close();
  };
}

const sendMessage = async () => {
  if (!input.value) return;

  const text = input.value;
  messages.value.push({ id: Date.now(), text: text, sender: "me" });
  input.value = "";

  const span = tracer.startSpan('chat_send_message_http');

  await context.with(trace.setSpan(context.active(), span), async () => {
    try {
        const headers = {
            'Content-Type': 'application/json'
        };
        propagation.inject(context.active(), headers);

        const response = await fetch(`${GATEWAY_URL}/api/chat/messages`, {
            method: 'POST',
            headers: headers,
            body: JSON.stringify({ text: text })
        });

        if (!response.ok) {
            throw new Error('Server error: ' + response.status);
        }

        span.addEvent("message_sent_success");
    } catch (e) {
        console.error("[ChatWidget] Send error:", e);
        span.recordException(e);
        messages.value.push({ id: Date.now(), text: "System: Failed to send", sender: "them" });
    } finally {
        span.end();
    }
  });
}

onMounted(() => {
  connectWebSocket();
})

onUnmounted(() => {
  clearTimeout(reconnectTimer);
  if (socket) {
    socket.onclose = null;
    socket.close();
  }
})
</script>

<template>
  <div class="chat-box">
    <div class="chat-header">
      <h3>Chat #{{ chatId }} (Notifications Active)</h3>
    </div>

    <div class="messages">
      <div v-for="m in messages" :key="m.id" :class="['msg', m.sender]">
        {{ m.text }}
      </div>
    </div>

    <div class="input-area">
      <input v-model="input" @keyup.enter="sendMessage" placeholder="Type a message..." />
      <button @click="sendMessage" class="send-btn">Send</button>
    </div>
  </div>
</template>

<style scoped>
.chat-box { display: flex; flex-direction: column; height: 400px; background: #222; border: 1px solid #444; border-radius: 8px; }
.chat-header { padding: 10px; border-bottom: 1px solid #444; background: #333; }
.messages { flex: 1; padding: 10px; overflow-y: auto; display: flex; flex-direction: column; gap: 10px; }
.msg { padding: 8px 12px; border-radius: 15px; max-width: 70%; word-wrap: break-word; }
.msg.them { background: #444; align-self: flex-start; }
.msg.me { background: #0056b3; align-self: flex-end; }
.input-area { padding: 10px; display: flex; gap: 10px; border-top: 1px solid #444; }
input { flex: 1; padding: 8px; border-radius: 4px; border: 1px solid #555; background: #333; color: white; }
button { padding: 8px 16px; background: #0056b3; color: white; border: none; border-radius: 4px; cursor: pointer; }
button:hover { background: #004494; }
</style>
