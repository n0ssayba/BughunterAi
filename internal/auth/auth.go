package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

var ErrNomDejaPris = errors.New("ce nom d'utilisateur est deja pris")
var ErrIdentifiants = errors.New("identifiants invalides")

// sessions en memoire (perdues si le serveur redemarre)
var sessions = map[string]int64{}
var verrou sync.Mutex

func HashPassword(motDePasse string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(motDePasse), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPassword(hash string, motDePasse string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(motDePasse)) == nil
}

func Register(db *sql.DB, username string, motDePasse string) error {
	hash, err := HashPassword(motDePasse)
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, hash)
	if err != nil {
		return ErrNomDejaPris
	}
	return nil
}

func Login(db *sql.DB, username string, motDePasse string) (string, error) {
	var id int64
	var hash string
	err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", username).Scan(&id, &hash)
	if err == sql.ErrNoRows {
		return "", ErrIdentifiants
	}
	if err != nil {
		return "", err
	}
	if !CheckPassword(hash, motDePasse) {
		// meme erreur que pour un nom inconnu
		return "", ErrIdentifiants
	}
	return nouvelleSession(id), nil
}

func nouvelleSession(userID int64) string {
	b := make([]byte, 32)
	rand.Read(b)
	jeton := hex.EncodeToString(b)

	verrou.Lock()
	sessions[jeton] = userID
	verrou.Unlock()
	return jeton
}

func UserID(jeton string) (int64, bool) {
	verrou.Lock()
	id, ok := sessions[jeton]
	verrou.Unlock()
	return id, ok
}

func Logout(jeton string) {
	verrou.Lock()
	delete(sessions, jeton)
	verrou.Unlock()
}
