# Omnitrix Architecture

Based on reverse-engineering OpenCode, here's the complete architecture breakdown.

## Project Structure

```
omnitrix/
├── cmd/
│   └── omnitrix/          # CLI entry point (cobra)
├── internal/
│   ├── app/               # Main application orchestration
│   ├── agents/            # Agent implementations (coder, title, summarizer)
│   ├── tui/               # Bubble Tea TUI components
│   │   ├── components/    # Reusable UI components (chat, dialogs, etc.)
│   │   ├── page/          # Different pages (chat, logs, sessions)
│   │   ├── layout/        # Layout managers
│   │   ├── theme/         # Color themes and styling
│   │   └── util/          # UI utilities
│   ├── llm/               # LLM provider abstraction
│   │   ├── agent/         # Agent orchestration logic
│   │   ├── models/        # Model definitions
│   │   ├── provider/      # Provider implementations (OpenAI, Anthropic, etc.)
│   │   ├── prompt/        # Prompt engineering
│   │   └── tools/         # Tool implementations
│   ├── lsp/               # LSP client implementation
│   │   ├── protocol/      # LSP protocol types
│   │   ├── client.go      # Main LSP client
│   │   ├── methods.go     # LSP methods (completion, definition, etc.)
│   │   └── language.go    # Language detection
│   ├── db/                # Database layer (SQLite)
│   │   ├── migrations/    # SQL migrations
│   │   ├── queries/       # sqlc queries
│   │   └── models.go      # Generated models
│   ├── session/           # Session management
│   ├── message/           # Message handling
│   ├── history/           # File change tracking
│   ├── permission/        # Permission system for file operations
│   ├── config/            # Configuration management
│   ├── diff/              # Diff generation utilities
│   ├── pubsub/            # Event bus for UI updates
│   └── logging/           # Structured logging
└── pkg/
    └── models/            # Shared data structures

```

## Core Components Explained

### 1. Entry Point (cmd/)

**Purpose**: CLI interface using Cobra

Key responsibilities:
- Parse command-line flags (debug, cwd, prompt, output-format)
- Initialize configuration
- Set up database connection
- Create App instance
- Launch TUI or run in non-interactive mode

```go
// Two modes:
// 1. Interactive: Launch Bubble Tea TUI
// 2. Non-interactive: Run single prompt, return result
```

### 2. App (internal/app/)

**Purpose**: Central orchestration layer

```go
type App struct {
    Sessions    session.Service      // Session CRUD
    Messages    message.Service      // Message storage
    History     history.Service      // File change tracking
    Permissions permission.Service   // Approval workflow
    CoderAgent  agent.Service        // Main AI agent
    LSPClients  map[string]*lsp.Client  // LSP per language
}
```

**Key methods**:
- `New()`: Initialize all services, start LSP clients
- `RunNonInteractive()`: Execute single prompt
- `Shutdown()`: Clean up resources

### 3. Agent System (internal/llm/agent/)

**Purpose**: Orchestrate AI conversations with tool calling

**Core Loop**:
```
1. Receive user message
2. Load conversation history from DB
3. Build context (file contents, LSP info, etc.)
4. Send to LLM with available tools
5. If tool calls requested:
   a. Execute tools (with permission checks)
   b. Add results to conversation
   c. Send back to LLM
   d. Repeat until done
6. Save final response to DB
7. Generate title (if first message)
```

**Agent Types**:
- `coder`: Main agent with all tools
- `title`: Generates session titles
- `summarizer`: Compacts old messages

**Streaming**:
```go
// Agent returns a channel for real-time updates
func (a *agent) Run(ctx, sessionID, prompt) (<-chan AgentEvent, error)

type AgentEvent struct {
    Type    AgentEventType  // "response", "error", "summarize"
    Message message.Message
    Error   error
}
```

### 4. Tool System (internal/llm/tools/)

**Purpose**: Functions AI can call to interact with codebase

**Interface**:
```go
type BaseTool interface {
    Info() ToolInfo  // Name, description, parameters
    Run(ctx, params) (ToolResponse, error)
}
```

**Core Tools**:

1. **view**: Read file contents with line numbers
   - Supports offset/limit for large files
   - Includes LSP diagnostics
   - File type detection

2. **write**: Create/modify files
   - Permission system integration
   - Diff generation
   - Conflict detection (check if file modified since last read)
   - File history tracking

3. **bash**: Execute shell commands
   - Permission required
   - Timeout support
   - Working directory awareness

4. **grep**: Search file contents
   - Regex support
   - Context lines
   - Multiple file patterns

5. **ls**: List directory contents
   - Recursive option
   - Pattern filtering
   - Respect .gitignore

6. **glob**: Find files by pattern
   - doublestar patterns (**/*.ts)
   - Exclude patterns

7. **diagnostics**: Get LSP errors/warnings
   - Per-file or all files
   - Severity filtering

**Permission System**:
```go
// Tools request permission before executing
p := w.permissions.Request(
    permission.CreatePermissionRequest{
        SessionID:   sessionID,
        Path:        filePath,
        ToolName:    "write",
        Action:      "write",
        Description: "Create file foo.go",
        Params:      { FilePath, Diff },
    },
)
if !p {
    return ErrorPermissionDenied
}
```

### 5. LSP Integration (internal/lsp/)

**Purpose**: Code intelligence across all languages

**Architecture**:
- One LSP client per language
- Spawned as child processes
- JSON-RPC over stdin/stdout
- Async message handling

**Client Structure**:
```go
type Client struct {
    Cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout *bufio.Reader
    
    handlers   map[int32]chan *Message  // Response handlers
    diagnostics map[DocumentUri][]Diagnostic  // Cache
    openFiles   map[string]*OpenFileInfo
}
```

**Key Methods**:
- `Initialize()`: Handshake with LSP server
- `TextDocumentDidOpen()`: Notify file opened
- `TextDocumentCompletion()`: Get completions
- `TextDocumentDefinition()`: Go to definition
- `TextDocumentHover()`: Get hover info

**Notification Handlers**:
```go
// LSP servers push diagnostics
client.RegisterNotificationHandler(
    "textDocument/publishDiagnostics",
    func(params) {
        // Cache diagnostics
        // Publish to UI via pubsub
    },
)
```

**Language Detection**:
```go
// Map file extensions to LSP servers
".go"   -> "gopls"
".ts"   -> "typescript-language-server"
".py"   -> "pyright"
".rs"   -> "rust-analyzer"
```

### 6. TUI (internal/tui/)

**Purpose**: Beautiful terminal interface using Bubble Tea

**Bubble Tea Pattern**:
```go
type Model interface {
    Init() Cmd                      // Initialize
    Update(Msg) (Model, Cmd)        // Handle events
    View() string                   // Render
}
```

**Main Model**:
```go
type appModel struct {
    width, height   int
    currentPage     page.PageID
    pages           map[page.PageID]tea.Model
    
    // Dialogs
    showPermissions bool
    permissions     dialog.PermissionDialogCmp
    
    showHelp bool
    help     dialog.HelpCmp
    
    showSessionDialog bool
    sessionDialog     dialog.SessionDialog
    
    // ... more dialogs
}
```

**Pages**:
- **Chat**: Main conversation interface
  - Message viewport (scrollable)
  - Input area (textarea)
  - Status bar
  
- **Sessions**: Session picker/manager
- **Logs**: Debug logs viewer
- **Commands**: Command palette

**Components**:
- `chat.ChatCmp`: Message list + input
- `dialog.PermissionDialogCmp`: Approve/deny file operations
- `dialog.SessionDialog`: Switch/create sessions
- `dialog.FilepickerCmp`: Select files to attach
- `core.StatusCmp`: Bottom status bar

**Event Flow**:
```
User Input -> Update() -> Update Model State -> View() -> Render
                 |
                 v
          Send Cmd (async operation)
                 |
                 v
            Cmd returns Msg
                 |
                 v
          Update() called again
```

**Pubsub Integration**:
```go
// Services publish events
pubsub.Broker.Publish(AgentEvent{
    Type: AgentEventTypeResponse,
    Message: msg,
})

// TUI subscribes
ch := pubsub.Broker.Subscribe()
go func() {
    for event := range ch {
        program.Send(event)  // Send to Bubble Tea
    }
}()
```

### 7. Database (internal/db/)

**Purpose**: Persist sessions, messages, file history

**Technology**: SQLite with sqlc for type-safe queries

**Schema**:
```sql
CREATE TABLE sessions (
    id               TEXT PRIMARY KEY,
    parent_session_id TEXT,
    title            TEXT,
    message_count    INTEGER,
    prompt_tokens    INTEGER,
    completion_tokens INTEGER,
    cost             REAL,
    created_at       INTEGER,
    updated_at       INTEGER
);

CREATE TABLE messages (
    id          TEXT PRIMARY KEY,
    session_id  TEXT,
    role        TEXT,  -- "user", "assistant", "tool"
    parts       TEXT,  -- JSON array of content parts
    model       TEXT,
    created_at  INTEGER,
    updated_at  INTEGER,
    finished_at INTEGER
);

CREATE TABLE files (
    id         TEXT PRIMARY KEY,
    session_id TEXT,
    path       TEXT,
    content    TEXT,  -- Original content
    version    TEXT,  -- Current version
    created_at INTEGER,
    updated_at INTEGER
);
```

**sqlc Usage**:
```sql
-- queries/sessions.sql

-- name: GetSession :one
SELECT * FROM sessions WHERE id = ?;

-- name: CreateSession :one
INSERT INTO sessions (...) VALUES (...) RETURNING *;
```

Generates:
```go
func (q *Queries) GetSession(ctx context.Context, id string) (Session, error)
func (q *Queries) CreateSession(ctx context.Context, arg CreateSessionParams) (Session, error)
```

### 8. Configuration (internal/config/)

**Purpose**: Manage settings from multiple sources

**Sources** (priority order):
1. Environment variables
2. Local `.opencode.json` (in project dir)
3. Global config (`~/.config/opencode/config.json`)

**Structure**:
```json
{
  "data": {
    "directory": "~/.local/share/opencode"
  },
  "providers": {
    "openai": {
      "apiKey": "sk-...",
      "disabled": false
    },
    "anthropic": { ... }
  },
  "agents": {
    "coder": {
      "model": "claude-3-5-sonnet-20241022",
      "maxTokens": 8000
    },
    "title": {
      "model": "gpt-4o-mini",
      "maxTokens": 100
    }
  },
  "lsp": {
    "gopls": {
      "command": "gopls",
      "args": ["serve"]
    }
  },
  "contextPaths": [
    ".cursorrules",
    "opencode.md"
  ],
  "tui": {
    "theme": "catppuccin"
  }
}
```

### 9. Message Format (internal/message/)

**Purpose**: Handle multi-modal content (text, images, code, tool results)

**Content Parts**:
```go
type ContentPart interface {
    String() string
}

type TextContent struct {
    Text string
}

type ImageContent struct {
    URL    string
    Base64 string
}

type ToolCallContent struct {
    ToolCallID string
    Name       string
    Input      string
}

type ToolResultContent struct {
    ToolCallID string
    Result     string
    IsError    bool
}
```

**Message Structure**:
```go
type Message struct {
    ID         string
    SessionID  string
    Role       string  // "user", "assistant", "system", "tool"
    Parts      []ContentPart
    Model      string
    CreatedAt  time.Time
}
```

### 10. Provider Abstraction (internal/llm/provider/)

**Purpose**: Unified interface for multiple AI providers

**Interface**:
```go
type Provider interface {
    SendMessages(ctx, messages, tools) (Response, error)
    StreamMessages(ctx, messages, tools) (<-chan StreamChunk, error)
    Model() models.Model
}
```

**Implementations**:
- OpenAI (GPT-4, GPT-3.5)
- Anthropic (Claude)
- Google (Gemini)
- AWS Bedrock
- Groq
- Azure OpenAI
- OpenRouter
- **We'll add**: Ollama, vLLM

**Function Calling**:
```json
{
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "view",
        "description": "Read file contents",
        "parameters": {
          "type": "object",
          "properties": {
            "file_path": {"type": "string"}
          },
          "required": ["file_path"]
        }
      }
    }
  ]
}
```

## Data Flow

### Complete Flow: User Message to Response

```
1. USER INPUT (TUI)
   ↓
2. chat.ChatCmp.Update(tea.KeyMsg{Enter})
   ↓
3. app.CoderAgent.Run(ctx, sessionID, userMessage)
   ↓
4. agent.Run():
   a. Load session history from DB
   b. Read context files (.cursorrules, etc.)
   c. Build message array
   d. Call provider.StreamMessages()
   ↓
5. PROVIDER (Anthropic/OpenAI):
   a. Send HTTP request
   b. Stream response chunks
   ↓
6. TOOL CALLING:
   if response contains tool_calls:
      for each tool_call:
          a. agent calls tool.Run()
          b. tool requests permission (if needed)
          c. TUI shows permission dialog
          d. user approves/denies
          e. tool executes
          f. result added to conversation
      go back to step 4
   ↓
7. FINAL RESPONSE:
   a. Save assistant message to DB
   b. Publish AgentEvent via pubsub
   ↓
8. TUI RECEIVES EVENT:
   a. Update chat viewport
   b. Render new message
   ↓
9. BACKGROUND TASKS:
   a. Generate session title (if first message)
   b. Check if need to summarize old messages
```

## Key Design Patterns

### 1. Service Pattern
```go
// Each domain has a Service interface
type SessionService interface {
    Create(ctx, title) (Session, error)
    Get(ctx, id) (Session, error)
    Update(ctx, session) (Session, error)
    Delete(ctx, id) error
}

// Implementation wraps DB queries
type sessionService struct {
    queries *db.Queries
}
```

### 2. Pubsub for Decoupling
```go
// Services don't know about UI
agent.Publish(AgentEvent{...})

// UI subscribes to events
ch := agent.Subscribe()
```

### 3. Context for Cancellation
```go
// User cancels request
ctx, cancel := context.WithCancel(ctx)
agent.Run(ctx, ...)
// Later...
cancel()  // Stops LLM request mid-stream
```

### 4. Dependency Injection
```go
// App creates all dependencies
app := &App{
    Sessions:    session.NewService(db),
    Messages:    message.NewService(db),
    CoderAgent:  agent.NewAgent(...),
}
```

## Performance Optimizations

1. **Streaming Responses**: Don't wait for full completion
2. **LSP Client Pooling**: Reuse clients per language
3. **Diagnostic Caching**: Avoid redundant LSP calls
4. **SQLite Pragmas**: WAL mode, busy timeout
5. **Context Summarization**: Compact old messages to save tokens

## Security Considerations

1. **Permission System**: All file writes require approval
2. **Command Execution**: User must approve bash commands
3. **Path Validation**: Prevent directory traversal
4. **Sandboxing**: LSP servers run as child processes

## Next Steps for Implementation

Now that you understand the architecture, we'll build Omnitrix with:

1. Start with data models and database
2. Implement provider abstraction (Ollama first)
3. Build basic tool system
4. Create LSP client
5. Implement agent orchestration
6. Build TUI components
7. Wire everything together

Ready to start coding?
