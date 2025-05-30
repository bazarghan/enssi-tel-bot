package handlers

import (
	"strings"
)

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
