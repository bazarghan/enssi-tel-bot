package formatters

import (
	"fmt"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/pkg/tgmarkdown"
)

// FormatQuizQuestion creates the text for a quiz question message.
func FormatQuizQuestion(q quiz.Question, qNum, total int) string {

	questionText := fmt.Sprintf(
		"سوال %d از %d:\n\nمعنی کلمه *%s* چه می باشد؟",
		qNum+1,
		total,
		tgmarkdown.Escape(q.Text),
	)

	var textBuilder strings.Builder

	textBuilder.WriteString(fmt.Sprintf("\n%s", questionText))
	textBuilder.WriteString("\n\n")

	for i, opt := range q.Options {

		textBuilder.WriteString(
			fmt.Sprintf(
				">%d\\. %s\n",
				i+1,
				tgmarkdown.Escape(opt.Text),
			),
		)

		seperatorText := "─────────────────────────"
		textBuilder.WriteString(fmt.Sprintf(">%s\n", tgmarkdown.Escape(seperatorText)))
	}

	return textBuilder.String()
}

// FormatQuizResult creates the text for a final result message.
func FormatQuizResult(r quiz.Result, attempt quiz.Attempt) string {
	var resultMessageBuilder strings.Builder
	resultMessageBuilder.WriteString(">" + tgmarkdown.Escape("آزمون به پایان رسید!") + "\n")
	resultMessageBuilder.WriteString(">\n")
	scoreLine := fmt.Sprintf("تو به %d سوال از %d سوال پاسخ صحیح دادی.", r.Score, r.TotalQuestions)
	resultMessageBuilder.WriteString(">" + tgmarkdown.Escape(scoreLine) + "\n")

	if r.Passed {
		resultMessageBuilder.WriteString(">" + tgmarkdown.Escape("آفرین! تونستی آزمون رو با موفقیت پشت سر بذاری. 🎉"))
	} else {
		resultMessageBuilder.WriteString(">" + tgmarkdown.Escape("متاسفانه نتونستی حد نصاب قبولی رو کسب کنی. 😔") + "\n")
		resultMessageBuilder.WriteString(">" + tgmarkdown.Escape("به همین دلیل باید این بخش رو دوباره مرور کنی."))
	}

	// --- Build Review Text ---
	var reviewTextBuilder strings.Builder
	if len(attempt.Questions) > 0 {
		reviewTextBuilder.WriteString("\n\n")
		reviewTextBuilder.WriteString(tgmarkdown.Escape("─────────────────────────") + "\n")
		reviewTextBuilder.WriteString("📝 *مرور سوالات:*\n")

		for i, q := range attempt.Questions {
			// In a full implementation, you'd fetch the user's answer.
			// For this slice, we'll just show the correct answer.
			var correctOptText string
			for _, opt := range q.Options {
				if opt.IsCorrect {
					correctOptText = opt.Text
					break
				}
			}

			reviewTextBuilder.WriteString(fmt.Sprintf("\n> *سوال %d:* %s\n", i+1, tgmarkdown.Escape(q.Text)))
			reviewTextBuilder.WriteString(fmt.Sprintf("> *پاسخ صحیح:* %s\n", tgmarkdown.Escape(correctOptText)))
		}
	}

	return resultMessageBuilder.String() + reviewTextBuilder.String()
}
