package formatters

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	"github.com/2000ostd/enssi-tel-bot/pkg/tgmarkdown"
	"strings"
)

// FormatCourseListMessage creates a formatted message for the list of courses.
func FormatCourseListMessage(courses []dto.CourseSummary) string {
	if len(courses) == 0 {
		return tgmarkdown.Escape("در حال حاضر دوره‌ای برای نمایش وجود ندارد.")
	}
	return tgmarkdown.Escape("لطفا یک دوره را برای شروع انتخاب کنید:")
}

// FormatCourseOverview formats the detailed course view.
func FormatCourseOverview(co dto.CourseOverview) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%s*\n", tgmarkdown.Escape(co.PersianTitle)))
	if co.Title != co.PersianTitle {
		sb.WriteString(fmt.Sprintf("_\\(%s\\)_\n", tgmarkdown.Escape(co.Title)))
	}
	sb.WriteString(fmt.Sprintf("\n%s\n", tgmarkdown.Escape(co.PersianFullDescription)))
	sb.WriteString(fmt.Sprintf("\nتعداد کلمات: *%d*\n", co.TotalWords))

	if co.IsCompleted {
		sb.WriteString("وضعیت: *تکمیل شده* 🏆\n")
	} else if co.IsStarted {
		sb.WriteString(fmt.Sprintf("پیشرفت شما: *%d%%*\n", co.ProgressPercentage))
	} else {
		sb.WriteString("وضعیت: *شروع نشده*\n")
	}
	return sb.String()
}

