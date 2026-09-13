package game

// bonus de temps du mode chronometre : plus l'affaire est bouclee vite,
// plus il est eleve. Les secondes viennent toujours du serveur.
func BonusTemps(secondes int) int {
	if secondes < 180 {
		return 10
	}
	if secondes < 360 {
		return 5
	}
	return 0
}

// bareme de la section 5.2 du sujet, avec le bonus de temps
func Score(mauvaisesHypotheses int, indicesUtilises int, bonusPremierEssai bool, secondes int) int {
	score := 100

	score = score - 10*mauvaisesHypotheses

	if indicesUtilises >= 1 {
		score = score - 5
	}
	if indicesUtilises >= 2 {
		score = score - 10
	}
	if indicesUtilises >= 3 {
		score = score - 15
	}

	if bonusPremierEssai {
		score = score + 10
	}

	score = score + BonusTemps(secondes)

	if score < 0 {
		score = 0
	}
	if score > 120 {
		score = 120
	}
	return score
}
