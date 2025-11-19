<script setup>
import { ref, onMounted } from 'vue'

const message = ref('Loading...')

onMounted(async () => {
  try {
    const res = await fetch('/api/hello')
    const data = await res.json()
    message.value = data.message
  } catch (e) {
    message.value = 'Error connecting to API'
  }
})
</script>

<template>
  <div style="text-align: center; margin-top: 50px; font-family: sans-serif;">
    <h1>Vue + Go + Nix</h1>
    <p>Backend says: <strong>{{ message }}</strong></p>
  </div>
</template>
