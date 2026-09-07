package game

import (
	"database/sql"
	"encoding/json"
	"errors"

	"bughunter-ai/internal/cases"
)

var ErrIntrouvable = errors.New("investigation introuvable")
var ErrDejaResolue = errors.New("cette enquete est deja resolue")
var ErrPlusDIndices = errors.New("les trois indices sont deja debloques")
var ErrSuspectInconnu = errors.New("ce suspect n'existe pas dans cette enquete")
var ErrFixInconnue = errors.New("cette correction n'existe pas dans cette enquete")
var ErrPasResolue = errors.New("le rapport n'est disponible qu'apres la resolution")

func chargerEnquete(db *sql.DB, userID int64, invID int64) (cases.Case, string, error) {
	var contentJSON, status string
	// le user_id dans le WHERE : impossible de jouer l'enquete d'un autre
	err := db.QueryRow(`SELECT c.content_json, i.status
		FROM investigations i
		JOIN cases c ON c.id = i.case_id
		WHERE i.id = ? AND i.user_id = ?`, invID, userID).Scan(&contentJSON, &status)
	if err == sql.ErrNoRows {
		return cases.Case{}, "", ErrIntrouvable
	}
	if err != nil {
		return cases.Case{}, "", err
	}

	var c cases.Case
	err = json.Unmarshal([]byte(contentJSON), &c)
	return c, status, err
}

// CaseOf retourne l'enquete d'une investigation du joueur.
// Utilisee par les routes pour construire les demandes a l'IA.
func CaseOf(db *sql.DB, userID int64, invID int64) (cases.Case, string, error) {
	return chargerEnquete(db, userID, invID)
}

func Start(db *sql.DB, userID int64, slug string) (int64, bool, error) {
	var caseID int64
	err := db.QueryRow("SELECT id FROM cases WHERE slug = ?", slug).Scan(&caseID)
	if err == sql.ErrNoRows {
		return 0, false, cases.ErrIntrouvable
	}
	if err != nil {
		return 0, false, err
	}

	var id int64
	err = db.QueryRow(`SELECT id FROM investigations
		WHERE user_id = ? AND case_id = ? AND status = 'started'`,
		userID, caseID).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if err != sql.ErrNoRows {
		return 0, false, err
	}

	resultat, err := db.Exec(
		"INSERT INTO investigations (user_id, case_id) VALUES (?, ?)",
		userID, caseID)
	if err != nil {
		return 0, false, err
	}
	id, err = resultat.LastInsertId()
	return id, false, err
}

func RecordSuspect(db *sql.DB, userID int64, invID int64, suspectID string) (bool, error) {
	c, status, err := chargerEnquete(db, userID, invID)
	if err != nil {
		return false, err
	}
	if status == "solved" {
		return false, ErrDejaResolue
	}

	existe := false
	for _, s := range c.Suspects {
		if s.ID == suspectID {
			existe = true
		}
	}
	if !existe {
		return false, ErrSuspectInconnu
	}

	correct := suspectID == c.CorrectSuspectID
	_, err = db.Exec(
		"INSERT INTO attempts (investigation_id, suspect_id, is_correct) VALUES (?, ?, ?)",
		invID, suspectID, correct)
	return correct, err
}

func UnlockHint(db *sql.DB, userID int64, invID int64) (int, string, error) {
	c, status, err := chargerEnquete(db, userID, invID)
	if err != nil {
		return 0, "", err
	}
	if status == "solved" {
		return 0, "", ErrDejaResolue
	}

	var deja int
	err = db.QueryRow("SELECT COUNT(*) FROM hints_used WHERE investigation_id = ?",
		invID).Scan(&deja)
	if err != nil {
		return 0, "", err
	}
	if deja >= 3 {
		return 0, "", ErrPlusDIndices
	}

	niveau := deja + 1
	_, err = db.Exec(
		"INSERT INTO hints_used (investigation_id, hint_level) VALUES (?, ?)",
		invID, niveau)
	if err != nil {
		return 0, "", err
	}
	return niveau, c.Hints[niveau-1], nil
}

type Rapport struct {
	Score               int    `json:"score"`
	CorrectSuspectID    string `json:"correct_suspect_id"`
	CorrectionCorrecte  string `json:"correction_correcte"`
	Lesson              string `json:"lesson"`
	MauvaisesHypotheses int    `json:"mauvaises_hypotheses"`
	IndicesUtilises     int    `json:"indices_utilises"`
	BonusPremierEssai   bool   `json:"bonus_premier_essai"`
	// rempli par l'assistant-detective quand OpenRouter est configure,
	// vide sinon : le rapport local reste complet (mode de secours F12)
	Commentaire string `json:"commentaire"`
}

func Verdict(db *sql.DB, userID int64, invID int64, suspectID string, fixID string) (bool, int, error) {
	c, status, err := chargerEnquete(db, userID, invID)
	if err != nil {
		return false, 0, err
	}
	if status == "solved" {
		return false, 0, ErrDejaResolue
	}

	var fixChoisie cases.Fix
	trouve := false
	for _, f := range c.Fixes {
		if f.ID == fixID {
			fixChoisie = f
			trouve = true
		}
	}
	if !trouve {
		return false, 0, ErrFixInconnue
	}

	correct := suspectID == c.CorrectSuspectID
	_, err = db.Exec(
		"INSERT INTO attempts (investigation_id, suspect_id, is_correct) VALUES (?, ?, ?)",
		invID, suspectID, correct)
	if err != nil {
		return false, 0, err
	}

	if !correct {
		return false, 0, nil
	}

	// le score se calcule depuis ce qui est enregistre en base
	var mauvaises, indices int
	db.QueryRow("SELECT COUNT(*) FROM attempts WHERE investigation_id = ? AND is_correct = 0",
		invID).Scan(&mauvaises)
	db.QueryRow("SELECT COUNT(*) FROM hints_used WHERE investigation_id = ?",
		invID).Scan(&indices)

	score := Score(mauvaises, indices, fixChoisie.Correct)

	_, err = db.Exec(`UPDATE investigations
		SET status = 'solved', score = ?, solved_at = datetime('now')
		WHERE id = ?`, score, invID)
	if err != nil {
		return false, 0, err
	}

	labelCorrect := ""
	for _, f := range c.Fixes {
		if f.Correct {
			labelCorrect = f.Label
		}
	}

	rapport := Rapport{
		Score:               score,
		CorrectSuspectID:    c.CorrectSuspectID,
		CorrectionCorrecte:  labelCorrect,
		Lesson:              c.Lesson,
		MauvaisesHypotheses: mauvaises,
		IndicesUtilises:     indices,
		BonusPremierEssai:   fixChoisie.Correct,
	}
	contentJSON, err := json.Marshal(rapport)
	if err != nil {
		return false, 0, err
	}
	_, err = db.Exec(
		"INSERT INTO reports (investigation_id, content_json) VALUES (?, ?)",
		invID, string(contentJSON))
	return true, score, err
}

// AjouterCommentaire complete le rapport avec le retour de l'IA.
func AjouterCommentaire(db *sql.DB, invID int64, commentaire string) error {
	var contentJSON string
	err := db.QueryRow("SELECT content_json FROM reports WHERE investigation_id = ?",
		invID).Scan(&contentJSON)
	if err != nil {
		return err
	}
	var rapport Rapport
	err = json.Unmarshal([]byte(contentJSON), &rapport)
	if err != nil {
		return err
	}
	rapport.Commentaire = commentaire
	nouveau, err := json.Marshal(rapport)
	if err != nil {
		return err
	}
	_, err = db.Exec("UPDATE reports SET content_json = ? WHERE investigation_id = ?",
		string(nouveau), invID)
	return err
}

func GetReport(db *sql.DB, userID int64, invID int64) (json.RawMessage, error) {
	_, status, err := chargerEnquete(db, userID, invID)
	if err != nil {
		return nil, err
	}
	if status != "solved" {
		return nil, ErrPasResolue
	}

	var contentJSON string
	err = db.QueryRow("SELECT content_json FROM reports WHERE investigation_id = ?",
		invID).Scan(&contentJSON)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(contentJSON), nil
}

type Ligne struct {
	InvestigationID int64  `json:"investigation_id"`
	Slug            string `json:"slug"`
	Title           string `json:"title"`
	Language        string `json:"language"`
	Concept         string `json:"concept"`
	Status          string `json:"status"`
	Score           int    `json:"score"`
	StartedAt       string `json:"started_at"`
}

type Progression struct {
	ScoreTotal       int      `json:"score_total"`
	EnquetesResolues int      `json:"enquetes_resolues"`
	MeilleurScore    int      `json:"meilleur_score"`
	Concepts         []string `json:"concepts"`
	Historique       []Ligne  `json:"historique"`
}

func Progress(db *sql.DB, userID int64) (Progression, error) {
	lignes, err := db.Query(`SELECT i.id, c.slug, c.content_json, c.language,
			i.status, COALESCE(i.score, 0), i.started_at
		FROM investigations i
		JOIN cases c ON c.id = i.case_id
		WHERE i.user_id = ?
		ORDER BY i.started_at DESC, i.id DESC`, userID)
	if err != nil {
		return Progression{}, err
	}
	defer lignes.Close()

	p := Progression{Historique: []Ligne{}, Concepts: []string{}}
	dejaVus := map[string]bool{}
	for lignes.Next() {
		var l Ligne
		var contentJSON string
		err = lignes.Scan(&l.InvestigationID, &l.Slug, &contentJSON,
			&l.Language, &l.Status, &l.Score, &l.StartedAt)
		if err != nil {
			return Progression{}, err
		}
		var c cases.Case
		json.Unmarshal([]byte(contentJSON), &c)
		l.Title = c.Title
		l.Concept = c.Concept

		if l.Status == "solved" {
			p.ScoreTotal = p.ScoreTotal + l.Score
			p.EnquetesResolues = p.EnquetesResolues + 1
			if l.Score > p.MeilleurScore {
				p.MeilleurScore = l.Score
			}
			// les concepts travailles (F10), une seule fois chacun
			if c.Concept != "" && !dejaVus[c.Concept] {
				dejaVus[c.Concept] = true
				p.Concepts = append(p.Concepts, c.Concept)
			}
		}
		p.Historique = append(p.Historique, l)
	}
	return p, nil
}
