package cmd

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/ncecere/webscraper/internal/config"
	"github.com/ncecere/webscraper/internal/scraper"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "webscraper",
		Short: "A web scraper that outputs markdown",
		Long:  `A web scraper that recursively scrapes websites and outputs the content in markdown format.`,
		Run: func(cmd *cobra.Command, args []string) {
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
			scraper.InitSemaphore(concurrentRequests)

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

			scraper.Wg.Add(1)
			go scraper.Scrape(ctx, startURL, outputPath, 0, baseURL, true)
			scraper.Wg.Wait()

			// Write external links to file
			scraper.WriteExternalLinksFile(outputPath)

			fmt.Println("Scraping completed")
		},
	}
)

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(config.InitConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default is $HOME/.webscraper.yaml)")
	rootCmd.Flags().String("start-url", "", "The URL to start scraping from")
	rootCmd.Flags().Int("max-depth", 3, "Maximum depth for recursive scraping")
	rootCmd.Flags().Int("concurrent-requests", 5, "Number of concurrent requests")
	rootCmd.Flags().String("output-path", "", "Path to save the scraped markdown files")
	rootCmd.Flags().Bool("scrape-external", false, "Whether to scrape external links")
	rootCmd.Flags().Int("external-depth", 1, "Maximum depth for external link scraping")

	viper.BindPFlag("start_url", rootCmd.Flags().Lookup("start-url"))
	viper.BindPFlag("max_depth", rootCmd.Flags().Lookup("max-depth"))
	viper.BindPFlag("concurrent_requests", rootCmd.Flags().Lookup("concurrent-requests"))
	viper.BindPFlag("output_path", rootCmd.Flags().Lookup("output-path"))
	viper.BindPFlag("scrape_external", rootCmd.Flags().Lookup("scrape-external"))
	viper.BindPFlag("external_depth", rootCmd.Flags().Lookup("external-depth"))
}
