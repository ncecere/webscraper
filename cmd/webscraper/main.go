package main

import (
	"fmt"
	"os"

	"github.com/ncecere/webscraper/internal/config"
)

func main() {
	if err := config.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
