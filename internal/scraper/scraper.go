package scraper

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/viper"
)

var (
	Wg            sync.WaitGroup
	visitedURLs   sync.Map
	semaphore     chan struct{}
	externalLinks sync.Map
)

func InitSemaphore(concurrentRequests int) {
	semaphore = make(chan struct{}, concurrentRequests)
}

func removeFragment(urlStr string) string {
	if idx := strings.Index(urlStr, "#"); idx != -1 {
		return urlStr[:idx]
	}
	return urlStr
}

func Scrape(ctx context.Context, urlStr string, outputPath string, depth int, baseURL *url.URL, isInternal bool) {
	defer Wg.Done()

	// Remove fragment from URL for visiting and file naming
	urlWithoutFragment := removeFragment(urlStr)

	select {
	case <-ctx.Done():
		return
	case semaphore <- struct{}{}:
		defer func() { <-semaphore }()
	}

	// Check if URL (without fragment) has been visited
	if _, visited := visitedURLs.Load(urlWithoutFragment); visited {
		fmt.Printf("Already scraped: %s\n", urlWithoutFragment)
		return
	}

	fmt.Printf("Scraping: %s (depth: %d)\n", urlStr, depth)

	// Create a context with a timeout
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", urlStr, nil)
	if err != nil {
		log.Printf("Error creating request for %s: %v\n", urlStr, err)
		return
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching %s: %v\n", urlStr, err)
		return
	}
	defer response.Body.Close()

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Printf("Error parsing %s: %v\n", urlStr, err)
		return
	}

	content, err := document.Find("body").Html()
	if err != nil {
		log.Printf("Error extracting content from %s: %v\n", urlStr, err)
		return
	}

	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(content)
	if err != nil {
		log.Printf("Error converting to markdown for %s: %v\n", urlStr, err)
		return
	}

	// Create directory for the domain
	u, _ := url.Parse(urlWithoutFragment)
	domainDir := filepath.Join(outputPath, strings.Replace(u.Hostname(), ".", "_", -1))
	err = os.MkdirAll(domainDir, os.ModePerm)
	if err != nil {
		log.Printf("Error creating directory for %s: %v\n", u.Hostname(), err)
		return
	}

	// Use URL without fragment for file naming
	filename := strings.Replace(urlWithoutFragment, "://", "_", -1)
	filename = strings.Replace(filename, "/", "_", -1) + ".md"
	filePath := filepath.Join(domainDir, filename)
	err = os.WriteFile(filePath, []byte(markdown), 0644)
	if err != nil {
		log.Printf("Error writing file for %s: %v\n", urlWithoutFragment, err)
		return
	}

	// Mark the URL (without fragment) as visited
	visitedURLs.Store(urlWithoutFragment, true)

	maxDepth := viper.GetInt("max_depth")
	externalDepth := viper.GetInt("external_depth")
	scrapeExternal := viper.GetBool("scrape_external")

	if (isInternal && depth < maxDepth) || (!isInternal && depth < externalDepth) {
		document.Find("a").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if exists {
				absoluteURL := toAbsoluteURL(href, baseURL)
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
						if _, visited := visitedURLs.Load(removeFragment(absoluteURL)); !visited {
							Wg.Add(1)
							go Scrape(ctx, absoluteURL, outputPath, depth+1, baseURL, !isExternalLink)
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

func toAbsoluteURL(href string, baseURL *url.URL) string {
	u, err := url.Parse(href)
	if err != nil {
		log.Printf("Error parsing URL %s: %v\n", href, err)
		return ""
	}

	if !u.IsAbs() {
		u = baseURL.ResolveReference(u)
	}

	return u.String()
}

func WriteExternalLinksFile(outputPath string) {
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
		log.Printf("Error writing external links file: %v\n", err)
	}
}
