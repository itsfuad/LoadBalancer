package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	// Parse command-line flags
	serverID := flag.Int("id", 1, "Server ID (1, 2, 3, etc.)")
	port := flag.Int("port", 8001, "Port to listen on")
	flag.Parse()

	// Create a handler that shows which server responded
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := fmt.Sprintf("Response from SERVER %d (port %d) at %s\n",
			*serverID, *port, time.Now().Format("15:04:05"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
		log.Printf("Server %d handled request from %s", *serverID, r.RemoteAddr)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Server %d is healthy", *serverID)))
	})

	// Start the server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Server %d starting on port %d", *serverID, *port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server %d failed to start: %v", *serverID, err)
	}
}
