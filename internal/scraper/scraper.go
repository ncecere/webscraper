package scraper

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"
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
	go scrapePage(ctx, startURL, outputPath, 0, baseURL, true)
	wg.Wait()

	// Write external links to file
	writeExternalLinksFile(outputPath)

	elapsedTime := time.Since(startTime)

	fmt.Println("Scraping completed")
	fmt.Printf("Documents created: %d\n", documentsCreated)
	fmt.Printf("URLs scanned: %d\n", urlsScanned)
	fmt.Printf("Total time: %s\n", elapsedTime)
}
