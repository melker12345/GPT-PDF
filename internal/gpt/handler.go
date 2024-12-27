package gpt

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

// Handler manages GPT operations
type Handler struct {
	client *openai.Client
}

// NewHandler creates a new GPT handler
func NewHandler() *Handler {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil
	}

	client := openai.NewClient(apiKey)
	return &Handler{client: client}
}

// Ask sends a question to GPT and returns the response
func (h *Handler) Ask(question string) (string, error) {
	resp, err := h.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: question,
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("error getting GPT response: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
}

// AnalyzePage analyzes the text content of a PDF page
func (h *Handler) AnalyzePage(text string, _ interface{}) (string, error) {
	prompt := fmt.Sprintf("Please analyze the following text from a PDF page:\n\n%s\n\nProvide a concise summary and highlight any key points.", text)

	resp, err := h.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a helpful assistant that analyzes PDF content. Focus on extracting key information and providing clear, concise summaries.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("error analyzing page: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
}

// AnalyzePageWithImage sends both the image and question to GPT-4 Vision
func (h *Handler) AnalyzePageWithImage(imgData []byte, question *string) (string, error) {
	// Convert image to base64
	b64Img := base64.StdEncoding.EncodeToString(imgData)

	// Create the request
	req := openai.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: *question,
					},
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL:    fmt.Sprintf("data:image/png;base64,%s", b64Img),
							Detail: openai.ImageURLDetailHigh,
						},
					},
				},
			},
		},
		MaxTokens: 1000,
	}

	// Send the request
	resp, err := h.client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("GPT API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from GPT")
	}

	return resp.Choices[0].Message.Content, nil
}
