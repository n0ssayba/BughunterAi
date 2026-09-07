package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func postAvecCookie(t *testing.T, url string, corps string, cookie *http.Cookie) *http.Response {
	req, _ := http.NewRequest("POST", url, strings.NewReader(corps))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func commencer(t *testing.T, ts *httptest.Server, cookie *http.Cookie, slug string) string {
	resp := postAvecCookie(t, ts.URL+"/api/investigations", `{"slug":"`+slug+`"}`, cookie)
	if resp.StatusCode != 201 {
		t.Fatal("demarrage rate, code :", resp.StatusCode)
	}
	var corps struct {
		InvestigationID int64 `json:"investigation_id"`
	}
	json.NewDecoder(resp.Body).Decode(&corps)
	return "/api/investigations/" + jsonNombre(corps.InvestigationID)
}

func jsonNombre(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func TestPartieComplete(t *testing.T) {
	ts := serveurDeTest(t)
	cookie := seConnecter(t, ts)
	base := commencer(t, ts, cookie, "js-total-mysterieux")

	resp := postAvecCookie(t, ts.URL+base+"/suspects", `{"suspect_id":"B"}`, cookie)
	var rep map[string]bool
	json.NewDecoder(resp.Body).Decode(&rep)
	if rep["correct"] {
		t.Error("B ne devrait pas etre la bonne cause")
	}

	resp = postAvecCookie(t, ts.URL+base+"/hints", "", cookie)
	if resp.StatusCode != 200 {
		t.Fatal("indice refuse, code :", resp.StatusCode)
	}

	resp = postAvecCookie(t, ts.URL+base+"/verdict", `{"suspect_id":"A","fix_id":"F1"}`, cookie)
	var verdict struct {
		Solved bool `json:"solved"`
		Score  int  `json:"score"`
	}
	json.NewDecoder(resp.Body).Decode(&verdict)
	if !verdict.Solved || verdict.Score != 95 {
		t.Error("attendu solved=true et score=95, obtenu", verdict)
	}

	resp = getAvecCookie(t, ts.URL+base+"/report", cookie)
	if resp.StatusCode != 200 {
		t.Error("rapport inaccessible apres resolution, code :", resp.StatusCode)
	}

	resp = getAvecCookie(t, ts.URL+"/api/progress", cookie)
	var prog struct {
		ScoreTotal       int `json:"score_total"`
		EnquetesResolues int `json:"enquetes_resolues"`
	}
	json.NewDecoder(resp.Body).Decode(&prog)
	if prog.EnquetesResolues != 1 || prog.ScoreTotal != 95 {
		t.Error("progression incorrecte :", prog)
	}
}

func TestRapportInterditAvantResolution(t *testing.T) {
	ts := serveurDeTest(t)
	cookie := seConnecter(t, ts)
	base := commencer(t, ts, cookie, "go-boucle-sans-fin")

	resp := getAvecCookie(t, ts.URL+base+"/report", cookie)
	if resp.StatusCode != 403 {
		t.Error("attendu 403, obtenu", resp.StatusCode)
	}
}

func TestVerdictApresResolutionRefuse(t *testing.T) {
	ts := serveurDeTest(t)
	cookie := seConnecter(t, ts)
	base := commencer(t, ts, cookie, "go-boucle-sans-fin")

	postAvecCookie(t, ts.URL+base+"/verdict", `{"suspect_id":"B","fix_id":"F1"}`, cookie)
	resp := postAvecCookie(t, ts.URL+base+"/verdict", `{"suspect_id":"B","fix_id":"F1"}`, cookie)
	if resp.StatusCode != 409 {
		t.Error("attendu 409, obtenu", resp.StatusCode)
	}
}

func TestBoucleDeJeuSansSession(t *testing.T) {
	ts := serveurDeTest(t)
	resp := postAvecCookie(t, ts.URL+"/api/investigations", `{"slug":"go-boucle-sans-fin"}`, nil)
	if resp.StatusCode != 401 {
		t.Error("attendu 401, obtenu", resp.StatusCode)
	}
}
