# BugHunter AI

Projet de stage EDUNAI (juillet – septembre 2026), B1 Ynov.

BugHunter AI est un jeu web où chaque bug est une enquête. On ouvre un
dossier, on lit le rapport d'incident (ce qui devait se passer, ce qui
s'est passé), on examine un petit bout de code et des pièces à
conviction, et on doit trouver le suspect coupable parmi trois causes
possibles. Ensuite on choisit la correction, et on reçoit un score et un
rapport qui explique le bug.

L'idée du sujet, c'est d'apprendre à raisonner devant un bug plutôt que
de copier une correction. Donc la solution n'est jamais donnée d'avance,
les indices coûtent des points, et l'IA (OpenRouter) sert d'assistant
qui reformule les indices sans révéler la réponse.

Stack : Go pour le serveur, SQLite pour la base, HTML/CSS/JS sans
framework pour l'interface, OpenRouter pour l'assistant.

## Lancer le projet

Il faut Go 1.22 minimum, à cause des routes du type `/api/cases/{slug}`
qui n'existent pas avant.

    go mod tidy
    copy .env.example .env      (cp sur mac / linux)
    go run ./cmd/server

Puis ouvrir http://localhost:8080. La base `bughunter.db` se crée toute
seule au premier lancement, et les six enquêtes sont chargées depuis
`fixtures/cases.json`. Pour repartir de zéro il suffit de supprimer le
fichier `.db`.

Le `.env` contient quatre variables, toutes facultatives :

    PORT                 port d'écoute, 8080 par défaut
    DB_PATH              fichier SQLite, bughunter.db par défaut
    OPENROUTER_API_KEY   la clé OpenRouter. Vide = le jeu tourne sans IA
    OPENROUTER_MODEL     le modèle, openrouter/auto par défaut

Attention pour le modèle : `openrouter/auto` route vers des modèles
payants, donc sans crédit sur le compte on a une erreur 402. J'ai
utilisé des modèles gratuits pendant le stage (ceux qui finissent par
`:free`), mais leurs identifiants changent régulièrement, un modèle qui
marchait le 7 septembre renvoyait 404 le 9. Avant une démo, vérifier sur
https://openrouter.ai/models qu'il existe encore. Au moment où j'écris,
`nex-agi/nex-n2.5-mini:free` fonctionne.

La clé ne doit jamais aller dans Git : `.env` est dans le `.gitignore`,
et `.env.example` ne contient que les noms.

## Comment on joue

1. On crée un profil (ou on se connecte), ça ouvre le bureau des enquêtes.
2. Le bureau montre les six dossiers avec leur langage, leur niveau et
   leur statut (à découvrir, en cours, classée). On peut filtrer par
   langage et par difficulté.
3. On ouvre un dossier : rapport d'incident, code, pièces à conviction,
   les trois suspects, et un chrono qui démarre.
4. On teste une hypothèse en cliquant sur un suspect. Si c'est la
   mauvaise, on perd des points mais on peut réessayer.
5. On peut demander jusqu'à trois indices, du plus vague au plus précis.
   Chacun coûte des points.
6. Quand on a le bon suspect, on choisit la correction. Si c'est la bonne,
   l'affaire est classée et le rapport final s'affiche.
7. Le profil garde l'historique, le score total, le meilleur score et les
   concepts déjà travaillés.

Il y a deux enquêtes par langage (Go, JavaScript, SQL) et deux par niveau
(découverte, investigation, expert). Les niveaux correspondent à la
difficulté du raisonnement : en découverte le message d'erreur désigne
presque la ligne, en expert rien ne plante et le piège est silencieux
(la comparaison `= NULL` en SQL, le script chargé avant le bouton en JS).

## Le score

C'est le serveur qui calcule, jamais le navigateur. Il part de ce qui est
enregistré en base (les tentatives et les indices utilisés), donc on ne
peut pas tricher en modifiant la page.

    départ                                 100
    mauvaise hypothèse                     -10 à chaque fois
    premier indice                          -5
    deuxième indice                        -10
    troisième indice                       -15
    bonne correction du premier coup       +10
    résolu en moins de 3 minutes           +10
    résolu en moins de 6 minutes            +5

Le total est borné entre 0 et 120. Le sujet donnait 110, j'ai ajouté 10
pour le bonus de rapidité (c'est le mode chronométré, une des extensions
proposées). Le temps est mesuré entre l'ouverture du dossier et le
verdict, en base, par le serveur. Le compteur qu'on voit à l'écran ne
fait qu'afficher. Si on ferme l'onglet et qu'on revient plus tard, le
chrono reprend où il en était, pas à zéro.

## L'assistant-détective

Quand une clé OpenRouter est configurée, deux choses passent par l'IA.

Les indices : l'indice écrit dans le dossier est envoyé à l'IA avec le
langage, le concept, le niveau d'indice et la dernière fausse piste du
joueur, et elle le reformule dans le ton d'un détective. On ne lui envoie
que ça, jamais de données du compte. La réponse est vérifiée avant
d'être affichée : vide, trop longue, ou contenant la bonne correction,
elle est rejetée et c'est l'indice d'origine qui part.

Le rapport final : l'IA rend un retour en quatre points, demandé en JSON
(ce que le joueur a bien observé, son erreur de raisonnement s'il y en a
eu, le concept à retenir, une courte question pour vérifier). Chaque
champ obligatoire est contrôlé ; s'il en manque un ou si la réponse
n'est pas du JSON, le rapport local s'affiche seul.

Dans tous les cas où l'IA ne répond pas (pas de clé, erreur, plus de 8
secondes), le jeu continue avec le contenu du dossier et le joueur voit
un message qui le dit, sans le détail technique. Le détail est dans les
logs du serveur. Pour simuler la panne en démo : retirer la clé du
`.env` et relancer.

L'IA ne décide jamais si une réponse est bonne, ne calcule jamais le
score et ne voit jamais de code envoyé par le joueur (il n'y en a pas,
le code des enquêtes est affiché comme du texte).

## L'affaire du jour

Une enquête est mise en avant chaque jour sur le bureau. Elle est choisie
à partir de la date (nombre de jours depuis 1970, modulo six), donc c'est
la même pour tout le monde, elle ne change pas quand on recharge, et les
six dossiers défilent chacun leur tour. Pas de tirage au sort, pas de
table en plus.

## Les routes

    POST  /api/register                        créer un profil
    POST  /api/login                           se connecter (cookie HttpOnly)
    POST  /api/logout                          se déconnecter
    GET   /api/cases                           la liste, filtres ?language= et ?difficulty=
    GET   /api/cases/daily                     l'affaire du jour
    GET   /api/cases/{slug}                    un dossier, sans la solution
    POST  /api/investigations                  commencer (ou reprendre) une enquête
    POST  /api/investigations/{id}/suspects    tester une hypothèse
    POST  /api/investigations/{id}/hints       le prochain indice (3 max)
    POST  /api/investigations/{id}/verdict     la cause + la correction
    GET   /api/investigations/{id}/report      le rapport, seulement une fois résolu
    GET   /api/progress                        historique, scores, concepts

Toutes les routes sauf les trois premières demandent la session. Aucune
ne renvoie `correct_suspect_id`, les indices, la leçon ou la bonne
correction avant la résolution : c'est la structure `PublicCase` qui
garantit ça côté Go, et un test le vérifie sur les six dossiers.

Petite chose à savoir avec Go 1.22 : `/api/cases/daily` et
`/api/cases/{slug}` cohabitent parce que le routeur préfère le motif
littéral, donc `daily` n'est jamais pris pour un slug.

## Organisation du code

    cmd/server/main.go        lit le .env, ouvre la base, charge les enquêtes, démarre
    internal/storage          ouverture SQLite et migration
    internal/auth             comptes (bcrypt) et sessions en mémoire
    internal/cases            le format JSON d'une enquête, sa validation, l'affaire du jour
    internal/game             les règles : hypothèses, indices, verdict, score, rapport
    internal/ai               le client OpenRouter et la validation de ses réponses
    internal/httpapi          les routes et le middleware de session
    web/                      index.html, css/style.css, js/app.js
    fixtures/cases.json       les six enquêtes
    migrations/001_init.sql   les six tables

La base a six tables : `users`, `cases`, `investigations`, `attempts`,
`hints_used`, `reports`. Les tentatives ne sont jamais modifiées ni
supprimées, on ne fait qu'insérer. Une contrainte `UNIQUE` empêche de
débloquer deux fois le même indice.

## Tests

    go test ./...

64 tests, avec `testing` et `httptest`, sans bibliothèque en plus. Ils
couvrent les règles du jeu (le score, l'ordre des indices, pas de
deuxième verdict, le rapport inaccessible avant résolution, la solution
jamais exposée), la validation de l'IA (réponse vide, trop longue, non
JSON, champ manquant, solution révélée, délai dépassé, avec un faux
OpenRouter local), l'affaire du jour et le chrono.

