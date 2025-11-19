package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("HTTP_PORT")
	if port == "" { port = "8080" }

	staticDir := os.Getenv("SERVER_STATIC_DIR")
	if staticDir == "" { staticDir = "./static" }

	fs := http.FileServer(http.Dir(staticDir))
	http.Handle("/", fs)

	log.Printf("🚀 Shell (Host) listening at :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
