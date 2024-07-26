Web Scraper Application
=======================

This command-line web scraper recursively scrapes websites and outputs the content in markdown format.

Installation:
-------------
1. Ensure you have Go installed on your system (version 1.16 or later).
2. Clone the repository:
   git clone https://github.com/yourusername/webscraper.git
3. Navigate to the project directory:
   cd webscraper
4. Build the binary:
   go build -o webscraper cmd/webscraper/main.go

Usage:
------
./webscraper [flags]

Flags:
------
--start-url            The URL to start scraping from (required)
--max-depth            Maximum depth for recursive scraping (default: 3)
--concurrent-requests  Number of concurrent requests (default: 5)
--output-path          Path to save the scraped markdown files (default: current directory)
--scrape-external      Whether to scrape external links (default: false)
--external-depth       Maximum depth for external link scraping (default: 1)
--config               Path to a custom config file

Examples:
---------
1. Basic usage with only the required start URL:
   ./webscraper --start-url https://example.com

2. Set maximum depth for recursive scraping:
   ./webscraper --start-url https://example.com --max-depth 5

3. Set the number of concurrent requests:
   ./webscraper --start-url https://example.com --concurrent-requests 10

4. Specify an output path for scraped files:
   ./webscraper --start-url https://example.com --output-path /path/to/output

5. Enable scraping of external links:
   ./webscraper --start-url https://example.com --scrape-external

6. Set the maximum depth for external link scraping:
   ./webscraper --start-url https://example.com --scrape-external --external-depth 2

7. Use a custom configuration file:
   ./webscraper --config /path/to/config.yaml

8. Combine multiple flags:
   ./webscraper --start-url https://example.com --max-depth 4 --concurrent-requests 8 --output-path ./scraped_content --scrape-external --external-depth 2

Output:
-------
The scraper creates a directory structure as follows:
output_path/
├── domain1_com/
│   ├── page1.md
│   ├── page2.md
│   └── ...
├── domain2_com/
│   ├── page1.md
│   ├── page2.md
│   └── ...
└── external_links.md

Each scraped page is saved as a separate markdown file within a directory named after its domain.
The external_links.md file contains a list of all external links encountered during scraping.

Configuration File:
-------------------
You can use a YAML configuration file instead of command-line flags. The default location is $HOME/.webscraper.yaml.
Example config file contents:

start_url: "https://example.com"
max_depth: 5
concurrent_requests: 10
output_path: "/path/to/output"
scrape_external: true
external_depth: 2

To use a custom config file:
./webscraper --config /path/to/your/config.yaml

Notes:
------
- The scraper respects the robots.txt file of the websites it scrapes.
- Be mindful of the websites you're scraping and ensure you have permission to do so.
- Scraping too aggressively might put unnecessary load on web servers or get your IP blocked.
- When using both command-line flags and a config file, command-line flags take precedence.

After scraping, the application will display:
- The number of documents created
- The number of URLs scanned
- The total time taken for the scraping process

Example output:
---------------
Scraping completed
Documents created: 150
URLs scanned: 200
Total time: 2m30s