package scraper

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/ncecere/webscraper/internal/utils"
	"github.com/spf13/viper"
)

func handleLinks(ctx context.Context, document *goquery.Document, urlStr string, outputPath string, depth int, baseURL *url.URL, isInternal bool) {
	maxDepth := viper.GetInt("max_depth")
	externalDepth := viper.GetInt("external_depth")
	scrapeExternal := viper.GetBool("scrape_external")

	if (isInternal && depth < maxDepth) || (!isInternal && depth < externalDepth) {
		document.Find("a").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if exists {
				// Skip mailto and tel links
				if strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") {
					return
				}

				absoluteURL := utils.ToAbsoluteURL(href, baseURL)
				if absoluteURL != "" {
					parsedURL, _ := url.Parse(absoluteURL)
					isExternalLink := parsedURL.Hostname() != baseURL.Hostname()

					if isExternalLink {
						// Add to external links
						links, _ := externalLinks.LoadOrStore(baseURL.Hostname(), &sync.Map{})
						links.(*sync.Map).Store(parsedURL.Hostname(), absoluteURL)
					}

					if !isExternalLink || (isExternalLink && scrapeExternal) {
						fmt.Printf("Found link: %s\n", absoluteURL)
						// Check if URL (without fragment) has been visited
						if _, visited := visitedURLs.Load(utils.RemoveFragment(absoluteURL)); !visited {
							wg.Add(1)
							go scrapePage(ctx, absoluteURL, outputPath, depth+1, baseURL, !isExternalLink)
						} else {
							fmt.Printf("Already queued or visited: %s\n", absoluteURL)
						}
					} else {
						fmt.Printf("Skipping external link: %s\n", absoluteURL)
					}
				} else {
					fmt.Printf("Skipping invalid link: %s\n", href)
				}
			}
		})
	}
}

func writeExternalLinksFile(outputPath string) {
	content := ""
	externalLinks.Range(func(key, value interface{}) bool {
		domain := key.(string)
		links := value.(*sync.Map)
		content += fmt.Sprintf("%s:\n", domain)
		links.Range(func(linkDomain, linkURL interface{}) bool {
			content += fmt.Sprintf("- %s\n", linkURL)
			return true
		})
		content += "\n"
		return true
	})

	filePath := filepath.Join(outputPath, "external_links.md")
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		fmt.Printf("Error writing external links file: %v\n", err)
	}
}
