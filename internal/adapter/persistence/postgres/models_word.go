package postgres

import "gorm.io/gorm"

// This file aggregates all models related to the Word domain.

type wordModel struct {
	gorm.Model
	Lang  string `gorm:"not null"`
	Title string `gorm:"not null"`
}

func (wordModel) TableName() string { return "words" }

type wordSourceModel struct {
	gorm.Model
	WordID          uint
	DefPrimary      string
	DefSecondary    string
	PartsOfSpeeches []partOfSpeechModel  `gorm:"foreignKey:WordSourceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Pronunciations  []pronunciationModel `gorm:"foreignKey:WordSourceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Phonetics       []phoneticModel      `gorm:"foreignKey:WordSourceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Images          []imageModel         `gorm:"foreignKey:WordSourceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (wordSourceModel) TableName() string { return "word_sources" }

type partOfSpeechModel struct {
	gorm.Model
	WordSourceID uint
	Title        string         `gorm:"not null"`
	Meanings     []meaningModel `gorm:"foreignKey:PartOfSpeechID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (partOfSpeechModel) TableName() string { return "part_of_speeches" }

type meaningModel struct {
	gorm.Model
	PartOfSpeechID uint
	Lang           string `gorm:"not null"`
	Title          string `gorm:"not null"`
}

func (meaningModel) TableName() string { return "meanings" }

type phoneticModel struct {
	gorm.Model
	WordSourceID uint
	Lang         string `gorm:"not null"`
	Title        string `gorm:"not null"`
}

func (phoneticModel) TableName() string { return "phonetics" }

type pronunciationModel struct {
	gorm.Model
	WordSourceID   uint
	Region         string `gorm:"not null"`
	URL            string `gorm:"not null"`
	TelgramVoiceID string
}

func (pronunciationModel) TableName() string { return "pronunciations" }

type imageModel struct {
	gorm.Model
	WordSourceID uint
	URL          string `gorm:"not null"`
}

func (imageModel) TableName() string { return "images" }
