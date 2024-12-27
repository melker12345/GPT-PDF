package gpt

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"strings"
)

type Handler struct {
	apiKey string
}

func NewHandler() *Handler {
	return &Handler{
		apiKey: os.Getenv("OPENAI_API_KEY"),
	}
}

func (h *Handler) imageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return "", fmt.Errorf("failed to encode image: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

type GPTRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type Content struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type GPTResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (h *Handler) AnalyzePage(pageDesc string, img image.Image) (string, error) {
	base64Img, err := h.imageToBase64(img)
	if err != nil {
		return "", fmt.Errorf("failed to convert image: %w", err)
	}

	reqBody := struct {
		Model     string `json:"model"`
		Messages  []any  `json:"messages"`
		MaxTokens int    `json:"max_tokens"`
	}{
		Model: "gpt-4o-mini",
		Messages: []any{
			map[string]any{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": fmt.Sprintf(`Analyze this PDF page. Here's the context: %s

Please provide a detailed analysis including:
1. A description of what you see on the page
2. Any key information or important points
3. The overall layout and structure
4. Any notable text, diagrams, or figures`, pageDesc),
					},
					{
						"type": "image_url",
						"image_url": map[string]any{
							"url":    fmt.Sprintf("data:image/jpeg;base64,%s", base64Img),
							"detail": "high",
						},
					},
				},
			},
		},
		MaxTokens: 300,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", string(body))
	}

	var gptResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &gptResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(gptResp.Choices) == 0 {
		return "", fmt.Errorf("no response from GPT")
	}

	return gptResp.Choices[0].Message.Content, nil
}

func (h *Handler) AskQuestion(question string, history []string) (string, error) {
	// For follow-up questions, we'll use a simpler text-only request
	messages := []map[string]any{
		{
			"role":    "system",
			"content": "You are a helpful assistant analyzing PDF content. Provide clear, well-structured responses.",
		},
	}

	// Add chat history
	for _, msg := range history {
		role := "user"
		content := msg

		if strings.HasPrefix(msg, "Assistant: ") {
			role = "assistant"
			content = strings.TrimPrefix(msg, "Assistant: ")
		} else if strings.HasPrefix(msg, "You: ") {
			content = strings.TrimPrefix(msg, "You: ")
		}

		messages = append(messages, map[string]any{
			"role":    role,
			"content": content,
		})
	}

	// Add current question
	messages = append(messages, map[string]any{
		"role":    "user",
		"content": question,
	})

	reqBody := struct {
		Model     string         `json:"model"`
		Messages  []map[string]any `json:"messages"`
		MaxTokens int           `json:"max_tokens"`
	}{
		Model:     "gpt-4o-mini",
		Messages:  messages,
		MaxTokens: 300,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", string(body))
	}

	var gptResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &gptResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(gptResp.Choices) == 0 {
		return "", fmt.Errorf("no response from GPT")
	}

	return gptResp.Choices[0].Message.Content, nil
}
