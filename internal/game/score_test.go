package game

import "testing"

func TestScoreParfait(t *testing.T) {
	if Score(0, 0, true) != 110 {
		t.Error("attendu 110, obtenu", Score(0, 0, true))
	}
}

func TestScoreSansBonus(t *testing.T) {
	if Score(0, 0, false) != 100 {
		t.Error("attendu 100, obtenu", Score(0, 0, false))
	}
}

func TestScoreMauvaiseHypothese(t *testing.T) {
	if Score(1, 0, false) != 90 {
		t.Error("attendu 90, obtenu", Score(1, 0, false))
	}
	if Score(2, 0, false) != 80 {
		t.Error("attendu 80, obtenu", Score(2, 0, false))
	}
}

func TestScoreIndices(t *testing.T) {
	if Score(0, 1, false) != 95 {
		t.Error("1 indice : attendu 95, obtenu", Score(0, 1, false))
	}
	if Score(0, 2, false) != 85 {
		t.Error("2 indices : attendu 85, obtenu", Score(0, 2, false))
	}
	if Score(0, 3, false) != 70 {
		t.Error("3 indices : attendu 70, obtenu", Score(0, 3, false))
	}
}

func TestScoreJamaisNegatif(t *testing.T) {
	if Score(15, 3, false) != 0 {
		t.Error("attendu 0, obtenu", Score(15, 3, false))
	}
}
