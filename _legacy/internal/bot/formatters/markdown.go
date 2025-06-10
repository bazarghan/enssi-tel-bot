package formatters

import "strings"

var mdV2Escaper = strings.NewReplacer(
	"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)",
	"~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
	"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!",
)

// EscapeMarkdownV2 ensures text is safe for Telegram's MarkdownV2 mode.
func EscapeMarkdownV2(text string) string {
	if text == "" {
		return ""
	}
	return mdV2Escaper.Replace(text)
}
