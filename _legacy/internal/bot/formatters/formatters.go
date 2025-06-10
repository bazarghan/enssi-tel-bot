package formatters

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/models"
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

func FormatCourseProgress(co *course.CourseOverview) string {
	// A small slice of motivational messages to pick from randomly.
	motivationalPhrases := []string{
		"هر کلمه‌ی جدید، یک آجر برای ساختن برج دانش توئه. 🏛️ آماده‌ای که آجر بعدی رو روی هم بذاریم؟",
		"فوق‌العاده‌ست! ✨ دانش مثل یک اقیانوسه و تو داری عالی پیش میری. موج بعدی رو بگیریم؟ 🌊",
		"عالیه که برگشتی! 🔥 یادگیری یک مسیره، نه یک مقصد. بیا قدم بعدی رو با هم برداریم.",
		"موتور یادگیریت روشنه! 🚀 آماده برای کلمه‌ی بعدی؟ بزن بریم!",
		"چه خوب که برای رشد خودت وقت میذاری. 🌱 هر قدمی که برمیداری، ارزشمنده. آماده‌ای برای ادامه؟",
	}

	// Seed the random number generator for variety each time.
	// Note: For Go 1.20+, creating a new Rand source is preferred: r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// But for this non-critical use case, the global Seed is simple and effective.
	rand.Seed(time.Now().UnixNano())
	// Pick a random phrase.
	randomPhrase := motivationalPhrases[rand.Intn(len(motivationalPhrases))]

	var sb strings.Builder
	// Course Title
	sb.WriteString(fmt.Sprintf("*%s*\n", EscapeMarkdownV2(co.PersianTitle)))
	if co.Title != co.PersianTitle {
		sb.WriteString(fmt.Sprintf("_\\(%s\\)_\n", EscapeMarkdownV2(co.Title)))
	}

	sb.WriteString("\n") // Add a space

	// --- Motivational Part ---

	startText := "سلام\\! شما تازه دوره رو شروع کردید و هنوز هیچ کلمه‌ای رو یاد نگرفتید\\. نگران نباشید، با هم اولین کلمه‌تون رو یاد می‌گیریم\\! 📝🚀\n\n"

	if co.UserProgressWords > 0 {
		startText = fmt.Sprintf(
			"تو تا اینجا *%d* کلمه از مجموع *%d* کلمه رو یاد گرفتی\\. دمت گرم\\! 💪\n\n",
			co.UserProgressWords,
			co.TotalWords,
		)
	}

	sb.WriteString(startText)
	if co.UserProgressWords > 0 {
		sb.WriteString(EscapeMarkdownV2(randomPhrase))
	}

	sb.WriteString("\n\n") // Add more spacing before the status line

	sb.WriteString(fmt.Sprintf("\nتعداد کلمات: *%d*\n", co.TotalWords))

	// Progress status
	if co.IsCompletedByUser {
		sb.WriteString("وضعیت: *تکمیل شده* 🏆\n")
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
			sb.WriteString(fmt.Sprintf("  \\- %s: %s\n", EscapeMarkdownV2(ach.Title), EscapeMarkdownV2(ach.Description)))
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
	return fmt.Sprintf("%s\n%s", header, review) // Both should be pre-escaped by service if necessary, or escape here.
	// Assuming service provides safe text or we escape it:
	// return fmt.Sprintf("%s\n\n%s", EscapeMarkdownV2(qr.ResultMessage), EscapeMarkdownV2(qr.ReviewText))
}

// FormatAllAchievementsList creates a formatted message listing all available achievements.
func FormatAllAchievementsList(achievements []models.Achievement) string {
	if len(achievements) == 0 {
		return EscapeMarkdownV2("در حال حاضر هیچ دستاوردی برای نمایش وجود ندارد.")
	}

	var sb strings.Builder
	sb.WriteString(EscapeMarkdownV2("🏆 لیست تمام دستاوردهای موجود 🏆\n\n"))

	for _, ach := range achievements {
		sb.WriteString(fmt.Sprintf("*%s*\n", EscapeMarkdownV2(ach.Title)))
		if ach.Description != "" {
			sb.WriteString(fmt.Sprintf("_%s_\n", EscapeMarkdownV2(ach.Description)))
		}
		sb.WriteString("\n") // Add a space between entries
	}

	return sb.String()
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
