package ai

// Client OpenRouter (sections 7 et 12 du sujet).
// La cle vient de OPENROUTER_API_KEY, jamais du code.
// Si la cle est absente ou si l'API echoue, le jeu continue avec
// le contenu local : c'est le mode de secours (F12).

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type Client struct {
	Cle    string
	Modele string
	URL    string
}

func New(cle string, modele string) *Client {
	return &Client{
		Cle:    cle,
		Modele: modele,
		URL:    "https://openrouter.ai/api/v1/chat/completions",
	}
}

// Actif dit si l'IA peut etre appelee (une cle est configuree).
func (c *Client) Actif() bool {
	return c != nil && c.Cle != ""
}

// demander envoie un message a l'API et retourne le texte de la reponse.
func (c *Client) demander(ctx context.Context, consigne string, message string) (string, error) {
	corps := map[string]any{
		"model": c.Modele,
		"messages": []map[string]string{
			{"role": "system", "content": consigne},
			{"role": "user", "content": message},
		},
	}
	corpsJSON, err := json.Marshal(corps)
	if err != nil {
		return "", err
	}

	requete, err := http.NewRequestWithContext(ctx, "POST", c.URL, bytes.NewReader(corpsJSON))
	if err != nil {
		return "", err
	}
	requete.Header.Set("Content-Type", "application/json")
	requete.Header.Set("Authorization", "Bearer "+c.Cle)

	reponse, err := http.DefaultClient.Do(requete)
	if err != nil {
		return "", err
	}
	defer reponse.Body.Close()
	if reponse.StatusCode != 200 {
		return "", errors.New("l'API a repondu " + reponse.Status)
	}

	// le format de reponse d'OpenRouter
	var donnees struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	err = json.NewDecoder(reponse.Body).Decode(&donnees)
	if err != nil {
		return "", err
	}
	if len(donnees.Choices) == 0 {
		return "", errors.New("reponse vide")
	}
	return strings.TrimSpace(donnees.Choices[0].Message.Content), nil
}

// le contexte envoye pour reformuler un indice (section 7.1) : langage,
// concept, hypothese choisie et niveau d'indice, rien de plus
type Indice struct {
	Titre     string
	Langage   string
	Concept   string
	Niveau    int
	Hypothese string // la derniere fausse piste du joueur, vide s'il n'y en a pas eu
	Texte     string // l'indice local a reformuler
	Spoiler   string // la bonne correction, pour verifier que la reponse ne la revele pas
}

var nomsNiveaux = map[int]string{1: "observation", 2: "direction", 3: "presque resolu"}

func (c *Client) ReformulerIndice(ctx context.Context, d Indice) (string, error) {
	consigne := "Tu es l'assistant-detective d'un jeu d'enquetes de debogage pour debutants. " +
		"Reformule l'indice fourni en francais, en une ou deux phrases, de facon pedagogique et " +
		"dans le ton d'un detective. Ne revele jamais la solution ni la correction. " +
		"Si une fausse piste est indiquee, aide le joueur a voir pourquoi elle ne tient pas, " +
		"toujours sans donner la bonne reponse. Reponds uniquement avec l'indice reformule."
	message := "Enquete : " + d.Titre +
		"\nLangage : " + d.Langage +
		"\nConcept : " + d.Concept +
		"\nNiveau d'indice : " + strconv.Itoa(d.Niveau) + " (" + nomsNiveaux[d.Niveau] + ")" +
		"\nIndice a reformuler : " + d.Texte
	if d.Hypothese != "" {
		message = message + "\nFausse piste choisie par le joueur : " + d.Hypothese
	}

	texte, err := c.demander(ctx, consigne, message)
	if err != nil {
		return "", err
	}
	// validation (section 15.2) : une reponse invalide est rejetee
	if texte == "" || len(texte) > 600 {
		return "", errors.New("reponse invalide")
	}
	if d.Spoiler != "" && strings.Contains(strings.ToLower(texte), strings.ToLower(d.Spoiler)) {
		return "", errors.New("la reponse revele la solution")
	}
	return texte, nil
}

// le retour personnalise du rapport final, dans la forme demandee par la
// section 7.2. Erreur peut rester vide : un joueur sans fausse piste n'a
// pas fait d'erreur de raisonnement
type Retour struct {
	Observation string `json:"observation"`
	Erreur      string `json:"erreur"`
	Concept     string `json:"concept"`
	Question    string `json:"question"`
}

func (c *Client) CommenterRapport(ctx context.Context, titre string, lecon string, mauvaises int, indices int, score int) (Retour, error) {
	consigne := "Tu es l'assistant-detective d'un jeu d'enquetes de debogage pour debutants. " +
		"Le joueur vient de resoudre une enquete. Redige en francais un retour positif, precis " +
		"et adapte a un debutant. Reponds uniquement avec un objet JSON, sans texte autour, " +
		"avec exactement ces quatre cles : " +
		"\"observation\" (ce que le joueur a correctement observe), " +
		"\"erreur\" (son erreur de raisonnement principale, ou une chaine vide s'il n'en a pas fait), " +
		"\"concept\" (le concept a retenir), " +
		"\"question\" (une question de verification tres courte)."
	message := "Enquete resolue : " + titre +
		"\nLecon du dossier : " + lecon +
		"\nMauvaises hypotheses : " + strconv.Itoa(mauvaises) +
		"\nIndices utilises : " + strconv.Itoa(indices) +
		"\nScore final : " + strconv.Itoa(score) + " / 120"

	texte, err := c.demander(ctx, consigne, message)
	if err != nil {
		return Retour{}, err
	}

	// certains modeles entourent le JSON de ```json ... ``` malgre la consigne
	texte = strings.TrimPrefix(texte, "```json")
	texte = strings.TrimPrefix(texte, "```")
	texte = strings.TrimSuffix(texte, "```")
	texte = strings.TrimSpace(texte)

	var r Retour
	err = json.Unmarshal([]byte(texte), &r)
	if err != nil {
		return Retour{}, errors.New("reponse non JSON")
	}
	// validation (section 15.2) : chaque champ obligatoire doit etre la
	if r.Observation == "" || r.Concept == "" || r.Question == "" {
		return Retour{}, errors.New("champ obligatoire manquant")
	}
	if len(r.Observation) > 500 || len(r.Erreur) > 500 || len(r.Concept) > 500 || len(r.Question) > 200 {
		return Retour{}, errors.New("reponse trop longue")
	}
	return r, nil
}
