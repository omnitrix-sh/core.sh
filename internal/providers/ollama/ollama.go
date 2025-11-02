package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/omnitrix-sh/core.sh/pkg/models"
)

type Provider struct {
	baseURL string
	client  *http.Client
	model   string
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Tools    []ollamaTool    `json:"tools,omitempty"`
}

type ollamaTool struct {
	Type     string                 `json:"type"`
	Function ollamaToolFunction     `json:"function"`
}

type ollamaToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ollamaChatResponse struct {
	Model              string        `json:"model"`
	CreatedAt          string        `json:"created_at"`
	Message            ollamaMessage `json:"message"`
	Done               bool          `json:"done"`
	TotalDuration      int64         `json:"total_duration,omitempty"`
	LoadDuration       int64         `json:"load_duration,omitempty"`
	PromptEvalCount    int           `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64         `json:"prompt_eval_duration,omitempty"`
	EvalCount          int           `json:"eval_count,omitempty"`
	EvalDuration       int64         `json:"eval_duration,omitempty"`
}

func NewProvider(baseURL, model string) *Provider {
	return &Provider{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  &http.Client{},
		model:   model,
	}
}

func (p *Provider) Chat(ctx context.Context, req models.ChatRequest) (*models.ChatResponse, error) {
	// Convert to Ollama format
	ollamaReq := p.convertRequest(req)
	ollamaReq.Stream = false

	// Marshal request
	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		p.baseURL+"/api/chat",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var ollamaResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &models.ChatResponse{
		ID:           ollamaResp.CreatedAt,
		Model:        ollamaResp.Model,
		Content:      ollamaResp.Message.Content,
		FinishReason: "stop",
		Usage: models.TokenUsage{
			PromptTokens:     ollamaResp.PromptEvalCount,
			CompletionTokens: ollamaResp.EvalCount,
			TotalTokens:      ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
		},
	}, nil
}

func (p *Provider) Stream(ctx context.Context, req models.ChatRequest) (<-chan models.StreamChunk, error) {
	// Convert to Ollama format
	ollamaReq := p.convertRequest(req)
	ollamaReq.Stream = true

	// Marshal request
	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		p.baseURL+"/api/chat",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Create channel for streaming
	chunks := make(chan models.StreamChunk)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var ollamaResp ollamaChatResponse
			if err := json.Unmarshal(line, &ollamaResp); err != nil {
				// Send error and continue
				chunks <- models.StreamChunk{
					Delta: fmt.Sprintf("[Error parsing response: %v]", err),
					Done:  true,
				}
				return
			}

			chunk := models.StreamChunk{
				ID:           ollamaResp.CreatedAt,
				Delta:        ollamaResp.Message.Content,
				Done:         ollamaResp.Done,
				FinishReason: "",
			}

			if ollamaResp.Done {
				chunk.FinishReason = "stop"
			}

			chunks <- chunk

			if ollamaResp.Done {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			chunks <- models.StreamChunk{
				Delta: fmt.Sprintf("[Stream error: %v]", err),
				Done:  true,
			}
		}
	}()

	return chunks, nil
}

func (p *Provider) convertRequest(req models.ChatRequest) ollamaChatRequest {
	messages := make([]ollamaMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = ollamaMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	ollamaReq := ollamaChatRequest{
		Model:    req.Model,
		Messages: messages,
	}

	// Convert tools if present
	if len(req.Tools) > 0 {
		ollamaReq.Tools = make([]ollamaTool, len(req.Tools))
		for i, tool := range req.Tools {
			ollamaReq.Tools[i] = ollamaTool{
				Type: tool.Type,
				Function: ollamaToolFunction{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			}
		}
	}

	return ollamaReq
}

func (p *Provider) Model() string {
	return p.model
}
