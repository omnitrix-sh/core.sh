package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/omnitrix-sh/core.sh/pkg/models"
)

type Provider struct {
	apiKey  string
	baseURL string
	client  *http.Client
	model   string
}

type openaiMessage struct {
	Role       string            `json:"role"`
	Content    *string           `json:"content,omitempty"`
	ToolCalls  []openaiToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
}

type openaiToolCall struct {
	ID       string              `json:"id"`
	Type     string              `json:"type"`
	Function openaiToolFunction  `json:"function"`
}

type openaiToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openaiChatRequest struct {
	Model    string          `json:"model"`
	Messages []openaiMessage `json:"messages"`
	Tools    []openaiTool    `json:"tools,omitempty"`
	Stream   bool            `json:"stream"`
}

type openaiTool struct {
	Type     string                 `json:"type"`
	Function map[string]interface{} `json:"function"`
}

type openaiChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openaiMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openaiStreamChunk struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content   string             `json:"content,omitempty"`
			ToolCalls []openaiToolCall   `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func NewProvider(apiKey, model string) *Provider {
	return &Provider{
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1",
		client:  &http.Client{},
		model:   model,
	}
}

func (p *Provider) Chat(ctx context.Context, req models.ChatRequest) (*models.ChatResponse, error) {
	openaiReq := p.convertRequest(req)
	openaiReq.Stream = false

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var openaiResp openaiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := openaiResp.Choices[0]
	
	// Convert tool calls
	var toolCalls []models.ToolCall
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls = make([]models.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)
			
			toolCalls[i] = models.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: models.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: args,
				},
			}
		}
	}

	content := ""
	if choice.Message.Content != nil {
		content = *choice.Message.Content
	}
	
	return &models.ChatResponse{
		ID:           openaiResp.ID,
		Model:        openaiResp.Model,
		Content:      content,
		ToolCalls:    toolCalls,
		FinishReason: choice.FinishReason,
		Usage: models.TokenUsage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		},
	}, nil
}

func (p *Provider) Stream(ctx context.Context, req models.ChatRequest) (<-chan models.StreamChunk, error) {
	openaiReq := p.convertRequest(req)
	openaiReq.Stream = true

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("openai API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	chunks := make(chan models.StreamChunk)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			
			if !bytes.HasPrefix([]byte(line), []byte("data: ")) {
				continue
			}
			
			line = line[6:] // Remove "data: " prefix
			
			if line == "[DONE]" {
				chunks <- models.StreamChunk{Done: true}
				return
			}

			var streamChunk openaiStreamChunk
			if err := json.Unmarshal([]byte(line), &streamChunk); err != nil {
				continue
			}

			if len(streamChunk.Choices) == 0 {
				continue
			}

			choice := streamChunk.Choices[0]
			
			chunk := models.StreamChunk{
				ID:    streamChunk.ID,
				Delta: choice.Delta.Content,
				Done:  choice.FinishReason != nil,
			}

			if choice.FinishReason != nil {
				chunk.FinishReason = *choice.FinishReason
			}

			chunks <- chunk
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

func (p *Provider) convertRequest(req models.ChatRequest) openaiChatRequest {
	messages := make([]openaiMessage, len(req.Messages))
	for i, msg := range req.Messages {
		openaiMsg := openaiMessage{
			Role:       string(msg.Role),
			ToolCallID: msg.ToolCallID,
		}
		
		// Only set content if not empty
		if msg.Content != "" {
			contentStr := msg.Content
			openaiMsg.Content = &contentStr
		}
		
		// Convert tool calls in message
		if len(msg.ToolCalls) > 0 {
			openaiMsg.ToolCalls = make([]openaiToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Function.Arguments)
				openaiMsg.ToolCalls[j] = openaiToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: openaiToolFunction{
						Name:      tc.Function.Name,
						Arguments: string(argsJSON),
					},
				}
			}
		}
		
		messages[i] = openaiMsg
	}

	openaiReq := openaiChatRequest{
		Model:    req.Model,
		Messages: messages,
	}

	// Convert tools
	if len(req.Tools) > 0 {
		openaiReq.Tools = make([]openaiTool, len(req.Tools))
		for i, tool := range req.Tools {
			openaiReq.Tools[i] = openaiTool{
				Type: tool.Type,
				Function: map[string]interface{}{
					"name":        tool.Function.Name,
					"description": tool.Function.Description,
					"parameters":  tool.Function.Parameters,
				},
			}
		}
	}

	return openaiReq
}

func (p *Provider) Model() string {
	return p.model
}
