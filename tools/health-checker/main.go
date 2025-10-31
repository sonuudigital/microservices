package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: health-checker <url>")
		os.Exit(1)
	}
	url := os.Args[1]

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		os.Exit(0)
	}

	fmt.Fprintf(os.Stderr, "Received status code: %d\n", resp.StatusCode)
	os.Exit(1)
}
