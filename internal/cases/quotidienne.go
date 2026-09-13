package cases

import (
	"database/sql"
	"time"
)

type Quotidienne struct {
	Date       string `json:"date"`
	Numero     int    `json:"numero"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Language   string `json:"language"`
	Difficulty string `json:"difficulty"`
}

// l'affaire du jour : la meme pour tout le monde et toute la journee, donc
// choisie a partir de la date et pas au hasard
func DuJour(db *sql.DB, jour time.Time) (Quotidienne, error) {
	liste, err := List(db, "", "")
	if err != nil {
		return Quotidienne{}, err
	}
	if len(liste) == 0 {
		return Quotidienne{}, ErrIntrouvable
	}

	// on repart de minuit sinon l'heure et le fuseau font changer le calcul
	// au milieu de la journee
	annee, mois, numeroDuJour := jour.Date()
	minuit := time.Date(annee, mois, numeroDuJour, 0, 0, 0, 0, time.UTC)

	// le nombre de jours depuis 1970 avance de 1 par jour, donc les enquetes
	// passent chacune leur tour
	jours := minuit.Unix() / 86400
	position := int(jours % int64(len(liste)))

	c := liste[position]
	return Quotidienne{
		Date:       minuit.Format("2006-01-02"),
		Numero:     position + 1,
		Slug:       c.Slug,
		Title:      c.Title,
		Language:   c.Language,
		Difficulty: c.Difficulty,
	}, nil
}
