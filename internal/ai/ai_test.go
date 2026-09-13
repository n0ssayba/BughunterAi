package ai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fauxOpenRouter simule l'API : il repond toujours le texte donne.
func fauxOpenRouter(t *testing.T, texte string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("la cle n'a pas ete envoyee dans l'en-tete Authorization")
		}
		w.Header().Set("Content-Type", "application/json")
		reponse := `{"choices":[{"message":{"content":` + texte + `}}]}`
		w.Write([]byte(reponse))
	}))
}

func clientDeTest(url string) *Client {
	c := New("cle-de-test", "modele-de-test")
	c.URL = url
	return c
}

func TestSansCleInactif(t *testing.T) {
	// regle du sujet (12) : sans cle, l'IA est simplement inactive
	c := New("", "modele")
	if c.Actif() {
		t.Error("un client sans cle ne doit pas etre actif")
	}
}

func TestReformulationValide(t *testing.T) {
	serveur := fauxOpenRouter(t, `"Observe bien le type de chaque valeur, detective."`)
	defer serveur.Close()

	texte, err := clientDeTest(serveur.URL).ReformulerIndice(context.Background(), Indice{
		Titre: "Le total mysterieux", Texte: "Observe le type de chaque valeur.",
		Spoiler: "Number(a) + Number(b)"})
	if err != nil {
		t.Fatal(err)
	}
	if texte == "" {
		t.Error("la reformulation est vide")
	}
}

func TestReponseQuiReveleLaSolution(t *testing.T) {
	// regle du sujet (15.2) : une reponse d'IA invalide est rejetee
	serveur := fauxOpenRouter(t, `"La solution est d'ecrire Number(a) + Number(b) !"`)
	defer serveur.Close()

	_, err := clientDeTest(serveur.URL).ReformulerIndice(context.Background(), Indice{
		Titre: "Le total mysterieux", Texte: "Observe le type de chaque valeur.",
		Spoiler: "Number(a) + Number(b)"})
	if err == nil {
		t.Error("une reponse qui revele la correction a ete acceptee")
	}
}

func TestReponseVideRejetee(t *testing.T) {
	serveur := fauxOpenRouter(t, `""`)
	defer serveur.Close()

	_, err := clientDeTest(serveur.URL).ReformulerIndice(context.Background(), Indice{Titre: "Titre", Texte: "Indice."})
	if err == nil {
		t.Error("une reponse vide a ete acceptee")
	}
}

func TestTimeout(t *testing.T) {
	// regle du sujet (15.2) : un delai depasse declenche le repli
	serveur := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer serveur.Close()

	ctx, annuler := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer annuler()
	_, err := clientDeTest(serveur.URL).ReformulerIndice(ctx, Indice{Titre: "Titre", Texte: "Indice."})
	if err == nil {
		t.Error("l'appel aurait du echouer par timeout")
	}
}

func TestCommentaireDeRapport(t *testing.T) {
	// regle du sujet (15.2) : une reponse structuree correcte est acceptee
	rapportJSON := `{"observation":"Tu as vu que typeof renvoyait string.",` +
		`"erreur":"","concept":"Convertir avant d'additionner.",` +
		`"question":"Que donne '2' + 2 ?"}`
	serveur := fauxOpenRouter(t, strconv.Quote(rapportJSON))
	defer serveur.Close()

	r, err := clientDeTest(serveur.URL).CommenterRapport(context.Background(),
		"Le total mysterieux", "Les valeurs d'un formulaire sont des chaines.", 0, 1, 105)
	if err != nil {
		t.Fatal(err)
	}
	if r.Observation == "" || r.Concept == "" || r.Question == "" {
		t.Error("un champ du retour est vide :", r)
	}
	if r.Erreur != "" {
		t.Error("sans fausse piste, erreur doit rester vide, obtenu", r.Erreur)
	}
}

func TestCommentaireEntoureDeBalises(t *testing.T) {
	// certains modeles renvoient ```json ... ``` malgre la consigne
	rapportJSON := "```json\n" + `{"observation":"a","erreur":"","concept":"b","question":"c"}` + "\n```"
	serveur := fauxOpenRouter(t, strconv.Quote(rapportJSON))
	defer serveur.Close()

	_, err := clientDeTest(serveur.URL).CommenterRapport(context.Background(), "T", "L", 0, 0, 100)
	if err != nil {
		t.Error("le JSON entoure de balises a ete refuse :", err)
	}
}

func TestCommentaireSansChampObligatoire(t *testing.T) {
	// regle du sujet (15.2) : une reponse sans champ obligatoire est refusee
	sansQuestion := `{"observation":"a","erreur":"","concept":"b"}`
	serveur := fauxOpenRouter(t, strconv.Quote(sansQuestion))
	defer serveur.Close()

	_, err := clientDeTest(serveur.URL).CommenterRapport(context.Background(), "T", "L", 0, 0, 100)
	if err == nil {
		t.Error("un retour sans question de verification a ete accepte")
	}
}

func TestCommentaireNonJSON(t *testing.T) {
	// regle du sujet (15.2) : une reponse non JSON declenche le repli
	serveur := fauxOpenRouter(t, `"Bravo detective, belle enquete !"`)
	defer serveur.Close()

	_, err := clientDeTest(serveur.URL).CommenterRapport(context.Background(), "T", "L", 0, 0, 100)
	if err == nil {
		t.Error("un retour en texte libre a ete accepte a la place du JSON")
	}
}

func TestContexteEnvoyePourUnIndice(t *testing.T) {
	// regle du sujet (7.1) : on envoie langage, concept, hypothese choisie
	// et niveau d'indice
	var recu string
	serveur := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corps, _ := io.ReadAll(r.Body)
		recu = string(corps)
		w.Write([]byte(`{"choices":[{"message":{"content":"Regarde bien, detective."}}]}`))
	}))
	defer serveur.Close()

	_, err := clientDeTest(serveur.URL).ReformulerIndice(context.Background(), Indice{
		Titre: "Le total mysterieux", Langage: "javascript", Concept: "string-number-conversion",
		Niveau: 2, Hypothese: "console.log modifie les valeurs", Texte: "Observe le type.",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, attendu := range []string{"javascript", "string-number-conversion", "console.log modifie", "Niveau d'indice : 2"} {
		if !strings.Contains(recu, attendu) {
			t.Error("le contexte envoye a l'IA ne contient pas :", attendu)
		}
	}
}
