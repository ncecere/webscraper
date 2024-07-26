package scraper

import (
	"fmt"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/ncecere/webscraper/internal/utils"
)

func convertToMarkdown(content *goquery.Selection) string {
	converter := md.NewConverter("", true, nil)

	// Configure the converter to skip converting images
	converter.AddRules(md.Rule{
		Filter: []string{"img"},
		Replacement: func(content string, selec *goquery.Selection, options *md.Options) *string {
			return md.String("")
		},
	})

	// Add custom rules for headings
	converter.AddRules(md.Rule{
		Filter: []string{"h1", "h2", "h3", "h4", "h5", "h6"},
		Replacement: func(content string, selec *goquery.Selection, options *md.Options) *string {
			class, _ := selec.Attr("class")
			level := 2 // default to h2
			switch class {
			case "h2":
				level = 2
			case "h3":
				level = 3
			case "h4":
				level = 4
			}
			return md.String(strings.Repeat("#", level) + " " + content)
		},
	})

	html, _ := content.Html()
	markdown, _ := converter.ConvertString(html)
	return markdown
}

func improveMarkdownStructure(markdown, pageTitle string) string {
	lines := strings.Split(markdown, "\n")
	var improvedLines []string
	var toc []string
	inCodeBlock := false

	improvedLines = append(improvedLines, "# "+pageTitle+"\n")
	improvedLines = append(improvedLines, "## Table of Contents\n")

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "```") {
			inCodeBlock = !inCodeBlock
		}
		if !inCodeBlock && (strings.HasPrefix(trimmedLine, "## ") || strings.HasPrefix(trimmedLine, "### ") || strings.HasPrefix(trimmedLine, "#### ")) {
			tocEntry := strings.TrimLeft(trimmedLine, "# ")
			toc = append(toc, "- ["+tocEntry+"](#"+utils.SanitizeAnchor(tocEntry)+")")
		}
		if trimmedLine != "" || inCodeBlock {
			improvedLines = append(improvedLines, line)
		}
	}

	// Insert table of contents
	improvedMarkdown := strings.Join(improvedLines[:2], "\n") + "\n" + strings.Join(toc, "\n") + "\n\n---\n\n" + strings.Join(improvedLines[2:], "\n")

	// Add timestamp and URL at the bottom
	improvedMarkdown += fmt.Sprintf("\n\n---\n\nScraped on %s", time.Now().Format(time.RFC3339))

	return improvedMarkdown
}
