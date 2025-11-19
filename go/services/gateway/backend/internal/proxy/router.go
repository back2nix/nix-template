package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"gateway/internal/config"
)

func RegisterRoutes(mux *http.ServeMux, cfg *config.Config) {
	// Greeter Service
	greeterProxy := createProxy(cfg.Services.Greeter.URL)
	mux.Handle("/api/greeter/", http.StripPrefix("/api/greeter", greeterProxy))

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Main page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
    <title>API Gateway</title>
    <style>
        body { font-family: monospace; max-width: 800px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        .endpoint { background: #f4f4f4; padding: 10px; margin: 10px 0; border-radius: 5px; }
        .method { color: #0066cc; font-weight: bold; }
        a { color: #0066cc; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <h1>🚀 API Gateway</h1>
    <p>Available endpoints:</p>

    <div class="endpoint">
        <span class="method">GET</span>
        <a href="/health">/health</a> - Gateway health check
    </div>

    <div class="endpoint">
        <span class="method">GET</span>
        <a href="/api/greeter/health">/api/greeter/health</a> - Greeter service health
    </div>

    <div class="endpoint">
        <span class="method">GET</span>
        <a href="/api/greeter/api/hello?name=World">/api/greeter/api/hello?name=World</a> - Greeter API
    </div>

    <hr>
    <p>Services:</p>
    <ul>
        <li><a href="http://localhost:9002">Shell (Host)</a> - :9002</li>
        <li><a href="http://localhost:8081">Greeter Service</a> - :8081</li>
    </ul>
</body>
</html>`))
	})
}

func createProxy(targetURL string) http.Handler {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("Invalid service URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `{"error":"Service unavailable"}`)
	}

	return proxy
}
