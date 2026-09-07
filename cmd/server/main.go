package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strings"

	"bughunter-ai/internal/ai"
	"bughunter-ai/internal/cases"
	"bughunter-ai/internal/httpapi"
	"bughunter-ai/internal/storage"
)

func chargerEnv(chemin string) {
	fichier, err := os.Open(chemin)
	if err != nil {
		return
	}
	defer fichier.Close()

	scanner := bufio.NewScanner(fichier)
	for scanner.Scan() {
		ligne := strings.TrimSpace(scanner.Text())
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			continue
		}
		nom, valeur, ok := strings.Cut(ligne, "=")
		if !ok {
			continue
		}
		if os.Getenv(nom) == "" {
			os.Setenv(nom, strings.TrimSpace(valeur))
		}
	}
}

func valeurOuDefaut(nom string, parDefaut string) string {
	v := os.Getenv(nom)
	if v == "" {
		return parDefaut
	}
	return v
}

func main() {
	chargerEnv(".env")
	port := valeurOuDefaut("PORT", "8080")
	cheminDB := valeurOuDefaut("DB_PATH", "bughunter.db")

	db, err := storage.Open(cheminDB, "migrations/001_init.sql")
	if err != nil {
		log.Fatal("base de donnees : ", err)
	}
	defer db.Close()

	n, err := cases.LoadFixtures(db, "fixtures/cases.json")
	if err != nil {
		log.Fatal("enquetes : ", err)
	}
	log.Println(n, "enquetes chargees")

	// le client OpenRouter : sans cle, l'IA reste inactive et le jeu
	// fonctionne avec le contenu local (mode de secours, section 12)
	clientIA := ai.New(os.Getenv("OPENROUTER_API_KEY"),
		valeurOuDefaut("OPENROUTER_MODEL", "openrouter/auto"))
	if clientIA.Actif() {
		log.Println("assistant-detective : OpenRouter actif (" + clientIA.Modele + ")")
	} else {
		log.Println("assistant-detective : pas de cle, mode local")
	}

	api := &httpapi.Server{DB: db, AI: clientIA}
	mux := http.NewServeMux()
	mux.Handle("/api/", api.Routes())
	mux.Handle("/healthz", api.Routes())
	mux.Handle("/", http.FileServer(http.Dir("web")))

	log.Println("serveur sur http://localhost:" + port)
	err = http.ListenAndServe(":"+port, mux)
	if err != nil {
		log.Fatal(err)
	}
}
