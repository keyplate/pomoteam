package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/keyplate/pomoteam/internal"
	"github.com/keyplate/pomoteam/internal/timer"
)

func main() {
	godotenv.Load()
	port := os.Getenv("SERVER_PORT")

	serveMux := http.NewServeMux()
	ts := timer.NewHubService()

	serveMux.HandleFunc("/api/hub", internal.CorsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		timer.HandleCreateHub(ts, w, r)
	}))
	serveMux.HandleFunc("/api/hub/{hubId}", internal.CorsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		timer.HandleCheckHub(ts, w, r)
	}))
	serveMux.HandleFunc("GET /api/ws/{hubId}", func(w http.ResponseWriter, r *http.Request) {
		timer.ServeWs(ts, w, r)
	})

	server := http.Server{Handler: serveMux, Addr: port}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
