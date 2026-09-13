package storage

import (
	"path/filepath"
	"testing"
)

func TestOuvertureEtTables(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(chemin, "../../migrations/001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tables := []string{"users", "cases", "investigations", "attempts", "hints_used", "reports"}
	for _, table := range tables {
		var n int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("la table %s n'a pas ete creee", table)
		}
	}
}

func TestMemeIndiceDeuxFois(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(chemin, "../../migrations/001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.Exec("INSERT INTO users (username, password_hash) VALUES ('test', 'x')")
	db.Exec("INSERT INTO cases (slug, language, difficulty, content_json) VALUES ('c1', 'go', 'discovery', '{}')")
	db.Exec("INSERT INTO investigations (user_id, case_id) VALUES (1, 1)")

	_, err = db.Exec("INSERT INTO hints_used (investigation_id, hint_level) VALUES (1, 1)")
	if err != nil {
		t.Fatal("la premiere insertion devrait passer :", err)
	}
	_, err = db.Exec("INSERT INTO hints_used (investigation_id, hint_level) VALUES (1, 1)")
	if err == nil {
		t.Error("la base a accepte deux fois le meme indice")
	}
}

func TestScoreImpossibleRefuse(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(chemin, "../../migrations/001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.Exec("INSERT INTO users (username, password_hash) VALUES ('test', 'x')")
	db.Exec("INSERT INTO cases (slug, language, difficulty, content_json) VALUES ('c1', 'go', 'discovery', '{}')")

	_, err = db.Exec("INSERT INTO investigations (user_id, case_id, score) VALUES (1, 1, 200)")
	if err == nil {
		t.Error("la base a accepte un score de 200")
	}
}
