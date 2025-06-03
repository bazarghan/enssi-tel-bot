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
func (s *Service) formatWordForDisplay(word *models.Word, ws *models.WordSource) (string, error) {
	if word == nil || ws == nil {
		return "", fmt.Errorf("%w: word or word source data is nil", ErrFormattingFailed)
	}

	var mb strings.Builder

	// 1. Word Title
	if word.Title != "" {
		mb.WriteString(fmt.Sprintf("🔤 **%s**\n", escapeMarkdownV2(strings.ToUpper(word.Title))))
	}

	// 2. General Definitions (Primary/Secondary)
	if ws.DefPrimary != "" || ws.DefSecondary != "" {
		mb.WriteString("\n📋 *تعاریف اصلی:*\n")
		if ws.DefPrimary != "" {
			mb.WriteString(fmt.Sprintf("  ▪️ %s\n\n", escapeMarkdownV2(ws.DefPrimary)))
		}
		if ws.DefSecondary != "" {
			mb.WriteString(fmt.Sprintf("  ▪️ %s\n\n", escapeMarkdownV2(ws.DefSecondary)))
		}
	}

	// 3. Phonetics
	if len(ws.Phonetics) > 0 {
		hasContent := false
		var phoneticsBuilder strings.Builder
		for _, p := range ws.Phonetics {
			if p.Title != "" { // p.Title is the IPA string
				hasContent = true
				langTag := ""
				if p.Lang != "" {
					langTag = fmt.Sprintf(" \\(%s\\)", escapeMarkdownV2(p.Lang)) // IPA lang (US, UK)
				}
				phoneticsBuilder.WriteString(fmt.Sprintf("  ▫️ %s%s\n", escapeMarkdownV2(p.Title), langTag))
			}
		}
		if hasContent {
			mb.WriteString("\n🗣️ *تلفظ / IPA:*\n")
			mb.WriteString(phoneticsBuilder.String())
		}
	}

	// --- English Meanings Section ---
	englishSectionHeader := "\n\n🇬🇧 \\=\\=\\= **ENGLISH MEANINGS** \\=\\=\\= 🇬🇧\n"
	hasEnglishContent := false
	var englishSectionBuilder strings.Builder

	for _, pos := range ws.PartsOfSpeeches {
		var currentPosEnMeanings []string
		for _, meaning := range pos.Meanings {
			// Assuming blank lang or "en" means English
			if meaning.Title != "" && (strings.ToLower(meaning.Lang) == "en" || meaning.Lang == "") {
				currentPosEnMeanings = append(currentPosEnMeanings, escapeMarkdownV2(meaning.Title))
			}
		}
		if len(currentPosEnMeanings) > 0 {
			if !hasEnglishContent {
				englishSectionBuilder.WriteString(englishSectionHeader)
				hasEnglishContent = true
			}
			englishSectionBuilder.WriteString(fmt.Sprintf("\n  🏷️ **%s**\n\n", escapeMarkdownV2(pos.Title))) // POS Title
			for _, mText := range currentPosEnMeanings {
				englishSectionBuilder.WriteString(fmt.Sprintf("    💡 %s\n", mText))
			}
		}
	}
	if hasEnglishContent {
		mb.WriteString(englishSectionBuilder.String())
	}

	// --- Persian Meanings Section ---
	persianSectionHeader := "\n\n🇮🇷 \\=\\=\\= **معانی فارسی** \\=\\=\\= 🇮🇷\n"
	hasPersianContent := false
	var persianSectionBuilder strings.Builder

	for _, pos := range ws.PartsOfSpeeches {
		var currentPosFaMeanings []string
		for _, meaning := range pos.Meanings {
			if meaning.Title != "" && strings.ToLower(meaning.Lang) == "fa" {
				currentPosFaMeanings = append(currentPosFaMeanings, escapeMarkdownV2(meaning.Title))
			}
		}
		if len(currentPosFaMeanings) > 0 {
			if !hasPersianContent {
				persianSectionBuilder.WriteString(persianSectionHeader)
				hasPersianContent = true
			}
			persianPOSTitle := translatePOS(pos.Title)
			englishPOSTitleEscaped := escapeMarkdownV2(pos.Title)
			persianSectionBuilder.WriteString(fmt.Sprintf("\n  🏷️ *%s \\(%s\\):*\n\n", escapeMarkdownV2(persianPOSTitle), englishPOSTitleEscaped))
			for _, mText := range currentPosFaMeanings {
				persianSectionBuilder.WriteString(fmt.Sprintf("    💡 %s\n", mText))
			}
		}
	}
	if hasPersianContent {
		mb.WriteString(persianSectionBuilder.String())
	}

	if mb.Len() == 0 {
		return "", fmt.Errorf("%w: no content generated for word ID %d", ErrFormattingFailed, word.ID)
	}

	return mb.String(), nil
}
