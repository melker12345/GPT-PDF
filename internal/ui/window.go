package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"pdf-reader/internal/gpt"
	"pdf-reader/internal/pdf"
)

type MainWindow struct {
	window      fyne.Window
	pdfDisplay  *canvas.Image
	pageText    *widget.TextGrid
	gptView     *widget.Entry
	gptInput    *widget.Entry
	chatHistory []string
	pageNum     int
	zoomLevel   float64
	pdfHandler  *pdf.Handler
	gptHandler  *gpt.Handler
	statusBar   *widget.Label
	pageLabel   *widget.Label
	zoomLabel   *widget.Label
	scrollPDF   *container.Scroll
}

func NewMainWindow(window fyne.Window) *MainWindow {
	return &MainWindow{
		window:      window,
		pdfDisplay:  canvas.NewImageFromImage(nil),
		pageText:    widget.NewTextGrid(),
		gptView:     widget.NewMultiLineEntry(),
		gptInput:    widget.NewEntry(),
		chatHistory: make([]string, 0),
		pageNum:     1,
		zoomLevel:   1.0,
		pdfHandler:  pdf.NewHandler(),
		gptHandler:  gpt.NewHandler(),
		statusBar:   widget.NewLabel("No PDF loaded"),
		pageLabel:   widget.NewLabel("Page: -"),
		zoomLabel:   widget.NewLabel("Zoom: 100%"),
	}
}

func (m *MainWindow) prevPage() {
	if m.pageNum > 1 {
		m.pageNum--
		m.updatePageDisplay()
	}
}

func (m *MainWindow) nextPage() {
	if m.pageNum < m.pdfHandler.NumPages() {
		m.pageNum++
		m.updatePageDisplay()
	}
}

func (m *MainWindow) analyzePage() {
	if m.pdfHandler == nil {
		return
	}

	pageInfo, err := m.pdfHandler.GetPage(m.pageNum, m.zoomLevel)
	if err != nil {
		dialog.ShowError(err, m.window)
		return
	}

	// Use GPT to analyze the text
	analysis, err := m.gptHandler.AnalyzePage(pageInfo.Text)
	if err != nil {
		dialog.ShowError(err, m.window)
		return
	}

	m.gptView.SetText(analysis)
}

func (m *MainWindow) adjustZoom(delta float64) {
	newZoom := m.zoomLevel + delta
	if newZoom >= 0.25 && newZoom <= 5.0 {
		m.zoomLevel = newZoom
		m.updateZoomLabel()
		m.updatePageDisplay()
	}
}

func (m *MainWindow) Setup() {
	// Create toolbar
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.DocumentIcon(), func() {
			dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowError(err, m.window)
					return
				}
				if reader == nil {
					return
				}
				defer reader.Close()
				
				if err := m.pdfHandler.LoadPDF(reader.URI().Path()); err != nil {
					dialog.ShowError(err, m.window)
					return
				}
				m.pageNum = 1
				m.zoomLevel = 1.0
				m.updateZoomLabel()
				m.updatePageDisplay()
			}, m.window)
		}),
	)

	// Navigation controls
	prevBtn := widget.NewButton("Previous", m.prevPage)
	nextBtn := widget.NewButton("Next", m.nextPage)
	
	// Add zoom controls with finer adjustments
	zoomOutBtn := widget.NewButton("-", func() { m.adjustZoom(-0.1) })
	zoomInBtn := widget.NewButton("+", func() { m.adjustZoom(0.1) })
	
	// Add preset zoom levels
	zoomPresets := widget.NewSelect([]string{"50%", "75%", "100%", "125%", "150%", "200%"}, func(value string) {
		var zoom float64
		switch value {
		case "50%":
			zoom = 0.5
		case "75%":
			zoom = 0.75
		case "100%":
			zoom = 1.0
		case "125%":
			zoom = 1.25
		case "150%":
			zoom = 1.5
		case "200%":
			zoom = 2.0
		}
		if zoom != m.zoomLevel {
			m.zoomLevel = zoom
			m.updateZoomLabel()
			m.updatePageDisplay()
		}
	})
	zoomPresets.SetSelected("100%")
	
	// Combine all controls into one row
	navControls := container.NewHBox(
		prevBtn, 
		m.pageLabel, 
		nextBtn,
		widget.NewSeparator(),
		m.zoomLabel,
		zoomOutBtn,
		zoomPresets,
		zoomInBtn,
		widget.NewSeparator(),
		m.statusBar,
	)

	// Combine toolbar and navigation into one row
	topControls := container.NewHBox(
		toolbar,
		container.NewPadded(navControls),
	)

	// PDF display area
	m.pdfDisplay.FillMode = canvas.ImageFillContain
	m.pdfDisplay.SetMinSize(fyne.NewSize(600, 800))
	
	// Create a container to center the PDF display
	pdfContainer := container.NewCenter(m.pdfDisplay)
	
	// Create the main PDF content with centered display
	pdfContent := container.NewVBox(
		pdfContainer,
		m.pageText,
	)
	
	// Create a scroll container for the PDF content
	m.scrollPDF = container.NewScroll(pdfContent)

	// Set up GPT input and chat
	m.gptView.MultiLine = true
	m.gptView.Wrapping = fyne.TextWrapWord
	m.gptView.TextStyle = fyne.TextStyle{Monospace: true}
	
	m.gptInput.SetPlaceHolder("Ask a question about the PDF...")
	m.gptInput.OnSubmitted = func(q string) {
		if q == "" {
			return
		}
		
		// Add user question to chat
		m.appendToChat("You: " + q)
		
		// Get GPT response
		response, err := m.gptHandler.AskQuestion(q, m.chatHistory)
		if err != nil {
			dialog.ShowError(err, m.window)
			return
		}
		
		// Format and add response to chat
		formattedResponse := m.formatGPTResponse(response)
		m.appendToChat("Assistant: " + formattedResponse)
		
		// Clear input
		m.gptInput.SetText("")
	}

	// GPT analysis button
	analyzeBtn := widget.NewButton("Analyze with GPT", m.analyzePage)

	// Create chat interface with full height
	chatContainer := container.NewBorder(
		analyzeBtn,        // Top
		m.gptInput,       // Bottom
		nil,              // Left
		nil,              // Right
		container.NewScroll(m.gptView), // Center (takes remaining space)
	)

	// Create the main layout
	leftPanel := container.NewBorder(
		topControls,
		nil,
		nil,
		nil,
		m.scrollPDF,
	)

	// Main content split with adjusted ratio
	mainContent := container.NewHSplit(
		leftPanel,
		chatContainer,
	)
	mainContent.SetOffset(0.7)

	// Set minimum window size and content
	m.window.Resize(fyne.NewSize(1200, 800))
	m.window.SetContent(mainContent)

	// Add keyboard shortcuts for zoom
	m.window.Canvas().SetOnTypedKey(func(ke *fyne.KeyEvent) {
		switch ke.Name {
		case fyne.KeyPlus, fyne.KeyEqual:
			m.adjustZoom(0.1)
		case fyne.KeyMinus:
			m.adjustZoom(-0.1)
		case fyne.Key0:
			m.zoomLevel = 1.0
			m.updateZoomLabel()
			m.updatePageDisplay()
		}
	})
}

func (m *MainWindow) updateZoomLabel() {
	m.zoomLabel.SetText(fmt.Sprintf("Zoom: %.0f%%", m.zoomLevel*100))
}

func (m *MainWindow) updatePageDisplay() {
	if m.pdfHandler == nil {
		return
	}

	totalPages := m.pdfHandler.NumPages()
	if totalPages == 0 {
		return
	}

	// Update page display with zoom level
	pageInfo, err := m.pdfHandler.GetPage(m.pageNum, m.zoomLevel)
	if err != nil {
		dialog.ShowError(err, m.window)
		return
	}

	// Update image
	if pageInfo.Image != nil {
		m.pdfDisplay.Image = pageInfo.Image
		m.pdfDisplay.Resize(fyne.NewSize(
			float32(pageInfo.Width),
			float32(pageInfo.Height),
		))
		m.pdfDisplay.Refresh()
	}

	// Update text
	if pageInfo.Text != "" {
		m.pageText.SetText(pageInfo.Text)
	} else {
		m.pageText.SetText("")
	}
	
	// Update status
	m.pageLabel.SetText(fmt.Sprintf("Page %d of %d", m.pageNum, totalPages))
	m.statusBar.SetText(fmt.Sprintf("PDF loaded - %d pages", totalPages))

	// Scroll to top of page
	m.scrollPDF.Offset = fyne.NewPos(0, 0)
	m.scrollPDF.Refresh()
}

func (m *MainWindow) appendToChat(message string) {
	m.chatHistory = append(m.chatHistory, message)
	
	// Update chat view
	var chatText strings.Builder
	for _, msg := range m.chatHistory {
		chatText.WriteString(msg)
		chatText.WriteString("\n\n")
	}
	
	m.gptView.SetText(chatText.String())
	
	// Scroll to bottom
	m.gptView.CursorRow = len(m.gptView.Text)
}

func (m *MainWindow) formatGPTResponse(response string) string {
	// Split response into paragraphs
	paragraphs := strings.Split(response, "\n")
	
	// Format each paragraph
	var formatted strings.Builder
	for _, p := range paragraphs {
		// Trim whitespace
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		
		// Add paragraph to formatted text
		formatted.WriteString(p)
		formatted.WriteString("\n\n")
	}
	
	return strings.TrimSpace(formatted.String())
}
