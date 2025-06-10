package formatters

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
	"github.com/2000ostd/enssi-tel-bot/pkg/tgmarkdown"
	"strings"
)

// FormatWordForDisplay formats a domain Word entity into a MarkdownV2 string.
func FormatWordForDisplay(w word.Word) string {
	var mb strings.Builder

	// Header
	mb.WriteString(fmt.Sprintf(">*%s*\n", tgmarkdown.Escape(w.Title)))
	if w.Phonetic != "" {
		mb.WriteString(fmt.Sprintf("`%s`\n", tgmarkdown.Escape(w.Phonetic)))
	}
	mb.WriteString(tgmarkdown.Escape("─────────────────────────") + "\n")

	// Definitions
	if w.PrimaryDef != "" {
		mb.WriteString("*تعریف اصلی* : \n")
		mb.WriteString(fmt.Sprintf("> %s\n\n", tgmarkdown.Escape(w.PrimaryDef)))
	}
	if w.SecondaryDef != "" {
		mb.WriteString("*تعریف بلند* : \n")
		mb.WriteString(fmt.Sprintf("> %s\n\n", tgmarkdown.Escape(w.SecondaryDef)))
	}

	// Group meanings by POS
	var hasContent bool
	for _, pos := range w.PartsOfSpeech {
		hasContent = true
		mb.WriteString(fmt.Sprintf("`[ %s ]` :\n", tgmarkdown.Escape(pos.Title)))
		for _, m := range pos.Meanings {
			mb.WriteString(fmt.Sprintf("> \\- %s\n", tgmarkdown.Escape(m.Title)))
		}
		mb.WriteString("\n")
	}

	if !hasContent {
		mb.WriteString(tgmarkdown.Escape("محتوایی برای نمایش یافت نشد."))
	}

	return mb.String()
}
