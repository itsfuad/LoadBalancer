package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	// Start server 1 on port 8001
	go func() {
		mux1 := http.NewServeMux()
		mux1.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Response from SERVER 1 (port 8001) at %s", time.Now().Format("15:04:05"))
		})
		log.Println("Server 1 starting on port 8001")
		log.Fatal(http.ListenAndServe(":8001", mux1))
	}()

	// Start server 2 on port 8002
	go func() {
		mux2 := http.NewServeMux()
		mux2.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Response from SERVER 2 (port 8002) at %s", time.Now().Format("15:04:05"))
		})
		log.Println("Server 2 starting on port 8002")
		log.Fatal(http.ListenAndServe(":8002", mux2))
	}()

	// Keep the main goroutine alive
	select {}
}
