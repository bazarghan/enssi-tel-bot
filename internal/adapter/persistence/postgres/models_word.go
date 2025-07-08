package postgres

import (
	"gorm.io/gorm"
	"time"
)

// This file aggregates all models related to the Word domain.

type WordModel struct {
	gorm.Model
	Lang  string `gorm:"not null"`
	Title string `gorm:"not null"`
}

func (WordModel) TableName() string { return "words" }

type SourceModel struct {
	gorm.Model
	Words []WordSourceModel

	Title       string `gorm:"not null"`
	Description string
	URL         string `gorm:"not null"`
}

func (SourceModel) TableName() string { return "sources" }

type WordSourceModel struct {
	gorm.Model
	WordID   uint
	SourceID uint

	PartsOfSpeeches []PartOfSpeechModel  `gorm:"foreignKey:WordSourceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Pronunciations  []PronunciationModel `gorm:"foreignKey:WordSourceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Phonetics       []PhoneticModel      `gorm:"foreignKey:WordSourceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Images          []ImageModel         `gorm:"foreignKey:WordSourceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	DefPrimary   string
	DefSecondary string
}

func (WordSourceModel) TableName() string { return "word_sources" }

type PartOfSpeechModel struct {
	gorm.Model
	WordSourceID uint
	Title        string         `gorm:"not null"`
	Meanings     []MeaningModel `gorm:"foreignKey:PartOfSpeechID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (PartOfSpeechModel) TableName() string { return "part_of_speeches" }

type MeaningModel struct {
	gorm.Model
	PartOfSpeechID uint
	Lang           string `gorm:"not null"`
	Title          string `gorm:"not null"`
}

func (MeaningModel) TableName() string { return "meanings" }

type PhoneticModel struct {
	gorm.Model
	WordSourceID uint
	Lang         string `gorm:"not null"`
	Title        string `gorm:"not null"`
}

func (PhoneticModel) TableName() string { return "phonetics" }

type PronunciationModel struct {
	gorm.Model
	WordSourceID   uint
	Region         string `gorm:"not null"`
	URL            string `gorm:"not null"`
	TelgramVoiceID string
}

func (PronunciationModel) TableName() string { return "pronunciations" }

type ImageModel struct {
	gorm.Model
	WordSourceID uint
	URL          string `gorm:"not null"`
}

func (ImageModel) TableName() string { return "images" }

type StudiedWordModel struct {
	gorm.Model
	UserID             uint `gorm:"index"`
	WordID             uint `gorm:"index"`
	LastReviewedAt     time.Time
	NextReviewAt       time.Time `gorm:"index"`
	ReviewIntervalDays uint      `gorm:"default:1"`
	IsMastered         bool      `gorm:"default:false;not null"`
}

func (StudiedWordModel) TableName() string { return "word_studieds" }
