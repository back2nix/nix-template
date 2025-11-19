<script setup>
import { ref, onMounted } from 'vue'

const msg = ref('Connecting...')

// Этот компонент, загруженный в Shell, будет стучаться на порт 8081
onMounted(async () => {
  try {
    // Важно: полный URL, так как мы будем в Shell (на 8080)
    const res = await fetch('http://localhost:8081/api/hello')
    const data = await res.json()
    msg.value = data.message
  } catch (e) {
    msg.value = 'Error fetching from Greeter Service'
  }
})
</script>

<template>
  <div class="greeter-box">
    <h3>👋 Greeter Microservice Widget</h3>
    <p>Status: <strong>{{ msg }}</strong></p>
    <small>Loaded via Module Federation from :8081</small>
  </div>
</template>

<style scoped>
.greeter-box {
  border: 2px dashed #42b983;
  padding: 20px;
  border-radius: 8px;
  background: #f9f9f9;
  color: #333;
}
</style>
