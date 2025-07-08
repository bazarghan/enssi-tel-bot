package seeder

// JSON parsing structs from your original data populating project
type Phonetic struct {
	Lang  string `json:"lang"`
	Title string `json:"title"`
}

type Pronunciation struct {
	Region string `json:"region"`
	URL    string `json:"url"`
}

type Meaning struct {
	Lang  string `json:"lang"`
	Title string `json:"title"`
}

type PartOfSpeech struct {
	Title    string    `json:"title"`
	Meanings []Meaning `json:"meanings"`
}

type FlashCard struct {
	TelgramImageID    string `json:"image_id"`
	TelgramImageDocID string `json:"doc_id"`
}

type WordDetails struct {
	Lang              string          `json:"lang"`
	DefPrimary        string          `json:"def_primary"`
	DefSecondary      string          `json:"def_secondary"`
	PartOfSpeeches    []PartOfSpeech  `json:"part_of_speeches"`
	Pronunciations    []Pronunciation `json:"pronunciations"`
	Phonetics         []Phonetic      `json:"phonetics"`
	TelgramVoiceDocID string          `json:"tel_pronunciation_voice_file_id"`
	FlashCard         FlashCard       `json:"flashcard"`
}

type OrderedWordListItem struct {
	Word    string      `json:"word"`
	Details WordDetails `json:"details"`
}
