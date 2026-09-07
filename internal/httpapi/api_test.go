package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"bughunter-ai/internal/cases"
	"bughunter-ai/internal/storage"
)

func serveurDeTest(t *testing.T) *httptest.Server {
	chemin := filepath.Join(t.TempDir(), "test.db")
	db, err := storage.Open(chemin, "../../migrations/001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = cases.LoadFixtures(db, "../../fixtures/cases.json")
	if err != nil {
		t.Fatal(err)
	}

	s := &Server{DB: db}
	ts := httptest.NewServer(s.Routes())
	t.Cleanup(ts.Close)
	return ts
}

func seConnecter(t *testing.T, ts *httptest.Server) *http.Cookie {
	corps := `{"username":"nossayba","password":"monmotdepasse"}`
	resp, err := http.Post(ts.URL+"/api/register", "application/json", strings.NewReader(corps))
	if err != nil || resp.StatusCode != 201 {
		t.Fatal("inscription ratee, code :", resp.StatusCode)
	}

	resp, err = http.Post(ts.URL+"/api/login", "application/json", strings.NewReader(corps))
	if err != nil || resp.StatusCode != 200 {
		t.Fatal("connexion ratee")
	}
	for _, c := range resp.Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	t.Fatal("aucun cookie session recu")
	return nil
}

func getAvecCookie(t *testing.T, url string, cookie *http.Cookie) *http.Response {
	req, _ := http.NewRequest("GET", url, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestInscriptionEtConnexion(t *testing.T) {
	ts := serveurDeTest(t)
	cookie := seConnecter(t, ts)
	if !cookie.HttpOnly {
		t.Error("le cookie de session doit etre HttpOnly (section 9.1)")
	}
}

func TestMotDePasseTropCourt(t *testing.T) {
	ts := serveurDeTest(t)
	corps := `{"username":"nossayba","password":"court"}`
	resp, _ := http.Post(ts.URL+"/api/register", "application/json", strings.NewReader(corps))
	if resp.StatusCode != 400 {
		t.Error("attendu 400, obtenu", resp.StatusCode)
	}
}

func TestSansSessionRefuse(t *testing.T) {
	ts := serveurDeTest(t)
	resp := getAvecCookie(t, ts.URL+"/api/cases", nil)
	if resp.StatusCode != 401 {
		t.Error("sans session : attendu 401, obtenu", resp.StatusCode)
	}
}

func TestListeEtFiltres(t *testing.T) {
	ts := serveurDeTest(t)
	cookie := seConnecter(t, ts)

	resp := getAvecCookie(t, ts.URL+"/api/cases", cookie)
	var liste []cases.Resume
	json.NewDecoder(resp.Body).Decode(&liste)
	if len(liste) != 6 {
		t.Error("attendu 6 enquetes, obtenu", len(liste))
	}

	resp = getAvecCookie(t, ts.URL+"/api/cases?language=sql", cookie)
	liste = nil
	json.NewDecoder(resp.Body).Decode(&liste)
	if len(liste) != 2 {
		t.Error("filtre sql : attendu 2 enquetes, obtenu", len(liste))
	}
	for _, r := range liste {
		if r.Language != "sql" {
			t.Error("le filtre a laisse passer", r.Language)
		}
	}
}

func TestEnqueteIntrouvable(t *testing.T) {
	ts := serveurDeTest(t)
	cookie := seConnecter(t, ts)
	resp := getAvecCookie(t, ts.URL+"/api/cases/n-existe-pas", cookie)
	if resp.StatusCode != 404 {
		t.Error("attendu 404, obtenu", resp.StatusCode)
	}
}

func TestLaSolutionResteCachee(t *testing.T) {
	ts := serveurDeTest(t)
	cookie := seConnecter(t, ts)

	slugs := []string{
		"go-boucle-sans-fin", "go-indice-disparu",
		"js-bouton-silencieux", "js-total-mysterieux",
		"sql-dossiers-fantomes", "sql-compteur-impossible",
	}
	for _, slug := range slugs {
		resp := getAvecCookie(t, ts.URL+"/api/cases/"+slug, cookie)
		if resp.StatusCode != 200 {
			t.Fatal(slug, ": code", resp.StatusCode)
		}

		var brut json.RawMessage
		json.NewDecoder(resp.Body).Decode(&brut)
		texte := string(brut)

		if strings.Contains(texte, "correct_suspect_id") {
			t.Error(slug, ": la reponse contient correct_suspect_id")
		}
		if strings.Contains(texte, `"correct"`) {
			t.Error(slug, ": la reponse dit quelle correction est la bonne")
		}
		if strings.Contains(texte, "hints") {
			t.Error(slug, ": la reponse contient les indices")
		}
		if strings.Contains(texte, "lesson") {
			t.Error(slug, ": la reponse contient la lecon")
		}

		var pub cases.PublicCase
		json.Unmarshal(brut, &pub)
		if len(pub.Suspects) != 3 {
			t.Error(slug, ": il manque des suspects dans la vue publique")
		}
		if len(pub.Fixes) < 2 {
			t.Error(slug, ": il manque des corrections dans la vue publique")
		}
	}
}

func TestAffaireDuJourSansSolution(t *testing.T) {
	ts := serveurDeTest(t)
	cookie := seConnecter(t, ts)

	resp := getAvecCookie(t, ts.URL+"/api/cases/daily", cookie)
	if resp.StatusCode != 200 {
		t.Fatal("attendu 200, obtenu", resp.StatusCode)
	}

	var brut json.RawMessage
	json.NewDecoder(resp.Body).Decode(&brut)
	texte := string(brut)
	if strings.Contains(texte, "correct_suspect_id") || strings.Contains(texte, "hints") ||
		strings.Contains(texte, "lesson") {
		t.Error("l'affaire du jour laisse passer la solution")
	}

	var q cases.Quotidienne
	json.Unmarshal(brut, &q)
	if q.Slug == "" || q.Title == "" {
		t.Error("l'affaire du jour est vide")
	}

	// le dossier doit vraiment exister
	resp = getAvecCookie(t, ts.URL+"/api/cases/"+q.Slug, cookie)
	if resp.StatusCode != 200 {
		t.Error("le slug du jour ne s'ouvre pas, code", resp.StatusCode)
	}
}
