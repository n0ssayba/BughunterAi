package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
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

	texte, err := clientDeTest(serveur.URL).ReformulerIndice(context.Background(),
		"Le total mysterieux", "Observe le type de chaque valeur.", "Number(a) + Number(b)")
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

	_, err := clientDeTest(serveur.URL).ReformulerIndice(context.Background(),
		"Le total mysterieux", "Observe le type de chaque valeur.", "Number(a) + Number(b)")
	if err == nil {
		t.Error("une reponse qui revele la correction a ete acceptee")
	}
}

func TestReponseVideRejetee(t *testing.T) {
	serveur := fauxOpenRouter(t, `""`)
	defer serveur.Close()

	_, err := clientDeTest(serveur.URL).ReformulerIndice(context.Background(),
		"Titre", "Indice.", "")
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
	_, err := clientDeTest(serveur.URL).ReformulerIndice(ctx, "Titre", "Indice.", "")
	if err == nil {
		t.Error("l'appel aurait du echouer par timeout")
	}
}

func TestCommentaireDeRapport(t *testing.T) {
	serveur := fauxOpenRouter(t, `"Belle enquete, detective : une seule fausse piste avant de coincer le coupable."`)
	defer serveur.Close()

	texte, err := clientDeTest(serveur.URL).CommenterRapport(context.Background(),
		"Le total mysterieux", "Les valeurs d'un formulaire sont des chaines.", 1, 1, 95)
	if err != nil {
		t.Fatal(err)
	}
	if texte == "" {
		t.Error("le commentaire est vide")
	}
}

func TestReponseNonJSONRejetee(t *testing.T) {
	// regle du sujet (15.2) : une reponse qui n'est pas du JSON declenche
	// le repli local
	serveur := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html>page d'erreur du fournisseur</html>"))
	}))
	defer serveur.Close()

	_, err := clientDeTest(serveur.URL).ReformulerIndice(context.Background(),
		"Titre", "Indice.", "")
	if err == nil {
		t.Error("une reponse qui n'est pas du JSON a ete acceptee")
	}
}

func TestIndiceTropLongRejete(t *testing.T) {
	// regle du sujet (15.2) : un indice trop long est refuse
	long := ""
	for i := 0; i < 700; i++ {
		long = long + "a"
	}
	serveur := fauxOpenRouter(t, `"`+long+`"`)
	defer serveur.Close()

	_, err := clientDeTest(serveur.URL).ReformulerIndice(context.Background(),
		"Titre", "Indice.", "")
	if err == nil {
		t.Error("un indice de plus de 600 caracteres a ete accepte")
	}
}
