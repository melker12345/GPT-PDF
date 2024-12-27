package render

import (
	"bytes"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type MarkdownRenderer struct {
	md     goldmark.Markdown
	source []byte
}

func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{
		md: goldmark.New(
			goldmark.WithExtensions(),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
			),
		),
	}
}

func (r *MarkdownRenderer) RenderMarkdown(content string) fyne.CanvasObject {
	// Store source for later use
	r.source = []byte(content)

	// Parse the markdown content
	doc := r.md.Parser().Parse(text.NewReader(r.source))

	// Create a vertical box to hold all rendered elements
	vbox := container.NewVBox()

	// Walk through the AST and render each node
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Heading:
			text := getTextFromNode(node, r.source)
			size := 24 - (node.Level * 2) // Decrease size for deeper headings
			heading := canvas.NewText(text, nil)
			heading.TextSize = float32(size)
			heading.TextStyle.Bold = true
			vbox.Add(heading)
			vbox.Add(widget.NewSeparator()) // Add separator after heading

		case *ast.Paragraph:
			text := getTextFromNode(node, r.source)
			// Check for LaTeX blocks
			parts := strings.Split(text, "\\[")
			if len(parts) > 1 {
				for i, part := range parts {
					if i == 0 {
						if strings.TrimSpace(part) != "" {
							para := widget.NewLabel(strings.TrimSpace(part))
							para.Wrapping = fyne.TextWrapWord
							vbox.Add(para)
						}
						continue
					}
					
					eqParts := strings.Split(part, "\\]")
					if len(eqParts) > 1 {
						// Create a styled label for the equation
						equation := strings.TrimSpace(eqParts[0])
						eqLabel := widget.NewRichTextFromMarkdown(fmt.Sprintf("```math\n%s\n```", equation))
						vbox.Add(container.NewCenter(eqLabel))
						
						// Render remaining text if any
						if text := strings.TrimSpace(eqParts[1]); text != "" {
							para := widget.NewLabel(text)
							para.Wrapping = fyne.TextWrapWord
							vbox.Add(para)
						}
					}
				}
			} else {
				para := widget.NewLabel(text)
				para.Wrapping = fyne.TextWrapWord
				vbox.Add(para)
			}

		case *ast.List:
			text := getTextFromNode(node, r.source)
			// Format list items with proper indentation and bullets
			var formattedText strings.Builder
			for _, line := range strings.Split(text, "\n") {
				if strings.TrimSpace(line) != "" {
					formattedText.WriteString("• " + strings.TrimSpace(line) + "\n")
				}
			}
			list := widget.NewLabel(formattedText.String())
			list.Wrapping = fyne.TextWrapWord
			vbox.Add(list)

		case *ast.CodeBlock:
			text := string(node.Text(r.source))
			code := widget.NewTextGrid()
			code.SetText(text)
			vbox.Add(container.NewPadded(code))
		}

		return ast.WalkContinue, nil
	})

	scroll := container.NewScroll(vbox)
	scroll.SetMinSize(fyne.NewSize(400, 300)) // Set minimum size for better readability
	return scroll
}

func getTextFromNode(node ast.Node, source []byte) string {
	var buf bytes.Buffer
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if text := child.Text(source); len(text) > 0 {
			buf.Write(text)
			if child.NextSibling() != nil {
				buf.WriteString(" ")
			}
		}
		if child.Kind() == ast.KindEmphasis {
			buf.WriteString("*")
		}
	}
	return buf.String()
}
