# Omnitrix

a claude cli alternative with no vendor lock-in, inspired by opencode, warp and cursor

## What is this?

Think of it as having a really helpful developer sitting next to you, except they run entirely on your machine. Omnitrix connects to local AI models (like those from Ollama) and helps you with your code. It can read files, write code, explain things, and generally make your life easier.

The best part? Everything runs locally. Your code never leaves your machine.

## Quick Start

### Prerequisites

First, get Ollama running:

```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull a coding model (this one's pretty good and runs on most machines)
ollama pull deepseek-coder:6.7b
```

### Installation

```bash
# Clone this repo
git clone https://github.com/omnitrix-sh/core.sh
cd core.sh

# Build it
go build -o omnitrix ./cmd/omnitrix

# Try it out
./omnitrix "what files are in this directory?"
```

## How to Use

Just ask it questions like you would a coworker:

```bash
./omnitrix "show me the main.go file"
./omnitrix "create a hello world in Python"
./omnitrix "explain what this project does"
```

It remembers your conversation, so you can have back-and-forth chats.

## Configuration

Drop a `.omnitrix.json` in your project folder:

```json
{
  "default_model": "deepseek-coder:6.7b",
  "providers": {
    "ollama": {
      "enabled": true,
      "base_url": "http://localhost:11434"
    }
  }
}
```

Or just let it use the defaults. It'll figure things out.

## What Can It Do?

Right now, pretty basic stuff:
- Read and write files
- List directories
- Remember your conversation
- Stream responses so you see them as they happen

Coming soon:
- Better terminal UI
- Code intelligence (via LSP)
- More tools (grep, git commands, running tests)
- Support for cloud models if you want them

## Why Build This?

Because AI coding assistants are amazing, but most of them:
- Cost money
- Send your code to the cloud
- Don't let you choose your model
- Can't be customized

Omnitrix fixes all of that. It's yours to modify, extend, and make your own.

## Tech Stack

- **Go** - Fast, simple, gets the job done
- **SQLite** - For keeping conversation history
- **Ollama** - Runs AI models locally
- **Open Source Models** - DeepSeek Coder, Qwen, CodeLlama, etc.

## Contributing

Found a bug? Want to add a feature? PRs are welcome!

The code is pretty straightforward:
- `internal/agent/` - The brain that coordinates everything
- `internal/tools/` - Things the AI can do (read files, write code, etc.)
- `internal/providers/` - Talks to AI models
- `cmd/omnitrix/` - The CLI you interact with

## License

MIT - Do whatever you want with it

## Credits

Inspired by projects like OpenCode, Aider, and Warp.dev . Built with Bubble Tea for the eventual fancy TUI.

---

**Status**: Early but working! Expect bugs, breaking changes, and the occasional weird behavior. But it does work, and it's actually pretty useful.
