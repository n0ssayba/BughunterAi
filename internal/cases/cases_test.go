package cases

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bughunter-ai/internal/storage"
)

func enqueteValide() Case {
	return Case{
		Slug: "test-1", Title: "Test", Language: "go", Difficulty: "discovery",
		Concept:          "test",
		Incident:         Incident{Expected: "a", Observed: "b"},
		Code:             "code",
		Evidence:         []string{"preuve"},
		Suspects:         []Suspect{{ID: "A", Label: "a"}, {ID: "B", Label: "b"}, {ID: "C", Label: "c"}},
		CorrectSuspectID: "A",
		Hints:            []string{"h1", "h2", "h3"},
		Fixes:            []Fix{{ID: "F1", Label: "f1", Correct: true}, {ID: "F2", Label: "f2", Correct: false}},
		Lesson:           "lecon",
	}
}

func TestValidationEnqueteCorrecte(t *testing.T) {
	err := Validate(enqueteValide())
	if err != nil {
		t.Error("une enquete valide a ete refusee :", err)
	}
}

func TestValidationDeuxSuspects(t *testing.T) {
	c := enqueteValide()
	c.Suspects = c.Suspects[:2]
	if Validate(c) == nil {
		t.Error("une enquete avec deux suspects a ete acceptee")
	}
}

func TestValidationLangageInconnu(t *testing.T) {
	c := enqueteValide()
	c.Language = "python"
	if Validate(c) == nil {
		t.Error("un langage hors go/javascript/sql a ete accepte")
	}
}

func TestValidationSansCorrectionCorrecte(t *testing.T) {
	c := enqueteValide()
	c.Fixes[0].Correct = false
	if Validate(c) == nil {
		t.Error("une enquete sans correction correcte a ete acceptee")
	}
}

func TestValidationDeuxIndices(t *testing.T) {
	c := enqueteValide()
	c.Hints = c.Hints[:2]
	if Validate(c) == nil {
		t.Error("une enquete avec deux indices a ete acceptee")
	}
}

func TestPublicSansSolution(t *testing.T) {
	pub, err := json.Marshal(enqueteValide().Public())
	if err != nil {
		t.Fatal(err)
	}
	texte := string(pub)
	if strings.Contains(texte, "correct_suspect_id") {
		t.Error("la version publique contient correct_suspect_id")
	}
	if strings.Contains(texte, "correct") {
		t.Error("la version publique dit quelle correction est la bonne")
	}
	if strings.Contains(texte, "hints") || strings.Contains(texte, "lesson") {
		t.Error("la version publique contient les indices ou la lecon")
	}
}

func TestChargementDesFixtures(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "test.db")
	db, err := storage.Open(chemin, "../../migrations/001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	n, err := LoadFixtures(db, "../../fixtures/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	if n != 6 {
		t.Error("attendu 6 enquetes, obtenu", n)
	}

	for _, langage := range []string{"go", "javascript", "sql"} {
		liste, _ := List(db, langage, "")
		if len(liste) != 2 {
			t.Error(langage, ": attendu 2 enquetes, obtenu", len(liste))
		}
	}

	LoadFixtures(db, "../../fixtures/cases.json")
	toutes, _ := List(db, "", "")
	if len(toutes) != 6 {
		t.Error("apres rechargement : attendu 6, obtenu", len(toutes))
	}
}

func baseAvecEnquetes(t *testing.T) *sql.DB {
	chemin := filepath.Join(t.TempDir(), "test.db")
	db, err := storage.Open(chemin, "../../migrations/001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = LoadFixtures(db, "../../fixtures/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAffaireDuJourToujoursLaMeme(t *testing.T) {
	db := baseAvecEnquetes(t)
	jour := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)

	premier, err := DuJour(db, jour)
	if err != nil {
		t.Fatal(err)
	}
	// meme date mais en fin de journee : ca doit donner la meme affaire
	second, err := DuJour(db, jour.Add(23*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if premier.Slug != second.Slug {
		t.Error("l'affaire du jour a change dans la journee :", premier.Slug, second.Slug)
	}
	if premier.Date != "2026-09-04" {
		t.Error("date attendue 2026-09-04, obtenue", premier.Date)
	}
	if premier.Numero < 1 || premier.Numero > 6 {
		t.Error("numero hors des 6 enquetes :", premier.Numero)
	}
}

func TestAffaireDuJourChangeChaqueJour(t *testing.T) {
	db := baseAvecEnquetes(t)
	jour := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)

	vus := map[string]bool{}
	for i := 0; i < 6; i++ {
		q, err := DuJour(db, jour.AddDate(0, 0, i))
		if err != nil {
			t.Fatal(err)
		}
		if vus[q.Slug] {
			t.Error("l'enquete", q.Slug, "revient avant d'avoir fait le tour")
		}
		vus[q.Slug] = true
	}
	if len(vus) != 6 {
		t.Error("attendu 6 enquetes differentes en 6 jours, obtenu", len(vus))
	}
}
