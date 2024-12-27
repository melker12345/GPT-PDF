package ui

import (
	"fmt"
	"strings"

	"pdf-reader/internal/gpt"
	"pdf-reader/internal/pdf"
	"pdf-reader/internal/render"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type MainWindow struct {
	window      fyne.Window
	pdfDisplay  *canvas.Image
	gptView     *container.Scroll
	gptInput    *widget.Entry
	chatHistory []string
	pageNum     int
	zoomLevel   float64
	pdfHandler  *pdf.Handler
	gptHandler  *gpt.Handler
	mdRenderer  *render.MarkdownRenderer
	statusBar   *widget.Label
	pageLabel   *widget.Label
	zoomLabel   *widget.Label
	scrollPDF   *container.Scroll
}

func NewMainWindow(window fyne.Window) *MainWindow {
	m := &MainWindow{
		window:      window,
		pdfDisplay:  canvas.NewImageFromImage(nil),
		gptInput:    widget.NewEntry(),
		chatHistory: make([]string, 0),
		pageNum:     1,
		zoomLevel:   1.0,
		pdfHandler:  pdf.NewHandler(),
		gptHandler:  gpt.NewHandler(),
		mdRenderer:  render.NewMarkdownRenderer(),
		statusBar:   widget.NewLabel("No PDF loaded"),
		pageLabel:   widget.NewLabel("Page: -"),
		zoomLabel:   widget.NewLabel("Zoom: 100%"),
	}

	// Initialize the chat view with an empty container
	vbox := container.NewVBox()
	m.gptView = container.NewScroll(vbox)

	return m
}

func (m *MainWindow) prevPage() {
	if m.pageNum > 1 {
		m.pageNum--
		m.updatePageDisplay()
	}
}

func (m *MainWindow) nextPage() {
	if m.pageNum < m.pdfHandler.GetPageCount() {
		m.pageNum++
		m.updatePageDisplay()
	}
}

func (m *MainWindow) analyzePage() {
	if m.pdfHandler == nil {
		dialog.ShowError(fmt.Errorf("no PDF loaded"), m.window)
		return
	}

	// Get the current page with original size for better analysis
	pageInfo, err := m.pdfHandler.GetPage(m.pageNum, 1.0)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to get page: %w", err), m.window)
		return
	}

	// Create a description of the page
	pageDesc := fmt.Sprintf("This is page %d of the PDF document. The page dimensions are %dx%d pixels.",
		m.pageNum, pageInfo.Width, pageInfo.Height)

	// Send the page description and original image to GPT for analysis
	response, err := m.gptHandler.AnalyzePage(pageDesc, pageInfo.OriginalImg)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to analyze page: %w", err), m.window)
		return
	}

	m.appendToChat(fmt.Sprintf("Analysis of page %d:", m.pageNum))
	m.appendToChat("Assistant: " + response)
}

func (m *MainWindow) adjustZoom(delta float64) {
	newZoom := m.zoomLevel + delta
	if newZoom >= 0.1 && newZoom <= 5.0 {
		m.zoomLevel = newZoom
		m.updateZoomLabel()
		m.updatePageDisplay()
	}
}

func (m *MainWindow) updateZoomLabel() {
	m.zoomLabel.SetText(fmt.Sprintf("Zoom: %.0f%%", m.zoomLevel*100))
}

func (m *MainWindow) Setup() {
	// Set window title and size
	m.window.SetTitle("PDF Reader with GPT")
	m.window.Resize(fyne.NewSize(1200, 800))

	// Set up file open button
	openBtn := widget.NewButton("Open PDF", func() {
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
			m.updatePage()
		}, m.window)
	})

	// Navigation controls
	prevBtn := widget.NewButton("Previous", m.prevPage)
	nextBtn := widget.NewButton("Next", m.nextPage)

	// Add zoom controls with finer adjustments
	zoomOutBtn := widget.NewButton("-", func() { m.adjustZoom(-0.1) })
	zoomInBtn := widget.NewButton("+", func() { m.adjustZoom(0.1) })

	// Add preset zoom levels
	zoomPresets := widget.NewSelect([]string{"10%", "25%", "50%", "75%", "100%", "125%", "150%", "200%"}, func(value string) {
		var zoom float64
		switch value {
		case "10%":
			zoom = 0.1
		case "25%":
			zoom = 0.25
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
		m.zoomLevel = zoom
		m.updatePage()
		m.zoomLabel.SetText(fmt.Sprintf("Zoom: %d%%", int(zoom*100)))
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
	)

	// Create toolbar
	toolbar := container.NewHBox(
		openBtn,
		widget.NewSeparator(),
		navControls,
	)

	// PDF display area
	m.pdfDisplay.FillMode = canvas.ImageFillContain
	m.pdfDisplay.SetMinSize(fyne.NewSize(600, 800))

	// Create a container to center the PDF display
	pdfContainer := container.NewCenter(m.pdfDisplay)

	// Create a scroll container for the PDF content
	m.scrollPDF = container.NewScroll(pdfContainer)

	// Set up GPT input and chat
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
		m.appendToChat("Assistant: " + response)

		// Clear input
		m.gptInput.SetText("")
	}

	analyzeBtn := widget.NewButton("Analyze Current Page", m.analyzePage)

	// Create chat interface with full height
	chatContainer := container.NewBorder(
		analyzeBtn, // Top
		m.gptInput, // Bottom
		nil,        // Left
		nil,        // Right
		m.gptView,  // Center (takes remaining space)
	)

	// Create the main layout with a split
	split := container.NewHSplit(
		m.scrollPDF,   // Left side - PDF viewer
		chatContainer, // Right side - Chat interface
	)
	split.SetOffset(0.6) // Set the split to 60% PDF, 40% chat

	// Create the main layout
	mainContent := container.NewBorder(
		toolbar,     // Top
		m.statusBar, // Bottom
		nil,         // Left
		nil,         // Right
		split,       // Center
	)

	m.window.SetContent(mainContent)
}

func (m *MainWindow) updatePage() {
	if m.pdfHandler == nil {
		return
	}

	// Update page info
	pageInfo, err := m.pdfHandler.GetPage(m.pageNum, m.zoomLevel)
	if err != nil {
		dialog.ShowError(err, m.window)
		return
	}

	// Update page display
	m.updatePageDisplay()

	// Update status
	m.pageLabel.SetText(fmt.Sprintf("Page: %d/%d", m.pageNum, m.pdfHandler.GetPageCount()))
	m.statusBar.SetText(fmt.Sprintf("Page %d - %dx%d pixels", m.pageNum, pageInfo.Width, pageInfo.Height))
}

func (m *MainWindow) updatePageDisplay() {
	if m.pdfHandler == nil {
		return
	}

	totalPages := m.pdfHandler.GetPageCount()
	if totalPages == 0 {
		return
	}

	// Ensure page number is within bounds
	if m.pageNum < 1 {
		m.pageNum = 1
	} else if m.pageNum > totalPages {
		m.pageNum = totalPages
	}

	// Get page image
	pageInfo, err := m.pdfHandler.GetPage(m.pageNum, m.zoomLevel)
	if err != nil {
		dialog.ShowError(err, m.window)
		return
	}

	// Update image display
	m.pdfDisplay.Image = pageInfo.Image
	m.pdfDisplay.Refresh()

	// Update status
	m.pageLabel.SetText(fmt.Sprintf("Page: %d/%d", m.pageNum, totalPages))
	m.statusBar.SetText(fmt.Sprintf("Page %d - %dx%d pixels", m.pageNum, pageInfo.Width, pageInfo.Height))
}

func (m *MainWindow) appendToChat(message string) {
	m.chatHistory = append(m.chatHistory, message)

	// Update chat view with rendered markdown
	vbox := container.NewVBox()

	for _, msg := range m.chatHistory {
		if strings.HasPrefix(msg, "Assistant: ") {
			// Render assistant messages as markdown
			content := strings.TrimPrefix(msg, "Assistant: ")
			rendered := m.mdRenderer.RenderMarkdown(content)
			vbox.Add(rendered)
		} else {
			// Regular text for user messages
			label := widget.NewLabel(msg)
			label.Wrapping = fyne.TextWrapWord
			vbox.Add(label)
		}

		// Add spacing between messages
		vbox.Add(widget.NewSeparator())
	}

	// Update the scroll container content
	m.gptView.Content = vbox
	m.gptView.Refresh()
	// Scroll to bottom
	m.gptView.ScrollToBottom()
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
