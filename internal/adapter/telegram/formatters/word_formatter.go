package formatters

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
	"github.com/2000ostd/enssi-tel-bot/pkg/tgmarkdown"

	"strings"
)

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

// FormatWordForDisplay formats a domain Word entity into a MarkdownV2 string.
func FormatWordForDisplay(w word.Word) string {
	var mb strings.Builder

	// --- Header ---
	mb.WriteString(fmt.Sprintf(">*%s*\n", tgmarkdown.Escape(w.Title)))
	if w.Phonetic != "" {
		mb.WriteString(fmt.Sprintf("`%s`\n", tgmarkdown.Escape(w.Phonetic)))
	}
	mb.WriteString(tgmarkdown.Escape("─────────────────────────") + "\n")

	// --- Persian Meanings ---
	persianMeaningsByPOS := make(map[string][]string)
	for _, pos := range w.PartsOfSpeech {
		for _, meaning := range pos.Meanings {
			if strings.ToLower(meaning.Lang) == "fa" && strings.TrimSpace(meaning.Title) != "" {
				persianPOSTitle := translatePOS(pos.Title)
				persianMeaningsByPOS[persianPOSTitle] = append(persianMeaningsByPOS[persianPOSTitle], meaning.Title)
			}
		}
	}
	if len(persianMeaningsByPOS) > 0 {
		mb.WriteString("*معانی فارسی* :\n>\n")
		for posTitle, meanings := range persianMeaningsByPOS {
			mb.WriteString(fmt.Sprintf("> `[ %s ]` :\n", tgmarkdown.Escape(posTitle)))
			for _, m := range meanings {
				mb.WriteString(fmt.Sprintf(">  \\- %s\n", tgmarkdown.Escape(m)))
			}
			mb.WriteString("> \n")
		}
	}

	// --- Definitions ---
	if w.PrimaryDef != "" {
		mb.WriteString("*تعریف اصلی*:\n")
		mb.WriteString(fmt.Sprintf("> %s\n\n", tgmarkdown.Escape(w.PrimaryDef)))
	}
	if w.SecondaryDef != "" {
		mb.WriteString("*تعریف بلند*:\n")
		mb.WriteString(fmt.Sprintf("> %s\n\n", tgmarkdown.Escape(w.SecondaryDef)))
	}

	// --- English Meanings ---
	englishMeaningsByPOS := make(map[string][]string)
	for _, pos := range w.PartsOfSpeech {
		for _, meaning := range pos.Meanings {
			if (strings.ToLower(meaning.Lang) == "en" || meaning.Lang == "") && strings.TrimSpace(meaning.Title) != "" {
				englishMeaningsByPOS[pos.Title] = append(englishMeaningsByPOS[pos.Title], meaning.Title)
			}
		}
	}
	if len(englishMeaningsByPOS) > 0 {
		mb.WriteString("*معانی انگلیسی* :\n>\n")
		for posTitle, meanings := range englishMeaningsByPOS {
			mb.WriteString(fmt.Sprintf("> `[ %s ]` :\n", tgmarkdown.Escape(strings.ToLower(posTitle))))
			for _, m := range meanings {
				mb.WriteString(fmt.Sprintf("> \\- %s\n", tgmarkdown.Escape(m)))
			}
			mb.WriteString("> \n")
		}
	}

	finalStr := mb.String()
	if finalStr == "" {
		return tgmarkdown.Escape("محتوایی برای نمایش یافت نشد.")
	}
	return finalStr
}
