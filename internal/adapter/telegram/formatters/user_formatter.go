package formatters

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	"github.com/2000ostd/enssi-tel-bot/pkg/tgmarkdown"
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

	// Achievement formatting will be added in a later slice.

	return sb.String()
}
