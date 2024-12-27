package ui

import (
	"bytes"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/memleak-io/gpt-pdf/internal/gpt"
	"github.com/memleak-io/gpt-pdf/internal/pdf"
)

type MainWindow struct {
	window         fyne.Window
	pdfHandler     *pdf.Handler
	gptHandler     *gpt.Handler
	currentPage    int
	pdfViewer      *canvas.Image
	pdfContainer   *fyne.Container
	imageContainer *fyne.Container
	chatView       *widget.TextGrid
	pageLabel      *widget.Label
	zoomLevel      float64
	chatInput      *widget.Entry
}

func NewMainWindow(window fyne.Window) *MainWindow {
	w := &MainWindow{
		window:      window,
		currentPage: 1,
		chatView:    widget.NewTextGrid(),
		zoomLevel:   1.0,
	}

	// Create GPT handler
	gptHandler := gpt.NewHandler()
	w.gptHandler = gptHandler

	// Create the PDF viewer
	w.pdfViewer = canvas.NewImageFromResource(nil)
	w.pdfViewer.FillMode = canvas.ImageFillContain
	w.pdfViewer.SetMinSize(fyne.NewSize(600, 800))
	w.imageContainer = container.NewCenter(w.pdfViewer)

	// Create navigation controls
	prevButton := widget.NewButton("Previous", w.previousPage)
	w.pageLabel = widget.NewLabel("Page: 1")
	nextButton := widget.NewButton("Next", w.nextPage)
	zoomInButton := widget.NewButton("Zoom In", w.zoomIn)
	zoomOutButton := widget.NewButton("Zoom Out", w.zoomOut)
	askGPTButton := widget.NewButton("Ask GPT", w.askGPT)

	// Create navigation container
	navContainer := container.NewHBox(
		prevButton,
		w.pageLabel,
		nextButton,
		widget.NewSeparator(),
		zoomInButton,
		zoomOutButton,
		widget.NewSeparator(),
		askGPTButton,
	)

	// Create PDF container with navigation
	w.pdfContainer = container.NewBorder(
		navContainer, // top
		nil,         // bottom
		nil,         // left
		nil,         // right
		w.imageContainer,
	)

	// Create chat input
	w.chatInput = widget.NewEntry()
	w.chatInput.SetPlaceHolder("Ask a question about the page...")
	w.chatInput.OnSubmitted = w.handleChatInput

	// Create chat container with scroll
	chatScroll := container.NewScroll(w.chatView)
	chatScroll.SetMinSize(fyne.NewSize(300, 600))
	chatContainer := container.NewBorder(nil, w.chatInput, nil, nil, chatScroll)

	// Create main container with split
	split := container.NewHSplit(w.pdfContainer, chatContainer)
	split.SetOffset(0.7)

	// Create toolbar
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.DocumentIcon(), w.openPDF),
	)

	// Set window content
	window.SetContent(container.NewBorder(toolbar, nil, nil, nil, split))
	window.Resize(fyne.NewSize(1200, 800))

	return w
}

func (w *MainWindow) zoomIn() {
	w.zoomLevel *= 1.2
	w.updatePageDisplay()
}

func (w *MainWindow) zoomOut() {
	w.zoomLevel *= 0.8
	w.updatePageDisplay()
}

func (w *MainWindow) askGPT() {
	if w.pdfHandler == nil {
		w.showError(fmt.Errorf("please open a PDF file first"))
		return
	}

	if w.gptHandler == nil {
		w.showError(fmt.Errorf("GPT handler not initialized"))
		return
	}

	// Get the current page as an image
	imgData, err := w.pdfHandler.RenderPage(w.currentPage)
	if err != nil {
		w.showError(fmt.Errorf("error capturing page: %v", err))
		return
	}

	question := w.chatInput.Text
	if question == "" {
		question = "What do you see in this image? Please describe it in detail."
	}

	w.appendToChatView(fmt.Sprintf("You: %s\n", question))

	// Send both the image and the question to GPT
	response, err := w.gptHandler.AnalyzePageWithImage(imgData, &question)
	if err != nil {
		w.showError(fmt.Errorf("error from GPT: %v", err))
		return
	}
	w.appendToChatView(fmt.Sprintf("GPT: %s\n", response))
}

func (w *MainWindow) handleChatInput(input string) {
	if input == "" {
		return
	}
	w.chatInput.SetText("")
	w.askGPT()
}

func (w *MainWindow) appendToChatView(text string) {
	currentText := w.chatView.Text()
	w.chatView.SetText(currentText + text)
	w.chatView.Refresh()
}

func (w *MainWindow) openPDF() {
	// If there's already a PDF open, close it
	if w.pdfHandler != nil {
		w.pdfHandler.Close()
		w.pdfHandler = nil
	}

	// Show file dialog
	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			w.showError(err)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()

		// Get the file path
		filePath := reader.URI().Path()
		if filePath[0] == '/' && len(filePath) > 2 && filePath[2] == ':' {
			// Remove leading slash from Windows paths
			filePath = filePath[1:]
		}

		// Create new PDF handler
		handler, err := pdf.NewHandler(filePath)
		if err != nil {
			w.showError(fmt.Errorf("error opening PDF: %v", err))
			return
		}

		w.pdfHandler = handler
		w.currentPage = 1
		w.zoomLevel = 1.0
		w.updatePageDisplay()
	}, w.window)

	// Only allow PDF files
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".pdf"}))
	fd.Show()
}

func (w *MainWindow) previousPage() {
	if w.pdfHandler == nil {
		return
	}

	if w.currentPage > 1 {
		w.currentPage--
		w.updatePageDisplay()
	}
}

func (w *MainWindow) nextPage() {
	if w.pdfHandler == nil {
		return
	}

	if w.currentPage < w.pdfHandler.GetNumPages() {
		w.currentPage++
		w.updatePageDisplay()
	}
}

func (w *MainWindow) updatePageDisplay() {
	if w.pdfHandler == nil {
		return
	}

	// Update page label
	w.pageLabel.SetText(fmt.Sprintf("Page: %d / %d", w.currentPage, w.pdfHandler.GetNumPages()))

	// Render the current page
	imgData, err := w.pdfHandler.RenderPage(w.currentPage)
	if err != nil {
		w.showError(err)
		return
	}

	// Create new image from the rendered page
	newImg := canvas.NewImageFromReader(bytes.NewReader(imgData), "page.png")
	newImg.FillMode = canvas.ImageFillContain
	
	// Apply zoom
	baseSize := fyne.NewSize(600, 800)
	zoomedSize := fyne.NewSize(
		float32(float64(baseSize.Width) * w.zoomLevel),
		float32(float64(baseSize.Height) * w.zoomLevel),
	)
	newImg.SetMinSize(zoomedSize)

	// Replace the old image
	w.pdfViewer = newImg
	w.imageContainer.Objects = []fyne.CanvasObject{newImg}
	w.imageContainer.Refresh()
}

func (w *MainWindow) showError(err error) {
	// Create a text entry with the error message
	errorText := widget.NewMultiLineEntry()
	errorText.SetText(err.Error())
	errorText.TextStyle = fyne.TextStyle{Monospace: true}
	errorText.Wrapping = fyne.TextWrapWord
	
	// Make it read-only but selectable
	errorText.Disable()

	// Create a custom dialog
	d := dialog.NewCustom("Error", "OK", errorText, w.window)
	d.Resize(fyne.NewSize(400, 200))
	d.Show()
}
