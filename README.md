# BugHunter AI

Projet de stage EDUNAI 2026. Un jeu web où chaque bug est une enquête
à résoudre : on lit le rapport d'incident, on examine le code et les
pièces à conviction, et on doit trouver le suspect coupable.

Stack : Go, SQLite, HTML/CSS/JS, OpenRouter.

## Pour lancer le projet

Il faut Go 1.22 minimum (à cause des routes du type /api/cases/{slug}).

    go mod tidy
    copy .env.example .env     (cp sur mac/linux)
    go run ./cmd/server

Puis ouvrir http://localhost:8080 dans le navigateur.
La base bughunter.db se crée toute seule au premier lancement.

## L'assistant-détective (OpenRouter)

Avec une clé dans .env (OPENROUTER_API_KEY), les indices sont reformulés
par l'IA et le rapport final reçoit un petit retour personnalisé. Sans
clé, ou si l'API échoue ou met trop longtemps, le jeu continue avec le
contenu local : c'est le mode de secours demandé par le sujet.

Pour simuler une panne : enlever la clé du .env et relancer, tout doit
continuer à marcher (les indices affichent alors la version locale).

## Tests

    go test ./...

## Les routes

- POST /api/register, /api/login, /api/logout
- GET /api/cases (filtres possibles : ?language=go&difficulty=discovery)
- GET /api/cases/daily -> l'affaire du jour, la même pour tout le monde
- GET /api/cases/{slug} -> le dossier sans la solution
- POST /api/investigations -> commencer une enquête
- POST /api/investigations/{id}/suspects -> tester une hypothèse
- POST /api/investigations/{id}/hints -> le prochain indice (3 max),
  reformulé par l'IA quand elle est configurée
- POST /api/investigations/{id}/verdict -> donner la cause + la correction
- GET /api/investigations/{id}/report -> le rapport (seulement après résolution)
- GET /api/progress -> historique et scores

Important : l'API n'envoie jamais la solution (correct_suspect_id, hints,
lesson) avant que l'enquête soit résolue, c'est la règle de la section 11
du sujet.

## Où j'en suis

Semaines 1 à 5 faites : conception, backend, boucle de jeu complète,
interface (thème bureau d'enquête) et l'assistant-détective OpenRouter
avec son repli local. Premier bonus fait : l'affaire du jour, un dossier
mis en avant sur le bureau, choisi à partir de la date (donc le même
pour tout le monde et toute la journée, sans hasard). Reste : la
stabilisation, puis le bonus des badges.
