package formatters

import (
	"fmt"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/services/course"
	"github.com/2000ostd/enssi-tel-bot/internal/services/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/services/user"
	"github.com/2000ostd/enssi-tel-bot/internal/services/word"
)

// FormatCourseSummary for display in a list.
func FormatCourseSummary(cs course.CourseSummaryView) string {
	var progressText string
	if cs.IsCompletedByUser {
		progressText = "(تکمیل شده)"
	} else if cs.ProgressPercentage > 0 {
		progressText = fmt.Sprintf("(%d%%)", cs.ProgressPercentage)
	} else {
		progressText = "(جدید)"
	}
	return EscapeMarkdownV2(fmt.Sprintf("%s %s - %s", cs.PersianTitle, progressText, cs.ShortDescription))
}

// FormatCourseOverview for detailed view.
func FormatCourseOverview(co *course.CourseOverview) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%s*\n", EscapeMarkdownV2(co.PersianTitle)))
	if co.Title != co.PersianTitle {
		sb.WriteString(fmt.Sprintf("_\\(%s\\)_\n", EscapeMarkdownV2(co.Title)))
	}
	sb.WriteString(fmt.Sprintf("\n%s\n", EscapeMarkdownV2(co.PersianFullDescription)))
	if co.FullDescription != "" && co.FullDescription != co.PersianFullDescription {
		sb.WriteString(fmt.Sprintf("\n\\-\\-\\-\n%s\n", EscapeMarkdownV2(co.FullDescription)))
	}
	sb.WriteString(fmt.Sprintf("\nتعداد کلمات: %d\n", co.TotalWords))

	if co.IsCompletedByUser {
		sb.WriteString("وضعیت: *تکمیل شده*\n")
	} else {
		sb.WriteString(fmt.Sprintf("پیشرفت شما: *%d%%*\n", co.ProgressPercentage))
	}
	return sb.String()
}

// FormatUserProfile for display.
func FormatUserProfile(upv *user.UserProfileView) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("👤 *پروفایل کاربری %s*\n\n", EscapeMarkdownV2(upv.FirstName)))
	if upv.Username != "" {
		sb.WriteString(fmt.Sprintf("نام کاربری تلگرام: @%s\n", EscapeMarkdownV2(upv.Username)))
	}
	sb.WriteString(fmt.Sprintf("نام: %s\n", EscapeMarkdownV2(upv.FirstName)))
	if upv.LastName != "" {
		sb.WriteString(fmt.Sprintf("نام خانوادگی: %s\n", EscapeMarkdownV2(upv.LastName)))
	}
	sb.WriteString(fmt.Sprintf("امتیاز: %d\n", upv.Score))
	sb.WriteString(fmt.Sprintf("کلمات مطالعه شده: %d\n", upv.WordsStudied)) // Ensure this is populated by service
	sb.WriteString(fmt.Sprintf("دوره‌های فعال: %d\n", upv.CoursesActive))   // Ensure this is populated

	if len(upv.Achievements) > 0 {
		sb.WriteString("\n🏆 *دستاوردها:*\n")
		for _, ach := range upv.Achievements {
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", EscapeMarkdownV2(ach.Title), EscapeMarkdownV2(ach.Description)))
		}
	}
	return sb.String()
}

// FormatWordForDisplay relies on wordData.FormattedText from the service.
// This function is a pass-through but shows where customization could happen.
func FormatWordForDisplay(wordData *word.WordDisplayData) string {
	if wordData == nil || wordData.FormattedText == "" {
		return "اطلاعات کلمه یافت نشد."
	}
	// The service already provides MarkdownV2 formatted text.
	return wordData.FormattedText
}

// FormatQuizResultForDisplay uses QuizResult.ResultMessage and QuizResult.ReviewText.
func FormatQuizResultForDisplay(qr *quiz.QuizResult) string {
	if qr == nil {
		return "نتیجه آزمون یافت نشد."
	}
	// The service already provides these formatted messages.
	header := qr.ResultMessage
	review := qr.ReviewText
	return fmt.Sprintf("%s\n\n%s", header, review) // Both should be pre-escaped by service if necessary, or escape here.
	// Assuming service provides safe text or we escape it:
	// return fmt.Sprintf("%s\n\n%s", EscapeMarkdownV2(qr.ResultMessage), EscapeMarkdownV2(qr.ReviewText))
}

// TranslatePOS translates English Part of Speech to Persian.
// (Copied from your handlers/helper.go, now centralized in formatters)
var posTranslationMap = map[string]string{
	"noun":           "اسم",
	"verb":           "فعل",
	"adjective":      "صفت",
	"adverb":         "قید",
	"pronoun":        "ضمیر",
	"preposition":    "حرف اضافه",
	"conjunction":    "حرف ربط",
	"interjection":   "صوت",
	"determiner":     "تعیین‌کننده",
	"phrasal verb":   "فعل عبارتی",
	"auxiliary verb": "فعل کمکی",
}

func TranslatePOS(posTitleEng string) string {
	titleLower := strings.ToLower(posTitleEng)
	if translated, ok := posTranslationMap[titleLower]; ok {
		return translated
	}
	return posTitleEng
}
