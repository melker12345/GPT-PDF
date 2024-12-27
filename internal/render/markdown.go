package render

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type MarkdownRenderer struct{}

func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{}
}

func (r *MarkdownRenderer) RenderToRichText(content string) *widget.RichText {
	richText := widget.NewRichText()
	richText.Wrapping = fyne.TextWrapWord

	// Split content into lines
	lines := strings.Split(content, "\n")
	var segments []widget.RichTextSegment

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for headings
		if strings.HasPrefix(line, "#") {
			level := 0
			for i := 0; i < len(line) && line[i] == '#'; i++ {
				level++
			}
			text := strings.TrimSpace(line[level:])
			segments = append(segments, &widget.TextSegment{
				Text: text + "\n",
				Style: widget.RichTextStyle{
					ColorName: theme.ColorNamePrimary,
				},
			})
			continue
		}

		// Check for bullet points
		if strings.HasPrefix(strings.TrimSpace(line), "* ") || strings.HasPrefix(strings.TrimSpace(line), "- ") {
			text := strings.TrimSpace(line[2:])
			segments = append(segments, &widget.TextSegment{
				Text: "• " + text + "\n",
			})
			continue
		}

		// Regular text
		segments = append(segments, &widget.TextSegment{
			Text: line + "\n",
		})
	}

	richText.Segments = segments
	return richText
}

func (r *MarkdownRenderer) CreateMessageContainer(content string, isUser bool) *fyne.Container {
	richText := r.RenderToRichText(content)

	// Style based on sender
	if isUser {
		for _, seg := range richText.Segments {
			if textSeg, ok := seg.(*widget.TextSegment); ok {
				textSeg.Style.ColorName = theme.ColorNamePrimary
			}
		}
	} else {
		for _, seg := range richText.Segments {
			if textSeg, ok := seg.(*widget.TextSegment); ok {
				textSeg.Style.ColorName = theme.ColorNameForeground
			}
		}
	}

	// Create a scrollable container
	scroll := container.NewScroll(richText)
	scroll.SetMinSize(fyne.NewSize(400, 100))

	return container.NewMax(scroll)
}
