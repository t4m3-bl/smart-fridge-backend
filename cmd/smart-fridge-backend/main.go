package main

import (
	"fmt"
	"os"
	"smart-fridge-backend/internal/config"
)

func main() {
	_, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
}
