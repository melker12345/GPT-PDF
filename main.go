package main

import (
	"fmt"
	"log"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/joho/godotenv"
	"pdf-reader/internal/ui"
)

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}
}

func main() {
	// Set up error logging
	logFile, err := os.Create("app.log")
	if err != nil {
		fmt.Printf("Error creating log file: %v\n", err)
		return
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Create and run the application
	application := app.New()
	if application == nil {
		log.Fatal("Failed to create application")
		return
	}

	window := application.NewWindow("PDF Reader with GPT")
	if window == nil {
		log.Fatal("Failed to create window")
		return
	}

	// Set minimum window size
	window.Resize(fyne.NewSize(1000, 600))
	window.SetFixedSize(false)

	mainWindow := ui.NewMainWindow(window)
	mainWindow.Setup()

	window.ShowAndRun()
}