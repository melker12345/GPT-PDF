package gpt

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type Handler struct {
	client *openai.Client
}

func NewHandler() *Handler {
	return &Handler{
		client: openai.NewClient(os.Getenv("OPENAI_API_KEY")),
	}
}

func (h *Handler) AnalyzePage(text string) (string, error) {
	prompt := fmt.Sprintf(`Analyze the following text/images from a PDF page and provide a clear, well-structured explanation of the content: "%s"`, text)

	return h.getGPTResponse(prompt, nil)
}

func (h *Handler) AskQuestion(question string, history []string) (string, error) {
	// Build conversation context from history
	var messages []openai.ChatCompletionMessage
	
	// Add system message
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: "You are a helpful assistant analyzing PDF content. Provide clear, well-structured responses.",
	})
	
	// Add chat history
	for _, msg := range history {
		role := openai.ChatMessageRoleUser
		content := msg
		
		if strings.HasPrefix(msg, "Assistant: ") {
			role = openai.ChatMessageRoleAssistant
			content = strings.TrimPrefix(msg, "Assistant: ")
		} else if strings.HasPrefix(msg, "You: ") {
			content = strings.TrimPrefix(msg, "You: ")
		}
		
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: content,
		})
	}
	
	// Add current question
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: question,
	})

	// Create chat completion request
	resp, err := h.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       openai.GPT3Dot5Turbo,
			Messages:    messages,
			MaxTokens:   500,
			Temperature: 0.7,
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to get GPT response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from GPT")
	}

	return resp.Choices[0].Message.Content, nil
}

func (h *Handler) getGPTResponse(prompt string, history []string) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "You are a helpful assistant analyzing PDF content. Provide clear, well-structured responses.",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	resp, err := h.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       openai.GPT3Dot5Turbo,
			Messages:    messages,
			MaxTokens:   500,
			Temperature: 0.7,
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to get GPT response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from GPT")
	}

	return resp.Choices[0].Message.Content, nil
}
