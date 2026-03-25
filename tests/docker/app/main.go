// Minimal HTTP server for Docker e2e tests.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const defaultBody = "hello from docker app"

func main() {
	body := os.Getenv("E2E_APP_BODY")
	if body == "" {
		body = defaultBody
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, body)
	})
	http.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "protected ok")
	})
	http.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		fmt.Fprint(w, "event: ready\n")
		fmt.Fprintf(w, "data: %s\n\n", body)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		time.Sleep(50 * time.Millisecond)
	})
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	log.Printf("e2e app listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
