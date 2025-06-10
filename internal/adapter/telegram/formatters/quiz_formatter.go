package formatters

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/pkg/tgmarkdown"
)

// FormatQuizQuestion creates the text for a quiz question message.
func FormatQuizQuestion(q quiz.Question, attemptID int) string {
	// Logic to format question text, ported from old view_helpers
	return tgmarkdown.Escape(q.Text)
}

// FormatQuizResult creates the text for a final result message.
func FormatQuizResult(r quiz.Result) string {
	// Logic to format result, ported from old quiz service formatter
	header := fmt.Sprintf("🏁 آزمون تمام شد! 🏁\n\nنمره شما: *%d* از *%d*", r.Score, r.TotalQuestions)
	var passText string
	if r.Passed {
		passText = "🎉 عالی بود! شما آزمون را با موفقیت گذراندید."
	} else {
		passText = "😔 متاسفانه حد نصاب قبولی را کسب نکردید. دوباره تلاش کنید!"
	}

	return fmt.Sprintf("%s\n%s", header, tgmarkdown.Escape(passText))
}
