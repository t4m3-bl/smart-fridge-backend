package main

import (
	"fmt"
	"os"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/workerapp"
)

func main() {
	cfg, err := config.LoadWorker()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "worker config:", err)
		os.Exit(1)
	}
	workerapp.New(cfg).Run()
}
