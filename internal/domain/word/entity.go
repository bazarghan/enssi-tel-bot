package word

// Word represents a single vocabulary word with its definitions and pronunciations.
type Word struct {
	ID             uint
	Title          string
	Phonetic       string
	PrimaryDef     string
	SecondaryDef   string
	PartsOfSpeech  []PartOfSpeech
	Pronunciations []Pronunciation
	ImageURL       string
}

// PartOfSpeech groups meanings by their grammatical type (e.g., noun, verb).
type PartOfSpeech struct {
	Title    string
	Meanings []Meaning
}

// Meaning is a single definition of a word, in a specific language.
type Meaning struct {
	Lang  string
	Title string
}

// Pronunciation holds audio data for a word.
type Pronunciation struct {
	ID       uint
	Region   string
	AudioURL string
}
