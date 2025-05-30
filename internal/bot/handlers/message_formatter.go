package handlers

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"strings"
)

func formatWordMarkdown(word *models.Word, ws *models.WordSource) string {
	if word == nil || ws == nil {
		return ""
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
			if p.Title != "" {
				hasContent = true
				langTag := ""
				if p.Lang != "" {
					// Escape the parentheses that are part of our format string
					langTag = fmt.Sprintf(" \\(%s\\)", escapeMarkdownV2(p.Lang))
				}
				phoneticsBuilder.WriteString(fmt.Sprintf("  ▫️ %s%s\n", escapeMarkdownV2(p.Title), langTag))
			}
		}
		if hasContent {
			mb.WriteString("\n🗣️ *تلفظ / IPA:*\n")
			mb.WriteString(phoneticsBuilder.String())
		}
	}

	englishSection := "\n\n🇬🇧 \\=\\=\\= **ENGLISH MEANINGS** \\=\\=\\= 🇬🇧\n"
	persianSection := "\n\n🇮🇷 \\=\\=\\= ** معانی فارسی ** \\=\\=\\= 🇮🇷\n"

	// --- English Meanings Section ---
	hasEnglishContent := false
	var englishSectionBuilder strings.Builder
	for _, pos := range ws.PartsOfSpeeches {
		var currentPosEnMeanings []string
		for _, meaning := range pos.Meanings {
			if meaning.Title != "" && (strings.ToLower(meaning.Lang) == "en" || meaning.Lang == "") {
				currentPosEnMeanings = append(currentPosEnMeanings, escapeMarkdownV2(meaning.Title))
			}
		}
		if len(currentPosEnMeanings) > 0 {
			if !hasEnglishContent {
				mb.WriteString(englishSection)
				hasEnglishContent = true
			}
			englishSectionBuilder.WriteString(fmt.Sprintf("\n  🏷️ **%s**\n\n", escapeMarkdownV2(pos.Title)))
			for _, mText := range currentPosEnMeanings {
				englishSectionBuilder.WriteString(fmt.Sprintf("    💡 %s\n", mText))
			}
		}
	}
	if hasEnglishContent {
		mb.WriteString(englishSectionBuilder.String())
	}

	// --- Persian Meanings Section ---
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
				mb.WriteString(persianSection)
				hasPersianContent = true
			}
			persianPOSTitle := translatePOS(pos.Title)
			englishPOSTitleEscaped := escapeMarkdownV2(pos.Title)
			// Escape the parentheses that are part of our format string
			persianSectionBuilder.WriteString(fmt.Sprintf("\n  🏷️ *%s:\\(%s\\)*\n\n", escapeMarkdownV2(persianPOSTitle), englishPOSTitleEscaped))
			for _, mText := range currentPosFaMeanings {
				persianSectionBuilder.WriteString(fmt.Sprintf("    💡 %s\n", mText))
			}
		}
	}
	if hasPersianContent {
		mb.WriteString(persianSectionBuilder.String())
	}

	return mb.String()
}
