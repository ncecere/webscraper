package config

import (
	"fmt"
	"os"

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
		Run:   runScraper,
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.webscraper.yaml)")
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

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".webscraper")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func runScraper(cmd *cobra.Command, args []string) {
	scraper.Run()
}
