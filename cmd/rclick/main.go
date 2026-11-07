package main

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/bambuo/rclick-go/internal/app"
	"github.com/bambuo/rclick-go/internal/config"
)

func main() {
	setupLogging()
	log.Printf("RClick %s starting", app.Version)

	store, err := config.Open("")
	if err != nil {
		log.Printf("configuration: %v", err)
	}

	if err := app.New(store).Run(); err != nil {
		log.Fatal(err)
	}
	log.Printf("RClick stopped")
}

func setupLogging() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	logDir := config.LogDir()
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return
	}
	file, err := os.OpenFile(filepath.Join(logDir, "rclick.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	log.SetOutput(io.MultiWriter(os.Stderr, file))
}
