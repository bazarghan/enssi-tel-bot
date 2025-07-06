package word

import (
	"time"
)

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

// StudiedWord tracks a user's learning progress for a single word in the SRS.
type StudiedWord struct {
	ID                 uint
	UserID             uint
	WordID             uint
	LastReviewedAt     time.Time
	NextReviewAt       time.Time
	ReviewIntervalDays uint
	IsMastered         bool
}

// DisplayablePronunciation contains both domain data and adapter-specific IDs.
type DisplayablePronunciation struct {
	Pronunciation
	TelegramVoiceID string
}

// DisplayableWord is a DTO for the repository layer, containing all data needed by a use case.
type DisplayableWord struct {
	Word
	CourseWordID       uint
	TelegramImageID    string
	TelegramImageDocID string
	Pronunciations     []DisplayablePronunciation
}
