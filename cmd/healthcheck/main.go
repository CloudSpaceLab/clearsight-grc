package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func check(client *http.Client, target string) error {
	response, err := client.Get(target)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("readiness returned %d", response.StatusCode)
	}
	return nil
}

func main() {
	target := "http://127.0.0.1:13281/health/ready"
	if len(os.Args) == 2 {
		target = os.Args[1]
	}
	if err := check(&http.Client{Timeout: 5 * time.Second}, target); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
