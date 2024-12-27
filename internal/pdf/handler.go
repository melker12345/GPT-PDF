package pdf

import (
	"fmt"
	"image"
	"sync"

	"github.com/gen2brain/go-fitz"
	"github.com/nfnt/resize"
)

type PageInfo struct {
	Image       image.Image
	OriginalImg image.Image
	Width       int
	Height      int
}

type Handler struct {
	doc   *fitz.Document
	cache sync.Map
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) LoadPDF(path string) error {
	if h.doc != nil {
		h.doc.Close()
	}
	h.cache.Range(func(key, value interface{}) bool {
		h.cache.Delete(key)
		return true
	})

	doc, err := fitz.New(path)
	if err != nil {
		return fmt.Errorf("failed to load PDF: %w", err)
	}

	h.doc = doc
	return nil
}

func (h *Handler) GetPage(pageNum int, zoom float64) (PageInfo, error) {
	if h.doc == nil {
		return PageInfo{}, fmt.Errorf("no PDF loaded")
	}

	if pageNum < 1 || pageNum > h.doc.NumPage() {
		return PageInfo{}, fmt.Errorf("invalid page number")
	}

	// Create cache key that includes zoom level
	cacheKey := fmt.Sprintf("%d-%.2f", pageNum, zoom)

	// Check cache first
	if cached, ok := h.cache.Load(cacheKey); ok {
		if pageInfo, ok := cached.(PageInfo); ok {
			return pageInfo, nil
		}
	}

	// Get original page image
	origImg, err := h.doc.Image(pageNum-1)
	if err != nil {
		return PageInfo{}, fmt.Errorf("failed to get page image: %w", err)
	}

	bounds := origImg.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	// Create zoomed image if needed
	var displayImg image.Image
	if zoom != 1.0 {
		newWidth := uint(float64(origWidth) * zoom)
		newHeight := uint(float64(origHeight) * zoom)
		displayImg = resize.Resize(newWidth, newHeight, origImg, resize.Lanczos3)
	} else {
		displayImg = origImg
	}

	pageInfo := PageInfo{
		Image:       displayImg,
		OriginalImg: origImg,
		Width:       origWidth,
		Height:      origHeight,
	}

	// Cache the result
	h.cache.Store(cacheKey, pageInfo)

	return pageInfo, nil
}

func (h *Handler) Close() error {
	if h.doc != nil {
		return h.doc.Close()
	}
	return nil
}

func (h *Handler) GetPageCount() int {
	if h.doc == nil {
		return 0
	}
	return h.doc.NumPage()
}
