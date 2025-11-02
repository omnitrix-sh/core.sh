package models

import "time"

// Provider types
type ProviderType string

const (
	ProviderOllama       ProviderType = "ollama"
	ProviderHuggingFace  ProviderType = "huggingface"
	ProviderVLLM         ProviderType = "vllm"
	ProviderOpenAI       ProviderType = "openai"      // Optional
	ProviderAnthropic    ProviderType = "anthropic"   // Optional
)

// Model represents an AI model configuration
type Model struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Provider     ProviderType `json:"provider"`
	ContextSize  int          `json:"context_size"`
	MaxTokens    int          `json:"max_tokens"`
	Capabilities Capabilities `json:"capabilities"`
}

// Capabilities defines what a model can do
type Capabilities struct {
	FunctionCalling bool `json:"function_calling"`
	Streaming       bool `json:"streaming"`
	Vision          bool `json:"vision"`
	CodeExecution   bool `json:"code_execution"`
}

// Message represents a conversation message
type Message struct {
	ID         string       `json:"id"`
	SessionID  string       `json:"session_id"`
	Role       Role         `json:"role"`
	Content    string       `json:"content"`
	Parts      []ContentPart `json:"parts,omitempty"`
	ToolCalls  []ToolCall   `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
	Model      string       `json:"model,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

// Role in conversation
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ContentPart for multi-modal content
type ContentPart interface {
	Type() string
	String() string
}

// TextPart is text content
type TextPart struct {
	Text string `json:"text"`
}

func (t TextPart) Type() string   { return "text" }
func (t TextPart) String() string { return t.Text }

// ImagePart is image content
type ImagePart struct {
	URL    string `json:"url,omitempty"`
	Base64 string `json:"base64,omitempty"`
}

func (i ImagePart) Type() string   { return "image" }
func (i ImagePart) String() string { return "[Image]" }

// ToolCall represents a function call by the AI
type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // "function"
	Function FunctionCall           `json:"function"`
}

// FunctionCall details
type FunctionCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult is the result of a tool execution
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error"`
}

// Session represents a conversation session
type Session struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Model            string    `json:"model"`
	Provider         string    `json:"provider"`
	MessageCount     int       `json:"message_count"`
	PromptTokens     int64     `json:"prompt_tokens"`
	CompletionTokens int64     `json:"completion_tokens"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// FileChange tracks file modifications
type FileChange struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	FilePath  string    `json:"file_path"`
	Operation string    `json:"operation"` // "create", "modify", "delete"
	OldContent string   `json:"old_content"`
	NewContent string   `json:"new_content"`
	Diff      string    `json:"diff"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatRequest for AI providers
type ChatRequest struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float32   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream"`
	Tools       []Tool    `json:"tools,omitempty"`
}

// ChatResponse from AI providers
type ChatResponse struct {
	ID           string     `json:"id"`
	Model        string     `json:"model"`
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string     `json:"finish_reason"`
	Usage        TokenUsage `json:"usage"`
}

// StreamChunk for streaming responses
type StreamChunk struct {
	ID           string     `json:"id"`
	Delta        string     `json:"delta"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string     `json:"finish_reason,omitempty"`
	Done         bool       `json:"done"`
}

// TokenUsage tracking
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Tool definition for function calling
type Tool struct {
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a callable function
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Config for the application
type Config struct {
	// Data storage
	DataDir string `json:"data_dir"`

	// Working directory
	WorkDir string `json:"work_dir"`

	// AI Providers
	Providers map[ProviderType]ProviderConfig `json:"providers"`

	// Default model
	DefaultModel string `json:"default_model"`
	
	// Default provider
	DefaultProvider string `json:"default_provider"`

	// LSP configurations
	LSP map[string]LSPConfig `json:"lsp"`

	// Context files to include
	ContextPaths []string `json:"context_paths"`

	// Debug mode
	Debug bool `json:"debug"`
}

// ProviderConfig for each AI provider
type ProviderConfig struct {
	Enabled  bool   `json:"enabled"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key,omitempty"`
	Models   []string `json:"models,omitempty"`
}

// LSPConfig for language servers
type LSPConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Enabled bool     `json:"enabled"`
}
