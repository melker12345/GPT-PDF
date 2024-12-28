package pdf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var gsPath string

func init() {
	if runtime.GOOS == "windows" {
		gsPath = "C:\\Program Files\\gs\\gs10.04.0\\bin\\gswin64c.exe"
	} else {
		gsPath = "/usr/bin/gs"
	}
}

// Handler manages PDF operations
type Handler struct {
	pdfPath   string
	pdfData   []byte
	numPages  int
	mutex     sync.Mutex
}

// NewHandler creates a new PDF handler
func NewHandler(pdfPath string) (*Handler, error) {
	// Check if Ghostscript exists
	if _, err := os.Stat(gsPath); err != nil {
		return nil, fmt.Errorf("Ghostscript not found at %s. Please install it from https://ghostscript.com/releases/gsdnld.html", gsPath)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("error getting absolute path: %v", err)
	}

	// Read the PDF file into memory
	pdfData, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("error reading PDF file: %v", err)
	}

	// Create a temporary file for the PDF
	tempFile, err := os.CreateTemp("", "gs-input-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("error creating temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write the PDF data to the temp file
	if _, err := tempFile.Write(pdfData); err != nil {
		return nil, fmt.Errorf("error writing temp file: %v", err)
	}

	// Get number of pages using Ghostscript
	cmd := exec.Command(gsPath, 
		"-dSAFER",       // Safer mode
		"-dNODISPLAY",   // No display
		"-dBATCH",       // Batch mode
		"-dNOPAUSE",     // No pause
		"-q",            // Quiet mode
		"-c",            // Begin executing PostScript
		fmt.Sprintf("(%s) (r) file runpdfbegin pdfpagecount = flush quit", strings.ReplaceAll(tempFile.Name(), "\\", "/")),
	)

	// Set working directory to system temp
	cmd.Dir = os.TempDir()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error getting page count: %v (%s)", err, output)
	}

	var numPages int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &numPages); err != nil {
		return nil, fmt.Errorf("error parsing page count: %v", err)
	}

	return &Handler{
		pdfPath:  absPath,
		pdfData:  pdfData,
		numPages: numPages,
	}, nil
}

// GetNumPages returns the total number of pages in the PDF
func (h *Handler) GetNumPages() int {
	return h.numPages
}

// Close releases resources
func (h *Handler) Close() {
	h.pdfData = nil
}

// RenderPage renders a specific page as a PNG image
func (h *Handler) RenderPage(pageNum int) ([]byte, error) {
	if pageNum < 1 || pageNum > h.numPages {
		return nil, fmt.Errorf("invalid page number: %d", pageNum)
	}

	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Create a temporary directory for the image
	tempDir, err := os.MkdirTemp("", "pdf-images-*")
	if err != nil {
		return nil, fmt.Errorf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a temporary file for the PDF
	tempFile, err := os.CreateTemp(tempDir, "gs-input-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("error creating temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write the PDF data to the temp file
	if _, err := tempFile.Write(h.pdfData); err != nil {
		return nil, fmt.Errorf("error writing temp file: %v", err)
	}
	tempFile.Close()

	// Set up the output PNG file path
	outFile := filepath.Join(tempDir, fmt.Sprintf("page-%d.png", pageNum))

	// Use Ghostscript to convert the PDF page to PNG with high resolution (300 DPI)
	cmd := exec.Command(gsPath,
		"-dSAFER",           // Safer mode
		"-dBATCH",           // Exit after processing
		"-dNOPAUSE",         // Don't pause between pages
		"-dQUIET",           // Suppress output
		"-dFirstPage="+fmt.Sprint(pageNum),  // Start page
		"-dLastPage="+fmt.Sprint(pageNum),   // End page
		"-sDEVICE=png16m",   // PNG output device
		"-r300",             // Resolution 300 DPI
		"-dTextAlphaBits=4", // Text antialiasing
		"-dGraphicsAlphaBits=4", // Graphics antialiasing
		"-o", outFile,       // Output file
		tempFile.Name(),     // Input PDF
	)

	// Set working directory to system temp
	cmd.Dir = tempDir

	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("error rendering page: %v (%s)", err, output)
	}

	// Read the generated image
	imgData, err := os.ReadFile(outFile)
	if err != nil {
		return nil, fmt.Errorf("error reading image file: %v", err)
	}

	return imgData, nil
}
