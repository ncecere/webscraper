package scraper

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/ncecere/webscraper/internal/utils"
	"github.com/spf13/viper"
)

var (
	visitedURLs      sync.Map
	wg               sync.WaitGroup
	semaphore        chan struct{}
	externalLinks    sync.Map
	documentsCreated int
	urlsScanned      int
	mutex            sync.Mutex
)

func Run() {
	startTime := time.Now()

	startURL := viper.GetString("start_url")
	if startURL == "" {
		fmt.Println("Please provide a start URL using --start-url flag or in the config file")
		return
	}

	outputPath := viper.GetString("output_path")
	if outputPath == "" {
		outputPath = "." // Default to current directory
	}

	baseURL, err := url.Parse(startURL)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the semaphore for concurrent requests
	concurrentRequests := viper.GetInt("concurrent_requests")
	semaphore = make(chan struct{}, concurrentRequests)

	fmt.Printf("Starting URL: %s\n", startURL)
	fmt.Printf("Maximum depth: %d\n", viper.GetInt("max_depth"))
	fmt.Printf("Concurrent requests: %d\n", concurrentRequests)
	fmt.Printf("Output path: %s\n", outputPath)
	fmt.Printf("Scrape external links: %v\n", viper.GetBool("scrape_external"))
	fmt.Printf("External links depth: %d\n", viper.GetInt("external_depth"))

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nReceived an interrupt, stopping services...")
		cancel()
	}()

	wg.Add(1)
	go scrape(ctx, startURL, outputPath, 0, baseURL, true)
	wg.Wait()

	// Write external links to file
	writeExternalLinksFile(outputPath)

	elapsedTime := time.Since(startTime)

	fmt.Println("Scraping completed")
	fmt.Printf("Documents created: %d\n", documentsCreated)
	fmt.Printf("URLs scanned: %d\n", urlsScanned)
	fmt.Printf("Total time: %s\n", elapsedTime)
}

func scrape(ctx context.Context, urlStr string, outputPath string, depth int, baseURL *url.URL, isInternal bool) {
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

	content, err := mainContent.Html()
	if err != nil {
		log.Printf("Error extracting content from %s: %v\n", urlStr, err)
		return
	}

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

	markdown, err := converter.ConvertString(content)
	if err != nil {
		log.Printf("Error converting to markdown for %s: %v\n", urlStr, err)
		return
	}

	// Improve markdown structure
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
	improvedMarkdown += fmt.Sprintf("\n\n---\n\nScraped from [%s](%s) on %s", urlStr, urlStr, time.Now().Format(time.RFC3339))

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
							go scrape(ctx, absoluteURL, outputPath, depth+1, baseURL, !isExternalLink)
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
		log.Printf("Error writing external links file: %v\n", err)
	}
}
