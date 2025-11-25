import { createApp } from 'vue'
import App from './App.vue'
import { initTracing } from './tracing'

// 1. Init Observability
initTracing();

/**
 * RealtimeClient (Infrastructure Layer)
 */
class RealtimeClient {
    constructor(endpoint) {
        this.endpoint = endpoint;
        this.socket = null;
        this.subscribers = new Map();
        this.isConnected = false;

        this.connect();
    }

    connect() {
        console.log(`[Realtime] Connecting to ${this.endpoint}...`);
        this.socket = new WebSocket(this.endpoint);

        this.socket.onopen = () => {
            this.isConnected = true;
            console.log('[Realtime] Connected');
            this._notifySubscribers('system:connected', { status: 'ok' });
        };

        this.socket.onmessage = (messageEvent) => {
            try {
                const event = JSON.parse(messageEvent.data);
                if (event.type) {
                    this._notifySubscribers(event.type, event.payload);
                }
            } catch (e) {
                console.error('[Realtime] Parsing error', e);
            }
        };

        this.socket.onclose = () => {
            this.isConnected = false;
            console.log('[Realtime] Disconnected. Retrying in 3s...');
            setTimeout(() => this.connect(), 3000);
        };

        this.socket.onerror = (err) => {
             console.error('[Realtime] Socket Error', err);
        };
    }

    subscribe(eventType, callback) {
        if (!this.subscribers.has(eventType)) {
            this.subscribers.set(eventType, []);
        }
        this.subscribers.get(eventType).push(callback);
    }

    unsubscribe(eventType, callback) {
        if (!this.subscribers.has(eventType)) return;
        const filtered = this.subscribers.get(eventType).filter(cb => cb !== callback);
        this.subscribers.set(eventType, filtered);
    }

    _notifySubscribers(type, payload) {
        if (this.subscribers.has(type)) {
            this.subscribers.get(type).forEach(cb => cb(payload));
        }
    }
}

// Читаем gateway URL из переменной окружения
const GATEWAY_URL = import.meta.env.VITE_GATEWAY_URL || 'http://localhost:8080';
const WS_GATEWAY_URL = GATEWAY_URL.replace(/^http/, 'ws');

console.log('Shell: Gateway URL:', GATEWAY_URL);
console.log('Shell: WebSocket URL:', WS_GATEWAY_URL);

// Инициализация соединения
const realtimeClient = new RealtimeClient(`${WS_GATEWAY_URL}/ws`);

const app = createApp(App);
app.provide('realtime', realtimeClient);
app.mount('#app');
