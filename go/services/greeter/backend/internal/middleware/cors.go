package middleware

import (
	"net/http"
)

// CORS Middleware для обработки Cross-Origin запросов
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// FIX: Вместо "*" используем Origin из запроса, чтобы поддерживать credentials, если понадобятся
		// Но для простоты разработки пока оставим логику: если есть Origin, эхо-отвечаем им.
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		// FIX: Добавляем traceparent и tracestate в список разрешенных заголовков
		// Без этого браузер блокирует отправку контекста трейсинга на бэкенд.
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, traceparent, tracestate, baggage, x-request-id")

		// Разрешаем браузеру читать эти заголовки в ответе (полезно для отладки)
		w.Header().Set("Access-Control-Expose-Headers", "traceparent, tracestate")

		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Если это Preflight запрос (OPTIONS), завершаем его тут с 200 OK
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
