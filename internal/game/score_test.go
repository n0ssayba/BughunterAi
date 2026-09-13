package game

import "testing"

// au-dela de 6 minutes il n'y a plus de bonus de temps
const lent = 400

func TestScoreParfait(t *testing.T) {
	if Score(0, 0, true, lent) != 110 {
		t.Error("attendu 110, obtenu", Score(0, 0, true, lent))
	}
}

func TestScoreSansBonus(t *testing.T) {
	if Score(0, 0, false, lent) != 100 {
		t.Error("attendu 100, obtenu", Score(0, 0, false, lent))
	}
}

func TestScoreMauvaiseHypothese(t *testing.T) {
	if Score(1, 0, false, lent) != 90 {
		t.Error("attendu 90, obtenu", Score(1, 0, false, lent))
	}
	if Score(2, 0, false, lent) != 80 {
		t.Error("attendu 80, obtenu", Score(2, 0, false, lent))
	}
}

func TestScoreIndices(t *testing.T) {
	if Score(0, 1, false, lent) != 95 {
		t.Error("1 indice : attendu 95, obtenu", Score(0, 1, false, lent))
	}
	if Score(0, 2, false, lent) != 85 {
		t.Error("2 indices : attendu 85, obtenu", Score(0, 2, false, lent))
	}
	if Score(0, 3, false, lent) != 70 {
		t.Error("3 indices : attendu 70, obtenu", Score(0, 3, false, lent))
	}
}

func TestScoreJamaisNegatif(t *testing.T) {
	if Score(15, 3, false, lent) != 0 {
		t.Error("attendu 0, obtenu", Score(15, 3, false, lent))
	}
}

func TestBonusTemps(t *testing.T) {
	if BonusTemps(60) != 10 {
		t.Error("moins de 3 minutes : attendu 10, obtenu", BonusTemps(60))
	}
	if BonusTemps(300) != 5 {
		t.Error("moins de 6 minutes : attendu 5, obtenu", BonusTemps(300))
	}
	if BonusTemps(600) != 0 {
		t.Error("au-dela de 6 minutes : attendu 0, obtenu", BonusTemps(600))
	}
}

func TestScoreAvecBonusTemps(t *testing.T) {
	// 100 + 10 (rapide)
	if Score(0, 0, false, 60) != 110 {
		t.Error("attendu 110, obtenu", Score(0, 0, false, 60))
	}
	// 100 - 5 (un indice) + 5 (moyennement rapide)
	if Score(0, 1, false, 300) != 100 {
		t.Error("attendu 100, obtenu", Score(0, 1, false, 300))
	}
}

func TestScoreBorneA120(t *testing.T) {
	// 100 + 10 premier essai + 10 temps = 120, le maximum
	if Score(0, 0, true, 10) != 120 {
		t.Error("attendu 120, obtenu", Score(0, 0, true, 10))
	}
}
