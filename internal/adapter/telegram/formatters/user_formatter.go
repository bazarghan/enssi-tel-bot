package formatters

import (
	"fmt"
	"github.com/bazarghan/enssi-tel-bot/internal/adapter/telegram/dto"
	"github.com/bazarghan/enssi-tel-bot/pkg/tgmarkdown"
	"strings"
)

// FormatUserProfile formats the profile DTO for display.
func FormatUserProfile(upv dto.UserProfile) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("👤 *پروفایل کاربری %s*\n\n", tgmarkdown.Escape(upv.FirstName)))
	if upv.Username != "" {
		sb.WriteString(fmt.Sprintf("نام کاربری تلگرام: @%s\n", tgmarkdown.Escape(upv.Username)))
	}
	sb.WriteString(fmt.Sprintf("نام: %s\n", tgmarkdown.Escape(upv.FirstName)))
	if upv.LastName != "" {
		sb.WriteString(fmt.Sprintf("نام خانوادگی: %s\n", tgmarkdown.Escape(upv.LastName)))
	}
	sb.WriteString(fmt.Sprintf("امتیاز: %d\n", upv.Score))
	sb.WriteString(fmt.Sprintf("کلمات مطالعه شده: %d\n", upv.WordsStudied))
	sb.WriteString(fmt.Sprintf("دوره‌های فعال: %d\n", upv.CoursesActive))

	if len(upv.Achievements) > 0 {
		sb.WriteString("\n🏆 *دستاوردها:*\n")
		for _, ach := range upv.Achievements {
			sb.WriteString(fmt.Sprintf("  - %s\n", tgmarkdown.Escape(ach.Title)))
		}
	}

	return sb.String()
}
