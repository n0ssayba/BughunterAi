package cases

import "errors"

const maxCode = 2000
const maxTexte = 500

func Validate(c Case) error {
	if c.Slug == "" || c.Title == "" || c.Code == "" || c.Lesson == "" {
		return errors.New("un champ obligatoire est vide")
	}
	if c.Incident.Expected == "" || c.Incident.Observed == "" {
		return errors.New("l'incident doit avoir expected et observed")
	}
	if len(c.Evidence) == 0 {
		return errors.New("il faut au moins une preuve")
	}

	if c.Language != "go" && c.Language != "javascript" && c.Language != "sql" {
		return errors.New("langage inconnu : " + c.Language)
	}

	if c.Difficulty != "discovery" && c.Difficulty != "investigation" && c.Difficulty != "expert" {
		return errors.New("difficulte inconnue : " + c.Difficulty)
	}

	if len(c.Suspects) != 3 {
		return errors.New("il faut exactement trois suspects")
	}

	trouve := false
	for _, s := range c.Suspects {
		if s.ID == c.CorrectSuspectID {
			trouve = true
		}
	}
	if !trouve {
		return errors.New("correct_suspect_id ne correspond a aucun suspect")
	}

	if len(c.Hints) != 3 {
		return errors.New("il faut exactement trois indices")
	}

	if len(c.Fixes) < 2 {
		return errors.New("il faut au moins deux corrections")
	}
	uneCorrecte := false
	for _, f := range c.Fixes {
		if f.Correct {
			uneCorrecte = true
		}
	}
	if !uneCorrecte {
		return errors.New("aucune correction n'est marquee correcte")
	}

	if len(c.Code) > maxCode {
		return errors.New("le code est trop long")
	}
	for _, e := range c.Evidence {
		if len(e) > maxTexte {
			return errors.New("une preuve est trop longue")
		}
	}
	for _, h := range c.Hints {
		if h == "" || len(h) > maxTexte {
			return errors.New("un indice est vide ou trop long")
		}
	}
	if len(c.Lesson) > maxTexte {
		return errors.New("la lecon est trop longue")
	}

	return nil
}
