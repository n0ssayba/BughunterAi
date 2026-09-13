package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	// "secours" et pas "locale" : l'IA etait la mais a ete refusee, le
	// navigateur doit pouvoir le dire au joueur (15.3)
	if rep.Source != "secours" {
		t.Error("attendu la source secours, obtenu", rep.Source)
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
	rapportJSON := `{"observation":"Bien vu.","erreur":"","concept":"Les types.","question":"Et '1' + 1 ?"}`
	ts := serveurAvecIA(t, strconv.Quote(rapportJSON))
	cookie := seConnecter(t, ts)
	base := commencer(t, ts, cookie, "js-total-mysterieux")

	resp := postAvecCookie(t, ts.URL+base+"/verdict", `{"suspect_id":"A","fix_id":"F1"}`, cookie)
	var verdict struct {
		Assistant string `json:"assistant"`
	}
	json.NewDecoder(resp.Body).Decode(&verdict)
	if verdict.Assistant != "ia" {
		t.Error("attendu assistant=ia, obtenu", verdict.Assistant)
	}

	resp = getAvecCookie(t, ts.URL+base+"/report", cookie)
	var rapport struct {
		Commentaire struct {
			Observation string `json:"observation"`
			Question    string `json:"question"`
		} `json:"commentaire"`
	}
	json.NewDecoder(resp.Body).Decode(&rapport)
	if rapport.Commentaire.Observation == "" || rapport.Commentaire.Question == "" {
		t.Error("le rapport ne contient pas le retour structure de l'IA")
	}
}

func TestRapportSansCommentaireQuandLIAEchoue(t *testing.T) {
	// l'IA repond du texte libre au lieu du JSON : le rapport local reste
	// complet et le navigateur est prevenu (12, 15.3)
	ts := serveurAvecIA(t, `"Bravo detective."`)
	cookie := seConnecter(t, ts)
	base := commencer(t, ts, cookie, "js-total-mysterieux")

	resp := postAvecCookie(t, ts.URL+base+"/verdict", `{"suspect_id":"A","fix_id":"F1"}`, cookie)
	var verdict struct {
		Solved    bool   `json:"solved"`
		Assistant string `json:"assistant"`
	}
	json.NewDecoder(resp.Body).Decode(&verdict)
	if !verdict.Solved {
		t.Fatal("l'enquete devrait etre resolue malgre l'echec de l'IA")
	}
	if verdict.Assistant != "secours" {
		t.Error("attendu assistant=secours, obtenu", verdict.Assistant)
	}

	resp = getAvecCookie(t, ts.URL+base+"/report", cookie)
	var brut map[string]any
	json.NewDecoder(resp.Body).Decode(&brut)
	if brut["lesson"] == "" {
		t.Error("le rapport local est incomplet")
	}
	if _, present := brut["commentaire"]; present {
		t.Error("un commentaire invalide a ete enregistre dans le rapport")
	}
}
