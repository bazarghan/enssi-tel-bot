package postgres

import (
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"gorm.io/gorm"
)

type quizModel struct {
	gorm.Model
	CourseID        uint
	Questions       []questionModel `gorm:"foreignKey:QuizID"`
	QuestionCount   uint
	Type            quiz.QuizType `gorm:"type:varchar(50);default:'COURSE_BLOCK'"`
	TriggerProgress uint
}

func (quizModel) TableName() string { return "quizzes" }

type questionModel struct {
	gorm.Model
	QuizID  uint
	Text    string
	Options []optionModel `gorm:"foreignKey:QuizQuestionID"`
	WordID  uint          `gorm:"index"`
}

func (questionModel) TableName() string { return "quiz_questions" }

type optionModel struct {
	gorm.Model
	QuizQuestionID uint   `gorm:"not null"`
	Text           string `gorm:"not null"`
	IsCorrect      bool   `gorm:"not null;default:false"`
}

func (optionModel) TableName() string { return "quiz_question_options" }

type attemptModel struct {
	gorm.Model
	QuizID                   uint `gorm:"not null"`
	UserID                   uint `gorm:"not null"`
	Score                    int  `gorm:"default:0"`
	IsCompleted              bool `gorm:"default:false"`
	Answers                  []answerModel
	CurrentQuestionIndex     int `gorm:"default:0"`
	CurrentQuestionMessageID int
}

func (attemptModel) TableName() string { return "quiz_attempts" }

type answerModel struct {
	gorm.Model
	QuizAttemptID        uint
	QuizQuestionID       uint
	QuizQuestionOptionID uint
	IsCorrect            bool
}

func (answerModel) TableName() string { return "quiz_answers" }
