package scraper

import (
	"sync"
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
