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
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ncecere/webscraper/internal/utils"
)

func scrapePage(ctx context.Context, urlStr string, outputPath string, depth int, baseURL *url.URL, isInternal bool) {
	defer wg.Done()

	// Remove fragment from URL for visiting and file naming
	urlWithoutFragment := utils.RemoveFragment(urlStr)

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

	// Increment URLs scanned counter
	mutex.Lock()
	urlsScanned++
	mutex.Unlock()

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

	// Extract the page title
	pageTitle := document.Find("title").Text()

	// Process the page content
	processedContent := processPageContent(document)

	// Convert to markdown
	markdown := convertToMarkdown(processedContent)

	// Improve markdown structure
	improvedMarkdown := improveMarkdownStructure(markdown, pageTitle)

	// Create directory for the domain
	u, _ := url.Parse(urlWithoutFragment)
	domainDir := filepath.Join(outputPath, utils.SanitizeFilename(u.Hostname()))
	err = os.MkdirAll(domainDir, os.ModePerm)
	if err != nil {
		log.Printf("Error creating directory for %s: %v\n", u.Hostname(), err)
		return
	}

	// Use URL without fragment for file naming
	filename := utils.SanitizeFilename(urlWithoutFragment) + ".md"
	filePath := filepath.Join(domainDir, filename)
	err = os.WriteFile(filePath, []byte(improvedMarkdown), 0644)
	if err != nil {
		log.Printf("Error writing file for %s: %v\n", urlWithoutFragment, err)
		return
	}

	// Increment documents created counter
	mutex.Lock()
	documentsCreated++
	mutex.Unlock()

	// Mark the URL (without fragment) as visited
	visitedURLs.Store(urlWithoutFragment, true)

	// Handle links
	handleLinks(ctx, document, urlStr, outputPath, depth, baseURL, isInternal)
}

func processPageContent(document *goquery.Document) *goquery.Selection {
	// Remove common navigation elements and menus
	document.Find("nav, .nav, .navbar, .menu, .navigation, header, footer, .sidebar, #sidebar, .skip-link, .skip-to-content").Remove()

	// Remove elements with role="navigation"
	document.Find("[role='navigation']").Remove()

	// Remove elements with aria-label containing "navigation" or "menu"
	document.Find("[aria-label]").Each(func(i int, s *goquery.Selection) {
		if ariaLabel, exists := s.Attr("aria-label"); exists {
			if strings.Contains(strings.ToLower(ariaLabel), "navigation") || strings.Contains(strings.ToLower(ariaLabel), "menu") {
				s.Remove()
			}
		}
	})

	// Remove "Skip to main content" links
	document.Find("a").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists && strings.Contains(href, "#content") {
			s.Remove()
		}
	})

	// Convert headings to make them subordinate to the page title
	document.Find("h1").Each(func(i int, s *goquery.Selection) {
		s.SetAttr("class", "h2")
	})
	document.Find("h2").Each(func(i int, s *goquery.Selection) {
		s.SetAttr("class", "h3")
	})
	document.Find("h3, h4, h5, h6").Each(func(i int, s *goquery.Selection) {
		s.SetAttr("class", "h4")
	})

	// Find the main content
	mainContent := document.Find("main, #main, .main, [role='main']")
	if mainContent.Length() == 0 {
		// If no main content found, use the body
		mainContent = document.Find("body")
	}

	return mainContent
}
