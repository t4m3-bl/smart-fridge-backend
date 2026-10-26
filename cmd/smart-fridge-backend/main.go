package main

import (
	"fmt"
	"os"
	"smart-fridge-backend/internal/app"
	"smart-fridge-backend/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	app.New(cfg).Run()
}
