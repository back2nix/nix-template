package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	// Читаем SHELL_HTTP_PORT, а не HTTP_PORT
	port := os.Getenv("SHELL_HTTP_PORT")
	if port == "" {
		port = "9002"
	}

	staticDir := os.Getenv("SHELL_STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static"
	}

	fs := http.FileServer(http.Dir(staticDir))
	http.Handle("/", fs)

	log.Printf("🚀 Shell (Host) listening at :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
