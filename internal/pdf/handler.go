package pdf

import (
	"fmt"
	"image"
	"sync"

	"github.com/gen2brain/go-fitz"
	"github.com/nfnt/resize"
)

type Handler struct {
	doc      *fitz.Document
	mutex    sync.RWMutex
	pageInfo map[string]*PageInfo
}

type PageInfo struct {
	Text    string
	Image   image.Image
	Width   float64
	Height  float64
}

func NewHandler() *Handler {
	return &Handler{
		pageInfo: make(map[string]*PageInfo),
	}
}

func (h *Handler) LoadPDF(filepath string) error {
	h.mutex.Lock()
	if h.doc != nil {
		h.doc.Close()
	}
	h.pageInfo = make(map[string]*PageInfo)
	h.mutex.Unlock()

	doc, err := fitz.New(filepath)
	if err != nil {
		return fmt.Errorf("failed to load PDF: %w", err)
	}

	h.mutex.Lock()
	h.doc = doc
	h.mutex.Unlock()
	return nil
}

func (h *Handler) GetPage(pageNum int, zoom float64) (*PageInfo, error) {
	h.mutex.RLock()
	if h.doc == nil {
		h.mutex.RUnlock()
		return nil, fmt.Errorf("no PDF loaded")
	}

	pageIndex := pageNum - 1
	if pageIndex < 0 || pageIndex >= h.doc.NumPage() {
		h.mutex.RUnlock()
		return nil, fmt.Errorf("page number out of range")
	}

	cacheKey := fmt.Sprintf("%d_%.2f", pageNum, zoom)

	if info, exists := h.pageInfo[cacheKey]; exists {
		h.mutex.RUnlock()
		return info, nil
	}
	h.mutex.RUnlock()

	h.mutex.Lock()
	defer h.mutex.Unlock()

	text, err := h.doc.Text(pageIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text: %w", err)
	}

	img, err := h.doc.Image(pageIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to render page: %w", err)
	}

	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	newWidth := uint(float64(origWidth) * zoom)
	newHeight := uint(float64(origHeight) * zoom)

	scaledImg := resize.Resize(newWidth, newHeight, img, resize.Lanczos3)

	info := &PageInfo{
		Text:   text,
		Image:  scaledImg,
		Width:  float64(newWidth),
		Height: float64(newHeight),
	}

	h.pageInfo[cacheKey] = info
	return info, nil
}

func (h *Handler) NumPages() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if h.doc == nil {
		return 0
	}
	return h.doc.NumPage()
}

func (h *Handler) Close() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.doc != nil {
		h.doc.Close()
		h.doc = nil
	}
	h.pageInfo = make(map[string]*PageInfo)
}
