package main

import (
	"log"
	"os"

	"fyne.io/fyne/v2/app"
	"github.com/joho/godotenv"
	"github.com/memleak-io/gpt-pdf/internal/ui"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}

	// Check for OpenAI API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	// Create and run application
	application := app.New()
	window := application.NewWindow("GPT PDF Reader")
	
	ui.NewMainWindow(window)
	window.Show()
	
	application.Run()
}