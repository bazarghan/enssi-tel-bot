package dto

import "github.com/bazarghan/enssi-tel-bot/internal/domain/quiz"

// QuestionView is a DTO for rendering a quiz question.
type QuestionView struct {
	AttemptID uint
	Index     int
	Total     int
	Text      string
	Options   []quiz.Option
}

// ResultView is a DTO for rendering the final quiz result.
type ResultView struct {
	Score          int
	TotalQuestions int
	Passed         bool
	ResultMessage  string
	ReviewText     string
}
