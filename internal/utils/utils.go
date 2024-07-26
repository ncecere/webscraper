package utils

import (
	"log"
	"net/url"
	"regexp"
	"strings"
)

// RemoveFragment removes the fragment part of a URL
func RemoveFragment(urlStr string) string {
	if idx := strings.Index(urlStr, "#"); idx != -1 {
		return urlStr[:idx]
	}
	return urlStr
}

// ToAbsoluteURL converts a relative URL to an absolute URL
func ToAbsoluteURL(href string, baseURL *url.URL) string {
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

// SanitizeFilename removes or replaces characters that are unsafe for filenames
func SanitizeFilename(filename string) string {
	// Replace unsafe characters with underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9-_.]`)
	return reg.ReplaceAllString(filename, "_")
}
