package formatters

import (
	"fmt"
	"github.com/bazarghan/enssi-tel-bot/internal/adapter/telegram/dto"
	"github.com/bazarghan/enssi-tel-bot/pkg/tgmarkdown"
	"strings"
)

// FormatCourseListMessage creates a formatted message for the list of courses.
func FormatCourseListMessage(courses []dto.CourseSummary) string {

	message := "حالا که تصمیم گرفتی یادگیری رو شروع کنی تو این مرحله باید مجموعه ای که می خوای رو انتخاب کنی و ادامه بدی 🙂"
	if len(courses) == 0 {
		message = "در حال حاضر هیچ دوره ای برای نمایش وجود ندارد."
	}

	var sb strings.Builder

	sb.WriteString("\\.\n")
	sb.WriteString(">  \n")
	sb.WriteString(fmt.Sprintf(">%s\n>  \n\n\\.\n", tgmarkdown.Escape(message)))

	return sb.String()
}

// FormatCourseOverview formats the detailed course view.
func FormatCourseOverview(co dto.CourseOverview) string {
	var sb strings.Builder

	var message string
	if !co.IsStarted {
		message = "اگه آماده ای پس منتظر چی هستی بزن رو شروع دوره که بی معطلی یادگیری رو شروع کنیم، در ضمن نگران نباش می تونی به صورت همزمان چند تا دوره رو شروع کنی و جلو ببری پس خیالت از این بابت راحت باشه 😉"
	} else if co.IsCompleted {
		message = "تبریک بابت تموم کردن دوره! 👏 حالا اگه دوست داری مطالب رو مرور کنی یا یه نگاهی دوباره بندازی، کافیه روی «مرور» بزنی و هرجا لازم بود مرورش کنی."
	} else {
		message = "خوش اومدی! وقتشه ادامه مسیر یادگیری‌ت رو پیش ببری. با یک کلیک روی «ادامه دوره» برگرد سراغ مطالب و یادگیری رو منظم‌تر ادامه بده."
	}

	sb.WriteString("\\.\n")
	sb.WriteString(">  \n")
	descriptionLines := strings.Split(tgmarkdown.Escape(co.PersianFullDescription), "\n")
	for _, line := range descriptionLines {
		// Prefix each line with "> " to create a valid blockquote
		sb.WriteString(fmt.Sprintf(">%s\n", line))
	}
	sb.WriteString(">  \n")
	sb.WriteString(fmt.Sprintf("\n%s\n\n", tgmarkdown.Escape(message)))

	if co.IsCompleted {
		sb.WriteString("وضعیت: *تکمیل شده* 🏆\n")
	} else if co.IsStarted {
		sb.WriteString(fmt.Sprintf("پیشرفت شما: *%d%%*\n", co.ProgressPercentage))
	} else {
		sb.WriteString("وضعیت: *شروع نشده*\n")
	}
	return sb.String()
}
