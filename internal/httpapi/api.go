package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"bughunter-ai/internal/ai"
	"bughunter-ai/internal/auth"
)

type Server struct {
	DB *sql.DB
	AI *ai.Client
}

func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		repondreJSON(w, 200, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /api/register", s.register)
	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("POST /api/logout", s.logout)

	mux.HandleFunc("GET /api/cases", s.avecSession(s.listeEnquetes))
	// Go 1.22 : le motif litteral passe avant {slug}, "daily" n'est donc
	// jamais pris pour un slug
	mux.HandleFunc("GET /api/cases/daily", s.avecSession(s.affaireDuJour))
	mux.HandleFunc("GET /api/cases/{slug}", s.avecSession(s.detailEnquete))

	mux.HandleFunc("POST /api/investigations", s.avecSession(s.commencerEnquete))
	mux.HandleFunc("POST /api/investigations/{id}/suspects", s.avecSession(s.enregistrerHypothese))
	mux.HandleFunc("POST /api/investigations/{id}/hints", s.avecSession(s.debloquerIndice))
	mux.HandleFunc("POST /api/investigations/{id}/verdict", s.avecSession(s.rendreVerdict))
	mux.HandleFunc("GET /api/investigations/{id}/report", s.avecSession(s.lireRapport))
	mux.HandleFunc("GET /api/progress", s.avecSession(s.lireProgression))

	return mux
}

func repondreJSON(w http.ResponseWriter, code int, donnees any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(donnees)
}

func repondreErreur(w http.ResponseWriter, code int, message string) {
	repondreJSON(w, code, map[string]string{"error": message})
}

// verifie le cookie avant d'appeler la route
func (s *Server) avecSession(suite func(http.ResponseWriter, *http.Request, int64)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			repondreErreur(w, 401, "connexion requise")
			return
		}
		userID, ok := auth.UserID(cookie.Value)
		if !ok {
			repondreErreur(w, 401, "session invalide")
			return
		}
		suite(w, r, userID)
	}
}
