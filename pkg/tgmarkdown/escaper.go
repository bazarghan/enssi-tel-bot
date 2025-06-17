package tgmarkdown

import "strings"

var mdV2Escaper = strings.NewReplacer(
	"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)",
	"~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
	"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!",
)

// Escape ensures text is safe for Telegram's MarkdownV2 mode.
func Escape(text string) string {
	if text == "" {
		return ""
	}
	return mdV2Escaper.Replace(text)
}
