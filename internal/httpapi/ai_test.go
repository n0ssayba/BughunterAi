package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bughunter-ai/internal/ai"
	"bughunter-ai/internal/cases"
	"bughunter-ai/internal/storage"
	"path/filepath"
)

// serveurAvecIA lance l'application avec un faux OpenRouter derriere.
func serveurAvecIA(t *testing.T, reponseIA string) *httptest.Server {
	faux := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":` + reponseIA + `}}]}`))
	}))
	t.Cleanup(faux.Close)

	chemin := filepath.Join(t.TempDir(), "test.db")
	db, err := storage.Open(chemin, "../../migrations/001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	cases.LoadFixtures(db, "../../fixtures/cases.json")

	clientIA := ai.New("cle-de-test", "modele-de-test")
	clientIA.URL = faux.URL
	s := &Server{DB: db, AI: clientIA}
	ts := httptest.NewServer(s.Routes())
	t.Cleanup(ts.Close)
	return ts
}

func TestIndiceReformuleParLIA(t *testing.T) {
	ts := serveurAvecIA(t, `"Regardez le type des valeurs, detective."`)
	cookie := seConnecter(t, ts)
	base := commencer(t, ts, cookie, "js-total-mysterieux")

	resp := postAvecCookie(t, ts.URL+base+"/hints", "", cookie)
	var rep struct {
		Hint   string `json:"hint"`
		Source string `json:"source"`
	}
	json.NewDecoder(resp.Body).Decode(&rep)
	if rep.Source != "ia" {
		t.Error("attendu source ia, obtenu", rep.Source)
	}
	if !strings.Contains(rep.Hint, "detective") {
		t.Error("l'indice ne semble pas reformule :", rep.Hint)
	}
}

func TestReponseInvalideRepliLocal(t *testing.T) {
	// regle du sujet (15.2) : reponse invalide -> le contenu local
	// prend le relais. Le faux OpenRouter revele la correction.
	ts := serveurAvecIA(t, `"Il faut ecrire Number(a) + Number(b), voila tout."`)
	cookie := seConnecter(t, ts)
	base := commencer(t, ts, cookie, "js-total-mysterieux")

	resp := postAvecCookie(t, ts.URL+base+"/hints", "", cookie)
	var rep struct {
		Hint   string `json:"hint"`
		Source string `json:"source"`
	}
	json.NewDecoder(resp.Body).Decode(&rep)
	if rep.Source != "locale" {
		t.Error("attendu le repli local, obtenu source", rep.Source)
	}
	if rep.Hint != "Observe le type de chaque valeur." {
		t.Error("l'indice local attendu n'est pas la :", rep.Hint)
	}
}

func TestSansCleToutResteLocal(t *testing.T) {
	// mode de secours F12 : sans cle, le jeu fonctionne pareil
	ts := serveurDeTest(t) // le serveur habituel, sans IA
	cookie := seConnecter(t, ts)
	base := commencer(t, ts, cookie, "js-total-mysterieux")

	resp := postAvecCookie(t, ts.URL+base+"/hints", "", cookie)
	var rep struct {
		Source string `json:"source"`
	}
	json.NewDecoder(resp.Body).Decode(&rep)
	if rep.Source != "locale" {
		t.Error("sans cle, la source doit etre locale, obtenu", rep.Source)
	}
}

func TestCommentaireDansLeRapport(t *testing.T) {
	ts := serveurAvecIA(t, `"Affaire rondement menee, detective."`)
	cookie := seConnecter(t, ts)
	base := commencer(t, ts, cookie, "js-total-mysterieux")

	postAvecCookie(t, ts.URL+base+"/verdict", `{"suspect_id":"A","fix_id":"F1"}`, cookie)
	resp := getAvecCookie(t, ts.URL+base+"/report", cookie)
	var rapport struct {
		Commentaire string `json:"commentaire"`
	}
	json.NewDecoder(resp.Body).Decode(&rapport)
	if rapport.Commentaire == "" {
		t.Error("le rapport ne contient pas le commentaire de l'IA")
	}
}
