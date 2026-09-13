package cases

type Incident struct {
	Expected string `json:"expected"`
	Observed string `json:"observed"`
}

type Suspect struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type Fix struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Correct bool   `json:"correct"`
}

type Case struct {
	Slug             string    `json:"slug"`
	Title            string    `json:"title"`
	Language         string    `json:"language"`
	Difficulty       string    `json:"difficulty"`
	Concept          string    `json:"concept"`
	Incident         Incident  `json:"incident"`
	Code             string    `json:"code"`
	Evidence         []string  `json:"evidence"`
	Suspects         []Suspect `json:"suspects"`
	CorrectSuspectID string    `json:"correct_suspect_id"`
	Hints            []string  `json:"hints"`
	Fixes            []Fix     `json:"fixes"`
	Lesson           string    `json:"lesson"`
}

type PublicFix struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ce qui part vers le navigateur : sans la solution ni les indices
type PublicCase struct {
	Slug       string      `json:"slug"`
	Title      string      `json:"title"`
	Language   string      `json:"language"`
	Difficulty string      `json:"difficulty"`
	Concept    string      `json:"concept"`
	Incident   Incident    `json:"incident"`
	Code       string      `json:"code"`
	Evidence   []string    `json:"evidence"`
	Suspects   []Suspect   `json:"suspects"`
	Fixes      []PublicFix `json:"fixes"`
}

func (c Case) Public() PublicCase {
	fixes := []PublicFix{}
	for _, f := range c.Fixes {
		fixes = append(fixes, PublicFix{ID: f.ID, Label: f.Label})
	}
	return PublicCase{
		Slug:       c.Slug,
		Title:      c.Title,
		Language:   c.Language,
		Difficulty: c.Difficulty,
		Concept:    c.Concept,
		Incident:   c.Incident,
		Code:       c.Code,
		Evidence:   c.Evidence,
		Suspects:   c.Suspects,
		Fixes:      fixes,
	}
}
