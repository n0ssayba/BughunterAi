package httpapi

import (
	"net/http"
	"time"

	"bughunter-ai/internal/cases"
)

func (s *Server) listeEnquetes(w http.ResponseWriter, r *http.Request, userID int64) {
	language := r.URL.Query().Get("language")
	difficulty := r.URL.Query().Get("difficulty")

	liste, err := cases.List(s.DB, language, difficulty)
	if err != nil {
		repondreErreur(w, 500, "erreur serveur")
		return
	}
	repondreJSON(w, 200, liste)
}

func (s *Server) detailEnquete(w http.ResponseWriter, r *http.Request, userID int64) {
	slug := r.PathValue("slug")

	c, err := cases.Get(s.DB, slug)
	if err == cases.ErrIntrouvable {
		repondreErreur(w, 404, "enquete introuvable")
		return
	}
	if err != nil {
		repondreErreur(w, 500, "erreur serveur")
		return
	}

	repondreJSON(w, 200, c.Public())
}

func (s *Server) affaireDuJour(w http.ResponseWriter, r *http.Request, userID int64) {
	q, err := cases.DuJour(s.DB, time.Now())
	if err != nil {
		repondreErreur(w, 500, "erreur serveur")
		return
	}
	repondreJSON(w, 200, q)
}
