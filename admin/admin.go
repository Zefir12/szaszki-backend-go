package adminServer

import (
	"log"
	"net/http"

	adminEndpoints "github.com/zefir/szaszki-go-backend/admin/endpoints"
)

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5174") // frontend URL
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-Token")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

func StartAdminServer() {
	http.HandleFunc("/games", withCORS(adminEndpoints.GetGamesEndpoint)) // your "games" endpoint
	adminPort := "3132"

	log.Println("Admin server listening on", adminPort)
	if err := http.ListenAndServe("0.0.0.0:"+adminPort, nil); err != nil {
		log.Fatalf("Admin server failed: %v", err)
	}
}
