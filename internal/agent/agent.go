package agent

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omnitrix-sh/core.sh/internal/db"
	"github.com/omnitrix-sh/core.sh/internal/providers/ollama"
	"github.com/omnitrix-sh/core.sh/internal/providers/openai"
	"github.com/omnitrix-sh/core.sh/internal/tools"
	"github.com/omnitrix-sh/core.sh/pkg/models"
)

type Agent struct {
	provider models.ProviderType
	model    string
	tools    []tools.Tool
	queries  *db.Queries
	ollama   *ollama.Provider
	openai   *openai.Provider
}

func New(provider models.ProviderType, model, baseURL, apiKey string, queries *db.Queries, availableTools []tools.Tool) *Agent {
	var ollamaProvider *ollama.Provider
	var openaiProvider *openai.Provider

	if provider == models.ProviderOllama {
		ollamaProvider = ollama.NewProvider(baseURL, model)
	} else if provider == models.ProviderOpenAI {
		openaiProvider = openai.NewProvider(apiKey, model)
	}

	return &Agent{
		provider: provider,
		model:    model,
		tools:    availableTools,
		queries:  queries,
		ollama:   ollamaProvider,
		openai:   openaiProvider,
	}
}

func (a *Agent) Chat(ctx context.Context, sessionID, userMessage string) (string, error) {
	messages, err := a.queries.ListMessagesBySession(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to load messages: %w", err)
	}

	modelMessages := make([]models.Message, len(messages))
	for i, msg := range messages {
		modelMessages[i] = models.Message{
			ID:        msg.ID,
			SessionID: msg.SessionID,
			Role:      models.Role(msg.Role),
			Content:   msg.Content,
			CreatedAt: time.Unix(msg.CreatedAt, 0),
		}
	}

	userMsg := models.Message{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      models.RoleUser,
		Content:   userMessage,
		CreatedAt: time.Now(),
	}
	modelMessages = append(modelMessages, userMsg)

	if err := a.saveMessage(ctx, userMsg); err != nil {
		return "", fmt.Errorf("failed to save user message: %w", err)
	}

	modelTools := make([]models.Tool, len(a.tools))
	for i, tool := range a.tools {
		modelTools[i] = tools.ToModelTool(tool)
	}

	// Tool calling loop
	maxIterations := 10
	for i := 0; i < maxIterations; i++ {
		req := models.ChatRequest{
			Model:    a.model,
			Messages: modelMessages,
			Tools:    modelTools,
			Stream:   false,
		}

		var response *models.ChatResponse
		switch a.provider {
		case models.ProviderOllama:
			response, err = a.ollama.Chat(ctx, req)
		case models.ProviderOpenAI:
			response, err = a.openai.Chat(ctx, req)
		default:
			return "", fmt.Errorf("unsupported provider: %s", a.provider)
		}

		if err != nil {
			return "", fmt.Errorf("failed to call provider: %w", err)
		}

		// If content is empty and we have tool calls, set empty string
		content := response.Content
		if content == "" && len(response.ToolCalls) > 0 {
			content = "" // Explicitly empty, not null
		}
		
		assistantMsg := models.Message{
			ID:        uuid.New().String(),
			SessionID: sessionID,
			Role:      models.RoleAssistant,
			Content:   content,
			Model:     response.Model,
			ToolCalls: response.ToolCalls,
			CreatedAt: time.Now(),
		}

		// If no tool calls, we're done
		if len(response.ToolCalls) == 0 {
			if err := a.saveMessage(ctx, assistantMsg); err != nil {
				return "", fmt.Errorf("failed to save assistant message: %w", err)
			}
			return response.Content, nil
		}
		
		// Save assistant message with tool calls
		if err := a.saveMessage(ctx, assistantMsg); err != nil {
			return "", fmt.Errorf("failed to save assistant message: %w", err)
		}

		modelMessages = append(modelMessages, assistantMsg)

		// Execute tool calls
		for _, toolCall := range response.ToolCalls {
			result, err := a.executeTool(ctx, toolCall)

			toolResultMsg := models.Message{
				ID:         uuid.New().String(),
				SessionID:  sessionID,
				Role:       models.RoleTool,
				ToolCallID: toolCall.ID,
				Content:    result,
				CreatedAt:  time.Now(),
			}

			if err != nil {
				toolResultMsg.Content = fmt.Sprintf("Error: %v", err)
			}

			modelMessages = append(modelMessages, toolResultMsg)

			if err := a.saveMessage(ctx, toolResultMsg); err != nil {
				return "", fmt.Errorf("failed to save tool result: %w", err)
			}
		}
	}

	return "", fmt.Errorf("exceeded maximum iterations (%d)", maxIterations)
}

func (a *Agent) executeTool(ctx context.Context, toolCall models.ToolCall) (string, error) {
	var tool tools.Tool
	for _, t := range a.tools {
		if t.Name() == toolCall.Function.Name {
			tool = t
			break
		}
	}

	if tool == nil {
		return "", fmt.Errorf("unknown tool: %s", toolCall.Function.Name)
	}

	result, err := tool.Execute(ctx, toolCall.Function.Arguments)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (a *Agent) Stream(ctx context.Context, sessionID, userMessage string) (<-chan string, error) {
	messages, err := a.queries.ListMessagesBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}

	modelMessages := make([]models.Message, len(messages))
	for i, msg := range messages {
		modelMessages[i] = models.Message{
			ID:        msg.ID,
			SessionID: msg.SessionID,
			Role:      models.Role(msg.Role),
			Content:   msg.Content,
			CreatedAt: time.Unix(msg.CreatedAt, 0),
		}
	}

	userMsg := models.Message{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      models.RoleUser,
		Content:   userMessage,
		CreatedAt: time.Now(),
	}
	modelMessages = append(modelMessages, userMsg)

	if err := a.saveMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	modelTools := make([]models.Tool, len(a.tools))
	for i, tool := range a.tools {
		modelTools[i] = tools.ToModelTool(tool)
	}

	req := models.ChatRequest{
		Model:    a.model,
		Messages: modelMessages,
		Tools:    modelTools,
		Stream:   true,
	}

	var chunks <-chan models.StreamChunk
	switch a.provider {
	case models.ProviderOllama:
		chunks, err = a.ollama.Stream(ctx, req)
	case models.ProviderOpenAI:
		chunks, err = a.openai.Stream(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", a.provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to start streaming: %w", err)
	}

	output := make(chan string)
	go func() {
		defer close(output)

		var fullContent string
		for chunk := range chunks {
			if chunk.Delta != "" {
				fullContent += chunk.Delta
				output <- chunk.Delta
			}

			if chunk.Done {
				assistantMsg := models.Message{
					ID:        uuid.New().String(),
					SessionID: sessionID,
					Role:      models.RoleAssistant,
					Content:   fullContent,
					Model:     a.model,
					CreatedAt: time.Now(),
				}
				a.saveMessage(ctx, assistantMsg)
				return
			}
		}
	}()

	return output, nil
}

func (a *Agent) saveMessage(ctx context.Context, msg models.Message) error {
	_, err := a.queries.CreateMessage(ctx, db.CreateMessageParams{
		ID:        msg.ID,
		SessionID: msg.SessionID,
		Role:      string(msg.Role),
		Content:   msg.Content,
		Model:     sql.NullString{String: msg.Model, Valid: msg.Model != ""},
		CreatedAt: msg.CreatedAt.Unix(),
		UpdatedAt: msg.CreatedAt.Unix(),
	})
	return err
}
