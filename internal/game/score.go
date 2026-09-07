package game

// bareme de la section 5.2 du sujet
func Score(mauvaisesHypotheses int, indicesUtilises int, bonusPremierEssai bool) int {
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

	if score < 0 {
		score = 0
	}
	if score > 110 {
		score = 110
	}
	return score
}
