// Minimal HTTP server for Docker e2e tests.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
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
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	log.Printf("e2e app listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
