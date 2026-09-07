// interface du jeu

let nomAgent = '';
let investigationId = null;
let enquete = null;
let suspectTrouve = null;
let affaireDuJour = null;
let filtreLangage = '';
let filtreDifficulte = '';

// appel generique vers l'API, le cookie de session part tout seul
async function api(methode, url, corps) {
  const options = { method: methode, headers: { 'Content-Type': 'application/json' } };
  if (corps) {
    options.body = JSON.stringify(corps);
  }
  const reponse = await fetch(url, options);
  let donnees = null;
  try {
    donnees = await reponse.json();
  } catch (e) {
    // pas de JSON dans la reponse
  }
  return { status: reponse.status, donnees: donnees };
}

// affiche un message : erreur, indice, reussite ou info
function message(type, texte, zoneId) {
  const zone = document.getElementById(zoneId || 'message');
  zone.innerHTML = '';
  if (!texte) {
    return;
  }
  const div = document.createElement('div');
  div.className = 'message ' + type;
  div.textContent = texte;
  zone.appendChild(div);
}

// desactive les boutons d'une zone pendant un appel au serveur, pour
// eviter les doubles envois (section 13.2)
function bloquer(zoneId, bloque) {
  const boutons = document.getElementById(zoneId).querySelectorAll('button');
  for (const b of boutons) {
    b.disabled = bloque;
  }
}

/* ----- navigation entre les ecrans ----- */

function montrer(vue) {
  const ecrans = ['accueil', 'connexion', 'inscription', 'bureau', 'dossier', 'profil'];
  for (const e of ecrans) {
    document.getElementById('vue-' + e).className = (e === vue) ? '' : 'cache';
  }
  const connecte = (vue === 'bureau' || vue === 'dossier' || vue === 'profil');
  document.getElementById('bandeau').className = connecte ? '' : 'cache';
  document.getElementById('contenu').className = connecte ? '' : 'cache';

  document.getElementById('nav-bureau').className = (vue === 'bureau' || vue === 'dossier') ? 'actif' : '';
  document.getElementById('nav-profil').className = (vue === 'profil') ? 'actif' : '';

  if (vue === 'bureau') {
    chargerBureau();
  }
  if (vue === 'profil') {
    chargerProfil();
  }
  message('', '');
  message('', '', 'message-login');
  message('', '', 'message-inscription');
}

/* ----- connexion et inscription ----- */

async function inscrire() {
  const nom = document.getElementById('inscription-nom').value.trim();
  const mdp = document.getElementById('inscription-mdp').value;
  const mdp2 = document.getElementById('inscription-mdp2').value;

  // on verifie ici les memes regles que le serveur, pour repondre tout de suite
  if (nom.length < 3) {
    message('erreur', "L'identifiant doit faire au moins 3 caractères.", 'message-inscription');
    return;
  }
  if (mdp.length < 8) {
    message('erreur', 'Le mot de passe doit faire au moins 8 caractères.', 'message-inscription');
    return;
  }
  if (mdp !== mdp2) {
    message('erreur', 'Les deux mots de passe ne sont pas identiques.', 'message-inscription');
    return;
  }

  bloquer('vue-inscription', true);
  const r = await api('POST', '/api/register', { username: nom, password: mdp });
  if (r.status === 201) {
    // le profil existe : on enchaine sur la connexion pour eviter de
    // retaper les memes identifiants
    await entrer(nom, mdp, 'message-inscription');
  } else if (r.status === 409) {
    message('erreur', 'Ce nom est déjà pris, choisissez-en un autre.', 'message-inscription');
  } else {
    message('erreur', 'Création impossible, réessayez.', 'message-inscription');
  }
  bloquer('vue-inscription', false);
}

async function connecter() {
  const nom = document.getElementById('login-nom').value.trim();
  const mdp = document.getElementById('login-mdp').value;
  if (nom === '' || mdp === '') {
    message('erreur', 'Remplissez les deux champs.', 'message-login');
    return;
  }
  bloquer('vue-connexion', true);
  await entrer(nom, mdp, 'message-login');
  bloquer('vue-connexion', false);
}

// la connexion elle-meme, appelee par les deux cartes
async function entrer(nom, mdp, zone) {
  const r = await api('POST', '/api/login', { username: nom, password: mdp });
  if (r.status !== 200) {
    message('erreur', 'Identifiant ou mot de passe incorrect.', zone);
    return;
  }
  nomAgent = nom;
  document.getElementById('agent-nom').textContent = 'Détective ' + nom;
  viderChampsAuth();
  montrer('bureau');
}

function viderChampsAuth() {
  const champs = ['login-nom', 'login-mdp', 'inscription-nom', 'inscription-mdp', 'inscription-mdp2'];
  for (const id of champs) {
    document.getElementById(id).value = '';
  }
}

async function deconnecter() {
  await api('POST', '/api/logout');
  nomAgent = '';
  investigationId = null;
  viderChampsAuth();
  montrer('accueil');
}

/* ----- bureau des enquetes ----- */

function filtrerLangage(valeur, bouton) {
  filtreLangage = valeur;
  activerBouton(bouton);
  chargerBureau();
}

function filtrerDifficulte(valeur, bouton) {
  filtreDifficulte = valeur;
  activerBouton(bouton);
  chargerBureau();
}

// met en surbrillance le bouton clique dans son groupe
function activerBouton(bouton) {
  const groupe = bouton.parentNode.querySelectorAll('button');
  for (const b of groupe) {
    b.className = '';
  }
  bouton.className = 'actif';
}

async function chargerBureau() {
  await chargerAffaireDuJour();

  // la progression : statistiques + statut de chaque dossier
  const statuts = {};
  const prog = await api('GET', '/api/progress');
  if (prog.status !== 200) {
    montrer('connexion');
    return;
  }
  let enCours = 0;
  for (const ligne of prog.donnees.historique) {
    if (ligne.status === 'solved' && statuts[ligne.slug] === undefined) {
      statuts[ligne.slug] = { texte: 'Classée · ' + ligne.score + ' pts', classe: 'classee' };
    }
    if (ligne.status === 'started' && statuts[ligne.slug] === undefined) {
      statuts[ligne.slug] = { texte: 'En cours', classe: 'encours' };
      enCours = enCours + 1;
    }
  }
  document.getElementById('stat-resolues').textContent = prog.donnees.enquetes_resolues;
  document.getElementById('stat-score').textContent = prog.donnees.score_total;
  document.getElementById('stat-encours').textContent = enCours;
  document.getElementById('agent-score').textContent = prog.donnees.score_total + ' pts · ' +
    prog.donnees.enquetes_resolues + ' affaires classées';

  // la liste des enquetes, avec les filtres
  let url = '/api/cases?language=' + filtreLangage + '&difficulty=' + filtreDifficulte;
  const r = await api('GET', url);
  if (r.status !== 200) {
    message('erreur', 'Impossible de charger le bureau.');
    return;
  }

  const bureau = document.getElementById('bureau');
  bureau.innerHTML = '';
  let numero = 0;
  for (const e of r.donnees) {
    numero = numero + 1;
    const statut = statuts[e.slug] || { texte: 'À découvrir', classe: 'decouvrir' };

    const fiche = document.createElement('button');
    fiche.className = 'fiche' + (statut.classe === 'encours' ? ' encours' : '');

    const num = document.createElement('span');
    num.className = 'numero';
    num.textContent = '#00' + numero;
    fiche.appendChild(num);

    const lang = document.createElement('span');
    lang.className = 'lang ' + e.language;
    lang.textContent = e.language.toUpperCase();
    fiche.appendChild(lang);

    if (affaireDuJour && e.slug === affaireDuJour.slug) {
      const badge = document.createElement('span');
      badge.className = 'badge-jour';
      badge.textContent = 'Affaire du jour';
      fiche.appendChild(badge);
    }

    const titre = document.createElement('h3');
    titre.textContent = e.title;
    fiche.appendChild(titre);

    const resume = document.createElement('div');
    resume.className = 'resume';
    resume.textContent = 'Difficulté : ' + e.difficulty;
    fiche.appendChild(resume);

    const st = document.createElement('div');
    st.className = 'statut';
    const span = document.createElement('span');
    span.className = statut.classe;
    span.textContent = statut.texte;
    st.appendChild(span);
    fiche.appendChild(st);

    const slug = e.slug;
    const n = numero;
    fiche.onclick = function () { ouvrirDossier(slug, n); };
    bureau.appendChild(fiche);
  }
}

// l'affaire mise en avant du jour : le serveur la choisit avec la date,
// donc elle reste la meme toute la journee
async function chargerAffaireDuJour() {
  const banniere = document.getElementById('affaire-jour');
  const r = await api('GET', '/api/cases/daily');
  if (r.status !== 200) {
    affaireDuJour = null;
    banniere.className = 'cadre affaire-jour cache';
    return;
  }
  affaireDuJour = r.donnees;

  const morceaux = affaireDuJour.date.split('-');
  document.getElementById('jour-date').textContent =
    morceaux[2] + '/' + morceaux[1] + '/' + morceaux[0];
  document.getElementById('jour-titre').textContent = affaireDuJour.title;
  document.getElementById('jour-details').textContent =
    affaireDuJour.language + ' · difficulté : ' + affaireDuJour.difficulty;
  banniere.className = 'cadre affaire-jour';
}

function ouvrirAffaireDuJour() {
  if (affaireDuJour) {
    ouvrirDossier(affaireDuJour.slug, affaireDuJour.numero);
  }
}

/* ----- le dossier ----- */

async function ouvrirDossier(slug, numero) {
  const r = await api('GET', '/api/cases/' + slug);
  if (r.status !== 200) {
    message('erreur', 'Impossible de charger ce dossier.');
    return;
  }
  enquete = r.donnees;
  suspectTrouve = null;

  const inv = await api('POST', '/api/investigations', { slug: slug });
  if (inv.status !== 200 && inv.status !== 201) {
    message('erreur', 'Impossible de commencer cette enquête.');
    return;
  }
  investigationId = inv.donnees.investigation_id;

  document.getElementById('dossier-ref').textContent =
    'Affaire #00' + numero + ' · ' + enquete.language + ' · ' + enquete.difficulty;
  document.getElementById('dossier-titre').textContent = enquete.title;
  document.getElementById('dossier-tampon').textContent = 'EN COURS';
  document.getElementById('dossier-tampon').className = 'tampon';
  document.getElementById('incident-attendu').textContent = enquete.incident.expected;
  document.getElementById('incident-observe').textContent = enquete.incident.observed;
  document.getElementById('code').textContent = enquete.code;

  const pieces = document.getElementById('pieces');
  pieces.innerHTML = '';
  let n = 0;
  for (const texte of enquete.evidence) {
    n = n + 1;
    const div = document.createElement('div');
    div.className = 'piece';
    const b = document.createElement('b');
    b.textContent = 'PIÈCE ' + n;
    div.appendChild(b);
    div.appendChild(document.createTextNode(texte));
    pieces.appendChild(div);
  }

  const suspects = document.getElementById('suspects');
  suspects.innerHTML = '';
  for (const s of enquete.suspects) {
    const btn = document.createElement('button');
    btn.className = 'suspect';
    const b = document.createElement('b');
    b.textContent = s.id;
    btn.appendChild(b);
    btn.appendChild(document.createTextNode(s.label));
    btn.onclick = function () { choisirSuspect(s.id); };
    suspects.appendChild(btn);
  }

  document.getElementById('indices').innerHTML = '';
  document.getElementById('bouton-indice').disabled = false;
  document.getElementById('corrections-zone').className = 'cache';
  document.getElementById('rapport').className = 'rapport-final cache';

  montrer('dossier');
  if (inv.donnees.reprise) {
    message('info', 'Reprise de l\'enquête en cours.');
  } else {
    message('info', 'Enquête ouverte. Examinez les pièces à conviction.');
  }
}

async function choisirSuspect(id) {
  bloquer('suspects', true);
  const r = await api('POST', '/api/investigations/' + investigationId + '/suspects',
    { suspect_id: id });
  bloquer('suspects', false);
  if (r.status === 409) {
    message('erreur', 'Cette affaire est déjà classée.');
    return;
  }
  if (r.status !== 200) {
    message('erreur', 'Hypothèse refusée.');
    return;
  }
  if (r.donnees.correct) {
    suspectTrouve = id;
    message('reussite', 'Bonne piste ! Choisissez maintenant la correction.');
    afficherCorrections();
  } else {
    message('erreur', 'Fausse piste (−10 pts). Réessayez ou demandez un indice.');
  }
}

function afficherCorrections() {
  const zone = document.getElementById('corrections');
  zone.innerHTML = '';
  for (const f of enquete.fixes) {
    const btn = document.createElement('button');
    btn.className = 'suspect';
    const b = document.createElement('b');
    b.textContent = f.id;
    btn.appendChild(b);
    btn.appendChild(document.createTextNode(f.label));
    btn.onclick = function () { choisirCorrection(f.id); };
    zone.appendChild(btn);
  }
  document.getElementById('corrections-zone').className = '';
}

async function demanderIndice() {
  const bouton = document.getElementById('bouton-indice');
  bouton.disabled = true;
  // l'assistant-detective peut prendre plusieurs secondes (section 13.2)
  message('chargement', "L'assistant-détective examine le dossier…");
  const r = await api('POST', '/api/investigations/' + investigationId + '/hints');
  bouton.disabled = false;

  if (r.status === 409) {
    message('erreur', r.donnees.error);
    return;
  }
  if (r.status !== 200) {
    message('erreur', 'Indice indisponible.');
    return;
  }
  message('indice', 'Indice ' + r.donnees.level + ' — ' + r.donnees.hint);
  const p = document.createElement('p');
  let ligne = r.donnees.level + '. ' + r.donnees.hint;
  if (r.donnees.source === 'ia') {
    ligne = ligne + ' (reformulé par l\'assistant-détective)';
  }
  p.textContent = ligne;
  document.getElementById('indices').appendChild(p);
}

async function choisirCorrection(fixId) {
  bloquer('corrections', true);
  message('chargement', "L'assistant-détective rédige le rapport…");
  const r = await api('POST', '/api/investigations/' + investigationId + '/verdict',
    { suspect_id: suspectTrouve, fix_id: fixId });
  bloquer('corrections', false);
  if (r.status !== 200) {
    message('erreur', 'Verdict refusé.');
    return;
  }
  if (!r.donnees.solved) {
    message('erreur', 'Le verdict n\'a pas abouti.');
    return;
  }
  message('reussite', 'Affaire classée ! Score : ' + r.donnees.score + ' points.');
  document.getElementById('dossier-tampon').textContent = 'CLASSÉE';
  document.getElementById('dossier-tampon').className = 'tampon classee';
  chargerRapport();
}

async function chargerRapport() {
  const r = await api('GET', '/api/investigations/' + investigationId + '/report');
  if (r.status !== 200) {
    return;
  }
  document.getElementById('rapport-score').textContent = r.donnees.score + ' / 110';
  document.getElementById('rapport-cause').textContent = r.donnees.correct_suspect_id;
  document.getElementById('rapport-correction').textContent = r.donnees.correction_correcte;
  document.getElementById('rapport-lecon').textContent = r.donnees.lesson;
  if (r.donnees.commentaire) {
    document.getElementById('rapport-commentaire').textContent = r.donnees.commentaire;
    document.getElementById('rapport-commentaire-ligne').className = '';
  } else {
    document.getElementById('rapport-commentaire-ligne').className = 'cache';
  }
  document.getElementById('rapport').className = 'rapport-final';
}

/* ----- profil ----- */

async function chargerProfil() {
  const r = await api('GET', '/api/progress');
  if (r.status !== 200) {
    montrer('connexion');
    return;
  }
  document.getElementById('profil-nom').textContent = nomAgent;
  document.getElementById('profil-score').textContent = r.donnees.score_total + ' pts';
  document.getElementById('profil-resolues').textContent = r.donnees.enquetes_resolues;
  document.getElementById('profil-meilleur').textContent = r.donnees.meilleur_score + ' pts';

  // les concepts deja travailles (F10)
  const zoneConcepts = document.getElementById('profil-concepts');
  zoneConcepts.innerHTML = '';
  if (r.donnees.concepts.length === 0) {
    const vide = document.createElement('p');
    vide.className = 'italique';
    vide.textContent = 'Aucun pour le moment : classez une première affaire.';
    zoneConcepts.appendChild(vide);
  }
  for (const concept of r.donnees.concepts) {
    const etiquette = document.createElement('span');
    etiquette.className = 'concept';
    etiquette.textContent = concept;
    zoneConcepts.appendChild(etiquette);
  }

  const corps = document.getElementById('profil-historique');
  corps.innerHTML = '';
  for (const l of r.donnees.historique) {
    const tr = document.createElement('tr');
    const valeurs = [
      l.title,
      l.language,
      l.concept,
      l.status === 'solved' ? 'Classée' : 'En cours',
      l.status === 'solved' ? l.score + ' pts' : '—',
      l.started_at.substring(0, 10)
    ];
    for (const v of valeurs) {
      const td = document.createElement('td');
      td.textContent = v;
      tr.appendChild(td);
    }
    corps.appendChild(tr);
  }
}
