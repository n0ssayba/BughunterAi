package cases

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
)

func LoadFixtures(db *sql.DB, chemin string) (int, error) {
	contenu, err := os.ReadFile(chemin)
	if err != nil {
		return 0, err
	}

	var liste []Case
	err = json.Unmarshal(contenu, &liste)
	if err != nil {
		return 0, err
	}

	for _, c := range liste {
		err = Validate(c)
		if err != nil {
			return 0, errors.New("enquete " + c.Slug + " : " + err.Error())
		}

		contentJSON, err := json.Marshal(c)
		if err != nil {
			return 0, err
		}
		// OR IGNORE : si le slug existe deja on ne fait rien
		_, err = db.Exec(
			"INSERT OR IGNORE INTO cases (slug, language, difficulty, content_json, source) VALUES (?, ?, ?, ?, 'local')",
			c.Slug, c.Language, c.Difficulty, string(contentJSON))
		if err != nil {
			return 0, err
		}
	}
	return len(liste), nil
}

type Resume struct {
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Language   string `json:"language"`
	Difficulty string `json:"difficulty"`
}

func List(db *sql.DB, language string, difficulty string) ([]Resume, error) {
	requete := "SELECT slug, content_json FROM cases WHERE 1=1"
	args := []any{}
	if language != "" {
		requete = requete + " AND language = ?"
		args = append(args, language)
	}
	if difficulty != "" {
		requete = requete + " AND difficulty = ?"
		args = append(args, difficulty)
	}
	requete = requete + " ORDER BY id"

	lignes, err := db.Query(requete, args...)
	if err != nil {
		return nil, err
	}
	defer lignes.Close()

	resultats := []Resume{}
	for lignes.Next() {
		var slug, contentJSON string
		err = lignes.Scan(&slug, &contentJSON)
		if err != nil {
			return nil, err
		}
		var c Case
		json.Unmarshal([]byte(contentJSON), &c)
		resultats = append(resultats, Resume{
			Slug: slug, Title: c.Title,
			Language: c.Language, Difficulty: c.Difficulty,
		})
	}
	return resultats, nil
}

var ErrIntrouvable = errors.New("enquete introuvable")

func Get(db *sql.DB, slug string) (Case, error) {
	var contentJSON string
	err := db.QueryRow("SELECT content_json FROM cases WHERE slug = ?", slug).Scan(&contentJSON)
	if err == sql.ErrNoRows {
		return Case{}, ErrIntrouvable
	}
	if err != nil {
		return Case{}, err
	}

	var c Case
	err = json.Unmarshal([]byte(contentJSON), &c)
	return c, err
}
