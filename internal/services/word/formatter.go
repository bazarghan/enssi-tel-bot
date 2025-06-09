package word

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"strings"
)

// mdV2Escaper is the Telegram MarkdownV2 escaper.
var mdV2Escaper = strings.NewReplacer(
	"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)",
	"~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
	"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!",
)

func escapeMarkdownV2(text string) string {
	if text == "" {
		return ""
	}
	return mdV2Escaper.Replace(text)
}

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

func translatePOS(posTitleEng string) string {
	titleLower := strings.ToLower(posTitleEng)
	if translated, ok := posTranslationMap[titleLower]; ok {
		return translated
	}
	return posTitleEng
}

// formatWordForDisplay takes the core word models and formats them into a Markdown string.
// This is an unexported method as it's a helper for GetWordDetailsForCourse.

// formatWordForDisplay takes the core word models and formats them into a new,
// structured MarkdownV2 string with quote blocks for each section.
func (s *Service) formatWordForDisplay(word *models.Word, ws *models.WordSource) (string, error) {
	if word == nil || ws == nil {
		return "", fmt.Errorf("%w: word or word source data is nil", ErrFormattingFailed)
	}

	var mb strings.Builder

	// --- Header ---
	// Word Title (Bold)
	mb.WriteString(fmt.Sprintf(">*%s*\n", escapeMarkdownV2(word.Title)))
	// Phonetics (Code block) - we'll take the first available one
	if len(ws.Phonetics) > 0 && ws.Phonetics[0].Title != "" {
		mb.WriteString(fmt.Sprintf("`%s`\n", escapeMarkdownV2(ws.Phonetics[0].Title)))
	}
	// Separator
	mb.WriteString(escapeMarkdownV2("─────────────────────────") + "\n")

	// --- Persian Meanings Block ---
	persianMeaningsByPOS := make(map[string][]string)
	for _, pos := range ws.PartsOfSpeeches {
		for _, meaning := range pos.Meanings {
			if strings.ToLower(meaning.Lang) == "fa" && strings.TrimSpace(meaning.Title) != "" {
				persianPOSTitle := translatePOS(pos.Title) // Translate "noun" to "اسم"
				persianMeaningsByPOS[persianPOSTitle] = append(persianMeaningsByPOS[persianPOSTitle], meaning.Title)
			}
		}
	}
	if len(persianMeaningsByPOS) > 0 {
		mb.WriteString("*معانی فارسی* :\n")
		mb.WriteString(">\n") // Zero-width space for an empty quoted line
		for posTitle, meanings := range persianMeaningsByPOS {
			mb.WriteString(fmt.Sprintf("> `[ %s ]` :\n", escapeMarkdownV2(posTitle)))
			for _, m := range meanings {
				mb.WriteString(fmt.Sprintf(">  \\- %s\n", escapeMarkdownV2(m)))
			}
			mb.WriteString(">  \n")
		}
		mb.WriteString("\n") // Space after the block
	}

	// --- Primary Definition Block ---
	if strings.TrimSpace(ws.DefPrimary) != "" {
		mb.WriteString("*تعریف اصلی* : \n")
		mb.WriteString("> \u200b\n")
		for _, line := range strings.Split(ws.DefPrimary, "\n") {
			mb.WriteString(fmt.Sprintf("> %s\n", escapeMarkdownV2(line)))

			mb.WriteString(">  \n")
		}
		mb.WriteString("\n")
	}

	// --- Secondary/Long Definition Block ---
	if strings.TrimSpace(ws.DefSecondary) != "" {
		mb.WriteString("*تعریف بلند* : \n")
		mb.WriteString("> \n")
		for _, line := range strings.Split(ws.DefSecondary, "\n") {
			mb.WriteString(fmt.Sprintf("> %s\n", escapeMarkdownV2(line)))

			mb.WriteString(">  \n")

		}
		mb.WriteString("\n")
	}

	// --- English Meanings Block ---
	englishMeaningsByPOS := make(map[string][]string)
	for _, pos := range ws.PartsOfSpeeches {
		// Group English meanings (lang="en" or empty)
		for _, meaning := range pos.Meanings {
			if (strings.ToLower(meaning.Lang) == "en" || meaning.Lang == "") && strings.TrimSpace(meaning.Title) != "" {
				englishMeaningsByPOS[pos.Title] = append(englishMeaningsByPOS[pos.Title], meaning.Title)
			}
		}
	}
	if len(englishMeaningsByPOS) > 0 {
		mb.WriteString("*معانی انگلیسی* :\n")
		mb.WriteString(">\n")
		for posTitle, meanings := range englishMeaningsByPOS {
			mb.WriteString(fmt.Sprintf("> `[ %s ]` :\n", escapeMarkdownV2(strings.ToLower(posTitle))))
			for _, m := range meanings {
				mb.WriteString(fmt.Sprintf("> \\- %s\n", escapeMarkdownV2(m)))

			}
			mb.WriteString(">  \n")
		}
		mb.WriteString("\\.\n")
	}

	if mb.Len() == 0 {
		return "", fmt.Errorf("%w: no content generated for word ID %d", ErrFormattingFailed, word.ID)
	}

	return mb.String(), nil
}
