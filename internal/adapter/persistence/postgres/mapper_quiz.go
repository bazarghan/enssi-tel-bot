package postgres

import "github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"

func toDomainAttempt(m attemptModel, qm quizModel) quiz.Attempt {
	questions := make([]quiz.Question, len(qm.Questions))
	for i, q := range qm.Questions {
		questions[i] = toDomainQuestion(q)
	}

	return quiz.Attempt{
		ID:                       m.ID,
		UserID:                   m.UserID,
		QuizID:                   m.QuizID,
		Score:                    m.Score,
		IsCompleted:              m.IsCompleted,
		CurrentQuestionIndex:     m.CurrentQuestionIndex,
		CurrentQuestionMessageID: m.CurrentQuestionMessageID,
		Questions:                questions,
		Type:                     qm.Type,
		CourseID:                 qm.CourseID,
	}
}

func toDomainQuestion(m questionModel) quiz.Question {
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

func toDomainOption(m optionModel) quiz.Option {
	return quiz.Option{
		ID:        m.ID,
		Text:      m.Text,
		IsCorrect: m.IsCorrect,
	}
}
