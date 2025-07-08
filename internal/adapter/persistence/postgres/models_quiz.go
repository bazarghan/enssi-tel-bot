package postgres

import (
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"gorm.io/gorm"
)

type QuizModel struct {
	gorm.Model
	CourseID        uint
	Questions       []QuestionModel `gorm:"foreignKey:QuizID"`
	QuestionCount   uint
	Type            quiz.QuizType `gorm:"type:varchar(50);default:'COURSE_BLOCK'"`
	TriggerProgress uint
}

func (QuizModel) TableName() string { return "quizzes" }

type QuestionModel struct {
	gorm.Model
	QuizID  uint
	Text    string
	Options []OptionModel `gorm:"foreignKey:QuizQuestionID"`
	WordID  uint          `gorm:"index"`
}

func (QuestionModel) TableName() string { return "quiz_questions" }

type OptionModel struct {
	gorm.Model
	QuizQuestionID uint   `gorm:"not null"`
	Text           string `gorm:"not null"`
	IsCorrect      bool   `gorm:"not null;default:false"`
}

func (OptionModel) TableName() string { return "quiz_question_options" }

type AttemptModel struct {
	gorm.Model
	QuizID                   uint          `gorm:"not null"`
	UserID                   uint          `gorm:"not null"`
	Score                    int           `gorm:"default:0"`
	IsCompleted              bool          `gorm:"default:false"`
	Answers                  []AnswerModel `gorm:"foreignKey:QuizAttemptID"`
	CurrentQuestionNum       int           `gorm:"default:0"`
	CurrentQuestionMessageID int
}

func (AttemptModel) TableName() string { return "quiz_attempts" }

type AnswerModel struct {
	gorm.Model
	QuizAttemptID        uint
	QuizQuestionID       uint
	QuizQuestionOptionID uint
	IsCorrect            bool
}

func (AnswerModel) TableName() string { return "quiz_answers" }
