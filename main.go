package main

import (
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	port := os.Getenv("SERVER_PORT")

	serveMux := http.NewServeMux()
	ts := &HubService{hubPool: map[uuid.UUID]*hub{}}

	serveMux.HandleFunc("POST /api/timer", func(w http.ResponseWriter, r *http.Request) {
		HandleCreateTimer(ts, w, r)
	})
	serveMux.HandleFunc("GET /ws/{timerId}", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(ts, w, r)
	})

	server := http.Server{Handler: serveMux, Addr: port}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
