# Web Scraper

This is a command-line web scraper that recursively scrapes websites and outputs the content in markdown format.

## Installation

1. Ensure you have Go installed on your system.
2. Clone the repository:
   git clone https://github.com/yourusername/webscraper.git
3. Navigate to the project directory:
   cd webscraper
4. Build the binary:
   go build -o webscraper cmd/webscraper/main.go

## Usage

The basic syntax for using the web scraper is:

`webscraper [flags]`

### Flags

- --start-url: The URL to start scraping from (required)
- --max-depth: Maximum depth for recursive scraping (default 3)
- --concurrent-requests: Number of concurrent requests (default 5)
- --output-path: Path to save the scraped markdown files (default: current directory)
- --scrape-external: Whether to scrape external links (default false)
- --external-depth: Maximum depth for external link scraping (default 1)
- --config: Path to a custom config file

## Examples

1. Scrape a website with default settings:
   webscraper --start-url https://example.com

2. Scrape a website with a maximum depth of 5:
   webscraper --start-url https://example.com --max-depth 5

3. Scrape a website and its external links:
   webscraper --start-url https://example.com --scrape-external

4. Scrape a website with 10 concurrent requests:
   webscraper --start-url https://example.com --concurrent-requests 10

5. Scrape a website and save the output to a specific directory:
   webscraper --start-url https://example.com --output-path /path/to/output

6. Use a custom configuration file:
   webscraper --config /path/to/config.yaml

## Functionality

The web scraper performs the following tasks:

1. Starts from the provided URL and recursively scrapes linked pages up to the specified maximum depth.
2. Converts the HTML content of each page to markdown format.
3. Saves each scraped page as a separate markdown file in a directory structure based on the domain.
4. Optionally scrapes external links (links to different domains) up to a specified depth.
5. Collects all external links and saves them in a separate markdown file.
6. Supports concurrent scraping to improve performance.

### Output

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

Each scraped page is saved as a separate markdown file within a directory named after its domain. The external_links.md file contains a list of all external links encountered during scraping.

## Configuration File

You can use a YAML configuration file instead of command-line flags. The default location for this file is $HOME/.webscraper.yaml. Here's an example of the configuration file contents:

start_url: "https://example.com"
max_depth: 5
concurrent_requests: 10
output_path: "/path/to/output"
scrape_external: true
external_depth: 2

## Notes

- The scraper respects the robots.txt file of the websites it scrapes.
- Be mindful of the websites you're scraping and ensure you have permission to do so.
- Scraping too aggressively might put unnecessary load on web servers or get your IP blocked.