package postgres

import (
	"github.com/bazarghan/enssi-tel-bot/internal/domain/quiz"
)

func toDomainAttempt(m AttemptModel, qm QuizModel, answers []AnswerModel) quiz.Attempt {
	questions := make([]quiz.Question, len(qm.Questions))
	for i, q := range qm.Questions {
		questions[i] = toDomainQuestion(q)
	}

	// Build the map of user answers
	ua := make(map[uint]uint)
	for _, ans := range answers {
		ua[ans.QuizQuestionID] = ans.QuizQuestionOptionID
	}

	return quiz.Attempt{
		ID:                       m.ID,
		UserID:                   m.UserID,
		QuizID:                   m.QuizID,
		TriggerProgress:          qm.TriggerProgress, // <-- ADD THIS MAPPING
		Score:                    m.Score,
		IsCompleted:              m.IsCompleted,
		CurrentQuestionIndex:     m.CurrentQuestionNum,
		CurrentQuestionMessageID: m.CurrentQuestionMessageID,
		Questions:                questions,
		Type:                     qm.Type,
		CourseID:                 qm.CourseID,
		UserAnswers:              ua,
	}
}

func toDomainQuestion(m QuestionModel) quiz.Question {
	options := make([]quiz.Option, len(m.Options))
	for i, o := range m.Options {
		options[i] = toDomainOption(o)
	}
	return quiz.Question{
		ID:      m.ID,
		Text:    m.Text,
		WordID:  m.WordID,
		Options: options,
	}
}

func toDomainOption(m OptionModel) quiz.Option {
	return quiz.Option{
		ID:        m.ID,
		Text:      m.Text,
		IsCorrect: m.IsCorrect,
	}
}
