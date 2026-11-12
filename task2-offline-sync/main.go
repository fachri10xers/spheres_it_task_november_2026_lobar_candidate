package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	port := envOr("PORT", "8000")
	dbPath := envOr("DB_PATH", "sync.db")

	st, err := openStore(dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.close()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /sync", handleSync(st))
	mux.HandleFunc("GET /patients/{id}", handleGetPatient(st))
	mux.HandleFunc("GET /stock/{id}", handleGetStock(st))
	mux.HandleFunc("GET /review-queue", handleReviewQueue(st))
	mux.HandleFunc("POST /reset", handleReset(st))

	addr := ":" + port
	log.Printf("sync server listening on %s (db: %s)", addr, dbPath)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
