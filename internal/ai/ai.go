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

// ReformulerIndice demande a l'assistant-detective de reformuler un
// indice local de facon pedagogique, sans reveler la solution (F11).
// spoiler = le texte de la bonne correction, pour verifier que la
// reponse ne la revele pas.
func (c *Client) ReformulerIndice(ctx context.Context, titre string, indice string, spoiler string) (string, error) {
	consigne := "Tu es l'assistant-detective d'un jeu d'enquetes de debogage pour debutants. " +
		"Reformule l'indice fourni en francais, en une ou deux phrases, de facon pedagogique et " +
		"dans le ton d'un detective. Ne revele jamais la solution ni la correction. " +
		"Reponds uniquement avec l'indice reformule."
	message := "Enquete : " + titre + "\nIndice a reformuler : " + indice

	texte, err := c.demander(ctx, consigne, message)
	if err != nil {
		return "", err
	}
	// validation (section 15.2) : une reponse invalide est rejetee
	if texte == "" || len(texte) > 600 {
		return "", errors.New("reponse invalide")
	}
	if spoiler != "" && strings.Contains(strings.ToLower(texte), strings.ToLower(spoiler)) {
		return "", errors.New("la reponse revele la solution")
	}
	return texte, nil
}

// CommenterRapport genere le retour personnalise du rapport final.
func (c *Client) CommenterRapport(ctx context.Context, titre string, lecon string, mauvaises int, indices int, score int) (string, error) {
	consigne := "Tu es l'assistant-detective d'un jeu d'enquetes de debogage pour debutants. " +
		"Redige en francais un court retour personnalise (2 ou 3 phrases) sur la partie du joueur, " +
		"encourageant et concret, dans le ton d'un detective qui cloture un dossier. " +
		"Reponds uniquement avec ce retour."
	message := "Enquete resolue : " + titre +
		"\nLecon du dossier : " + lecon +
		"\nMauvaises hypotheses : " + strconv.Itoa(mauvaises) +
		"\nIndices utilises : " + strconv.Itoa(indices) +
		"\nScore final : " + strconv.Itoa(score) + " / 110"

	texte, err := c.demander(ctx, consigne, message)
	if err != nil {
		return "", err
	}
	if texte == "" || len(texte) > 800 {
		return "", errors.New("reponse invalide")
	}
	return texte, nil
}
