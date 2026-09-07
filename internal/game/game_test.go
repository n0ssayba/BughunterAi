package game

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"bughunter-ai/internal/cases"
	"bughunter-ai/internal/storage"
)

func baseDeTest(t *testing.T) *sql.DB {
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
	db.Exec("INSERT INTO users (username, password_hash) VALUES ('test', 'x')")
	return db
}

func TestStartPuisReprise(t *testing.T) {
	db := baseDeTest(t)

	id1, reprise, err := Start(db, 1, "js-total-mysterieux")
	if err != nil {
		t.Fatal(err)
	}
	if reprise {
		t.Error("la premiere fois ne doit pas etre une reprise")
	}

	id2, reprise, err := Start(db, 1, "js-total-mysterieux")
	if err != nil {
		t.Fatal(err)
	}
	if !reprise || id2 != id1 {
		t.Error("attendu la reprise de l'investigation", id1, ", obtenu", id2)
	}
}

func TestStartDossierInconnu(t *testing.T) {
	db := baseDeTest(t)
	_, _, err := Start(db, 1, "n-existe-pas")
	if err != cases.ErrIntrouvable {
		t.Error("attendu ErrIntrouvable, obtenu", err)
	}
}

func TestHypotheseBonneEtMauvaise(t *testing.T) {
	db := baseDeTest(t)
	id, _, _ := Start(db, 1, "js-total-mysterieux")

	correct, err := RecordSuspect(db, 1, id, "B")
	if err != nil {
		t.Fatal(err)
	}
	if correct {
		t.Error("B n'est pas la bonne cause de cette enquete")
	}

	correct, err = RecordSuspect(db, 1, id, "A")
	if err != nil {
		t.Fatal(err)
	}
	if !correct {
		t.Error("A est la bonne cause de cette enquete")
	}
}

func TestHypotheseDUnAutreJoueur(t *testing.T) {
	db := baseDeTest(t)
	db.Exec("INSERT INTO users (username, password_hash) VALUES ('autre', 'x')")
	id, _, _ := Start(db, 1, "js-total-mysterieux")

	_, err := RecordSuspect(db, 2, id, "A")
	if err != ErrIntrouvable {
		t.Error("attendu ErrIntrouvable, obtenu", err)
	}
}

func TestIndicesDansLOrdre(t *testing.T) {
	db := baseDeTest(t)
	id, _, _ := Start(db, 1, "js-total-mysterieux")

	for attendu := 1; attendu <= 3; attendu++ {
		niveau, texte, err := UnlockHint(db, 1, id)
		if err != nil {
			t.Fatal(err)
		}
		if niveau != attendu {
			t.Error("attendu le niveau", attendu, ", obtenu", niveau)
		}
		if texte == "" {
			t.Error("l'indice", niveau, "est vide")
		}
	}

	_, _, err := UnlockHint(db, 1, id)
	if err != ErrPlusDIndices {
		t.Error("attendu ErrPlusDIndices, obtenu", err)
	}
}

func TestVerdictScoreComplet(t *testing.T) {
	db := baseDeTest(t)
	id, _, _ := Start(db, 1, "js-total-mysterieux")

	RecordSuspect(db, 1, id, "B")
	UnlockHint(db, 1, id)
	UnlockHint(db, 1, id)

	resolu, score, err := Verdict(db, 1, id, "A", "F1")
	if err != nil {
		t.Fatal(err)
	}
	if !resolu {
		t.Fatal("le verdict avec la bonne cause doit resoudre l'enquete")
	}
	if score != 85 {
		t.Error("attendu 85, obtenu", score)
	}
}

func TestVerdictMauvaiseCause(t *testing.T) {
	db := baseDeTest(t)
	id, _, _ := Start(db, 1, "js-total-mysterieux")

	resolu, _, err := Verdict(db, 1, id, "C", "F1")
	if err != nil {
		t.Fatal(err)
	}
	if resolu {
		t.Error("une mauvaise cause ne doit pas resoudre l'enquete")
	}

	resolu, score, _ := Verdict(db, 1, id, "A", "F1")
	if !resolu || score != 100 {
		t.Error("attendu 100 (100 - 10 + 10), obtenu", score)
	}
}

func TestPasDeDeuxiemeVerdict(t *testing.T) {
	db := baseDeTest(t)
	id, _, _ := Start(db, 1, "js-total-mysterieux")
	Verdict(db, 1, id, "A", "F1")

	_, _, err := Verdict(db, 1, id, "A", "F1")
	if err != ErrDejaResolue {
		t.Error("attendu ErrDejaResolue, obtenu", err)
	}
}

func TestRapportAvantEtApres(t *testing.T) {
	db := baseDeTest(t)
	id, _, _ := Start(db, 1, "js-total-mysterieux")

	_, err := GetReport(db, 1, id)
	if err != ErrPasResolue {
		t.Error("attendu ErrPasResolue, obtenu", err)
	}

	Verdict(db, 1, id, "A", "F2") // bonne cause, mauvaise correction
	rapport, err := GetReport(db, 1, id)
	if err != nil {
		t.Fatal(err)
	}
	texte := string(rapport)
	if !strings.Contains(texte, "lesson") || !strings.Contains(texte, "correct_suspect_id") {
		t.Error("le rapport ne contient pas la lecon ou la cause")
	}
	if !strings.Contains(texte, `"bonus_premier_essai":false`) {
		t.Error("le bonus ne devrait pas etre accorde avec la mauvaise correction")
	}
}

func TestProgression(t *testing.T) {
	db := baseDeTest(t)

	id1, _, _ := Start(db, 1, "js-total-mysterieux")
	Verdict(db, 1, id1, "A", "F1") // 110 ? non : 100 + 10 = 110
	Start(db, 1, "go-boucle-sans-fin")

	p, err := Progress(db, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Historique) != 2 {
		t.Error("attendu 2 lignes, obtenu", len(p.Historique))
	}
	if p.EnquetesResolues != 1 {
		t.Error("attendu 1 enquete resolue, obtenu", p.EnquetesResolues)
	}
	if p.ScoreTotal != 110 {
		t.Error("attendu un score total de 110, obtenu", p.ScoreTotal)
	}
}

func TestProgressionConceptsEtMeilleurScore(t *testing.T) {
	db := baseDeTest(t)

	// une enquete parfaite, une avec une fausse piste et un indice
	id1, _, _ := Start(db, 1, "js-total-mysterieux")
	Verdict(db, 1, id1, "A", "F1")

	id2, _, _ := Start(db, 1, "go-boucle-sans-fin")
	RecordSuspect(db, 1, id2, "B")
	UnlockHint(db, 1, id2)
	c, _, _ := CaseOf(db, 1, id2)
	Verdict(db, 1, id2, c.CorrectSuspectID, bonneCorrection(c))

	// une enquete commencee mais pas resolue : elle ne compte pas
	Start(db, 1, "sql-dossiers-fantomes")

	p, err := Progress(db, 1)
	if err != nil {
		t.Fatal(err)
	}
	if p.MeilleurScore != 110 {
		t.Error("attendu un meilleur score de 110, obtenu", p.MeilleurScore)
	}
	if len(p.Concepts) != 2 {
		t.Error("attendu 2 concepts travailles, obtenu", p.Concepts)
	}
	for _, l := range p.Historique {
		if l.Concept == "" {
			t.Error("le concept manque sur la ligne", l.Slug)
		}
	}
}

func bonneCorrection(c cases.Case) string {
	for _, f := range c.Fixes {
		if f.Correct {
			return f.ID
		}
	}
	return ""
}
