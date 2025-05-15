package handlers

import (
	"embed"
	"fmt"
	"strings"
)

var md2Escaper = strings.NewReplacer(
	"_", "\\_",
	"*", "\\*",
	"[", "\\[",
	"]", "\\]",
	"(", "\\(",
	")", "\\)",
	"~", "\\~",
	"`", "\\`",
	">", "\\>",
	"#", "\\#",
	"+", "\\+",
	"-", "\\-",
	"=", "\\=",
	"|", "\\|",
	"{", "\\{",
	"}", "\\}",
	".", "\\.", // <— escape dot
	"!", "\\!",
)

func escapeMarkdownV2(text string) string {
	return md2Escaper.Replace(text)
}

//go:embed texts/*.txt
var textFS embed.FS

// Texts holds all loaded copy
var Texts = map[string]string{}

func init() {
	files := []string{"start", "start_learning", "return_to_main_menu"}
	for _, name := range files {
		data, err := textFS.ReadFile(fmt.Sprintf("texts/%s.txt", name))
		if err != nil {
			panic(fmt.Errorf("loading text %q: %w", name, err))
		}
		Texts[name] = escapeMarkdownV2(string(data))
	}
}
