package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"bughunter-ai/internal/ai"
	"bughunter-ai/internal/cases"
	"bughunter-ai/internal/game"
)

func lireID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func (s *Server) commencerEnquete(w http.ResponseWriter, r *http.Request, userID int64) {
	var corps struct {
		Slug string `json:"slug"`
	}
	err := json.NewDecoder(r.Body).Decode(&corps)
	if err != nil || corps.Slug == "" {
		repondreErreur(w, 400, "corps de requete invalide")
		return
	}

	id, reprise, err := game.Start(s.DB, userID, corps.Slug)
	if err == cases.ErrIntrouvable {
		repondreErreur(w, 404, "enquete introuvable")
		return
	}
	if err != nil {
		repondreErreur(w, 500, "erreur serveur")
		return
	}

	code := 201
	if reprise {
		code = 200 // on reprend une enquete deja en cours
	}
	// en cas de reprise le chrono ne repart pas de zero
	secondes, err := game.SecondesEcoulees(s.DB, id)
	if err != nil {
		repondreErreur(w, 500, "erreur serveur")
		return
	}
	repondreJSON(w, code, map[string]any{
		"investigation_id": id,
		"reprise":          reprise,
		"secondes":         secondes,
	})
}

func (s *Server) enregistrerHypothese(w http.ResponseWriter, r *http.Request, userID int64) {
	invID, err := lireID(r)
	if err != nil {
		repondreErreur(w, 400, "identifiant invalide")
		return
	}
	var corps struct {
		SuspectID string `json:"suspect_id"`
	}
	err = json.NewDecoder(r.Body).Decode(&corps)
	if err != nil || corps.SuspectID == "" {
		repondreErreur(w, 400, "corps de requete invalide")
		return
	}

	correct, err := game.RecordSuspect(s.DB, userID, invID, corps.SuspectID)
	if err == game.ErrIntrouvable {
		repondreErreur(w, 404, "investigation introuvable")
		return
	}
	if err == game.ErrDejaResolue {
		repondreErreur(w, 409, "cette enquete est deja resolue")
		return
	}
	if err == game.ErrSuspectInconnu {
		repondreErreur(w, 400, "ce suspect n'existe pas dans cette enquete")
		return
	}
	if err != nil {
		repondreErreur(w, 500, "erreur serveur")
		return
	}
	repondreJSON(w, 200, map[string]bool{"correct": correct})
}

func (s *Server) debloquerIndice(w http.ResponseWriter, r *http.Request, userID int64) {
	invID, err := lireID(r)
	if err != nil {
		repondreErreur(w, 400, "identifiant invalide")
		return
	}

	niveau, texte, err := game.UnlockHint(s.DB, userID, invID)
	if err == game.ErrIntrouvable {
		repondreErreur(w, 404, "investigation introuvable")
		return
	}
	if err == game.ErrDejaResolue {
		repondreErreur(w, 409, "cette enquete est deja resolue")
		return
	}
	if err == game.ErrPlusDIndices {
		repondreErreur(w, 409, "les trois indices sont deja debloques")
		return
	}
	if err != nil {
		repondreErreur(w, 500, "erreur serveur")
		return
	}
	// l'assistant-detective (F11) reformule l'indice local. Sans cle la
	// source est "locale" ; si l'IA est la mais echoue, c'est "secours" :
	// l'indice local part tel quel et le joueur en est informe (15.3)
	source := "locale"
	if s.AI.Actif() {
		c, _, errC := game.CaseOf(s.DB, userID, invID)
		if errC == nil {
			source = "secours"
			d := ai.Indice{
				Titre:     c.Title,
				Langage:   c.Language,
				Concept:   c.Concept,
				Niveau:    niveau,
				Hypothese: derniereFaussePiste(s.DB, invID, c),
				Texte:     texte,
			}
			for _, f := range c.Fixes {
				if f.Correct {
					d.Spoiler = f.Label
				}
			}
			ctx, annuler := context.WithTimeout(r.Context(), 8*time.Second)
			defer annuler()
			reformule, errIA := s.AI.ReformulerIndice(ctx, d)
			if errIA == nil {
				texte = reformule
				source = "ia"
			} else {
				// rien ne casse pour le joueur, mais on veut savoir pourquoi
				log.Println("assistant-detective (indice) :", errIA)
			}
		}
	}
	repondreJSON(w, 200, map[string]any{"level": niveau, "hint": texte, "source": source})
}

func (s *Server) rendreVerdict(w http.ResponseWriter, r *http.Request, userID int64) {
	invID, err := lireID(r)
	if err != nil {
		repondreErreur(w, 400, "identifiant invalide")
		return
	}
	var corps struct {
		SuspectID string `json:"suspect_id"`
		FixID     string `json:"fix_id"`
	}
	err = json.NewDecoder(r.Body).Decode(&corps)
	if err != nil || corps.SuspectID == "" || corps.FixID == "" {
		repondreErreur(w, 400, "corps de requete invalide")
		return
	}

	resolu, score, err := game.Verdict(s.DB, userID, invID, corps.SuspectID, corps.FixID)
	if err == game.ErrIntrouvable {
		repondreErreur(w, 404, "investigation introuvable")
		return
	}
	if err == game.ErrDejaResolue {
		repondreErreur(w, 409, "cette enquete est deja resolue")
		return
	}
	if err == game.ErrFixInconnue {
		repondreErreur(w, 400, "cette correction n'existe pas dans cette enquete")
		return
	}
	if err != nil {
		repondreErreur(w, 500, "erreur serveur")
		return
	}

	if !resolu {
		repondreJSON(w, 200, map[string]any{"solved": false})
		return
	}
	// le retour personnalise de l'assistant-detective, si l'IA est la ;
	// sinon le rapport local suffit (F12)
	assistant := "inactif"
	if s.AI.Actif() {
		c, _, errC := game.CaseOf(s.DB, userID, invID)
		if errC == nil {
			assistant = "secours"
			var mauvaises, indices int
			s.DB.QueryRow("SELECT COUNT(*) FROM attempts WHERE investigation_id = ? AND is_correct = 0", invID).Scan(&mauvaises)
			s.DB.QueryRow("SELECT COUNT(*) FROM hints_used WHERE investigation_id = ?", invID).Scan(&indices)
			ctx, annuler := context.WithTimeout(r.Context(), 8*time.Second)
			defer annuler()
			retour, errIA := s.AI.CommenterRapport(ctx, c.Title, c.Lesson, mauvaises, indices, score)
			if errIA == nil {
				game.AjouterCommentaire(s.DB, invID, game.Commentaire{
					Observation: retour.Observation,
					Erreur:      retour.Erreur,
					Concept:     retour.Concept,
					Question:    retour.Question,
				})
				assistant = "ia"
			} else {
				log.Println("assistant-detective (rapport) :", errIA)
			}
		}
	}
	repondreJSON(w, 200, map[string]any{"solved": true, "score": score, "assistant": assistant})
}

// la derniere fausse piste du joueur, en clair, pour que l'assistant puisse
// s'y referer (section 7.1). Vide s'il n'y en a pas eu
func derniereFaussePiste(db *sql.DB, invID int64, c cases.Case) string {
	var suspectID string
	err := db.QueryRow(`SELECT suspect_id FROM attempts
		WHERE investigation_id = ? AND is_correct = 0
		ORDER BY id DESC LIMIT 1`, invID).Scan(&suspectID)
	if err != nil {
		return ""
	}
	for _, s := range c.Suspects {
		if s.ID == suspectID {
			return s.Label
		}
	}
	return ""
}

func (s *Server) lireRapport(w http.ResponseWriter, r *http.Request, userID int64) {
	invID, err := lireID(r)
	if err != nil {
		repondreErreur(w, 400, "identifiant invalide")
		return
	}

	rapport, err := game.GetReport(s.DB, userID, invID)
	if err == game.ErrIntrouvable {
		repondreErreur(w, 404, "investigation introuvable")
		return
	}
	if err == game.ErrPasResolue {
		repondreErreur(w, 403, "le rapport n'est disponible qu'apres la resolution")
		return
	}
	if err != nil {
		repondreErreur(w, 500, "erreur serveur")
		return
	}
	repondreJSON(w, 200, rapport)
}

func (s *Server) lireProgression(w http.ResponseWriter, r *http.Request, userID int64) {
	p, err := game.Progress(s.DB, userID)
	if err != nil {
		repondreErreur(w, 500, "erreur serveur")
		return
	}
	repondreJSON(w, 200, p)
}
