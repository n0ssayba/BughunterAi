package httpapi

import (
	"encoding/json"
	"net/http"

	"bughunter-ai/internal/auth"
)

type identifiants struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var ids identifiants
	err := json.NewDecoder(r.Body).Decode(&ids)
	if err != nil {
		repondreErreur(w, 400, "corps de requete invalide")
		return
	}
	if len(ids.Username) < 3 {
		repondreErreur(w, 400, "le nom doit faire au moins 3 caracteres")
		return
	}
	if len(ids.Password) < 8 {
		repondreErreur(w, 400, "le mot de passe doit faire au moins 8 caracteres")
		return
	}

	err = auth.Register(s.DB, ids.Username, ids.Password)
	if err == auth.ErrNomDejaPris {
		repondreErreur(w, 409, "ce nom d'utilisateur est deja pris")
		return
	}
	if err != nil {
		repondreErreur(w, 500, "erreur serveur")
		return
	}
	repondreJSON(w, 201, map[string]string{"status": "compte cree"})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var ids identifiants
	err := json.NewDecoder(r.Body).Decode(&ids)
	if err != nil {
		repondreErreur(w, 400, "corps de requete invalide")
		return
	}

	jeton, err := auth.Login(s.DB, ids.Username, ids.Password)
	if err != nil {
		repondreErreur(w, 401, "identifiants invalides")
		return
	}

	// HttpOnly : le javascript de la page ne peut pas lire le cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    jeton,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	repondreJSON(w, 200, map[string]string{"status": "connecte"})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		auth.Logout(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name: "session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
	})
	repondreJSON(w, 200, map[string]string{"status": "deconnecte"})
}
