package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/keyplate/pomoteam/internal/timer"
)

func main() {
	godotenv.Load()
	port := os.Getenv("SERVER_PORT")

	serveMux := http.NewServeMux()
	ts := timer.NewHubService()

	serveMux.HandleFunc("POST /api/timer", func(w http.ResponseWriter, r *http.Request) {
		timer.HandleCreateTimer(ts, w, r)
	})
	serveMux.HandleFunc("GET /ws/{hubId}", func(w http.ResponseWriter, r *http.Request) {
		timer.ServeWs(ts, w, r)
	})

	server := http.Server{Handler: serveMux, Addr: port}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
