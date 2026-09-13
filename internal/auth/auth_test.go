package auth

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"bughunter-ai/internal/storage"
)

func baseDeTest(t *testing.T) *sql.DB {
	chemin := filepath.Join(t.TempDir(), "test.db")
	db, err := storage.Open(chemin, "../../migrations/001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMotDePasseJamaisEnClair(t *testing.T) {
	db := baseDeTest(t)
	err := Register(db, "nossayba", "monmotdepasse")
	if err != nil {
		t.Fatal(err)
	}

	var hash string
	db.QueryRow("SELECT password_hash FROM users WHERE username = 'nossayba'").Scan(&hash)
	if hash == "monmotdepasse" {
		t.Error("le mot de passe est enregistre en clair dans la base")
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Error("le hash ne ressemble pas a un hash bcrypt :", hash)
	}
}

func TestLoginCorrect(t *testing.T) {
	db := baseDeTest(t)
	Register(db, "nossayba", "monmotdepasse")

	jeton, err := Login(db, "nossayba", "monmotdepasse")
	if err != nil {
		t.Fatal(err)
	}
	if jeton == "" {
		t.Error("le jeton de session est vide")
	}
	_, ok := UserID(jeton)
	if !ok {
		t.Error("le jeton retourne par Login n'est pas reconnu")
	}
}

func TestLoginMauvaisMotDePasse(t *testing.T) {
	db := baseDeTest(t)
	Register(db, "nossayba", "monmotdepasse")

	_, err := Login(db, "nossayba", "mauvais")
	if err != ErrIdentifiants {
		t.Error("attendu ErrIdentifiants, obtenu", err)
	}
}

func TestNomDejaPris(t *testing.T) {
	db := baseDeTest(t)
	Register(db, "nossayba", "monmotdepasse")

	err := Register(db, "nossayba", "autremotdepasse")
	if err != ErrNomDejaPris {
		t.Error("attendu ErrNomDejaPris, obtenu", err)
	}
}

func TestLogout(t *testing.T) {
	db := baseDeTest(t)
	Register(db, "nossayba", "monmotdepasse")
	jeton, _ := Login(db, "nossayba", "monmotdepasse")

	Logout(jeton)
	_, ok := UserID(jeton)
	if ok {
		t.Error("le jeton fonctionne encore apres Logout")
	}
}
