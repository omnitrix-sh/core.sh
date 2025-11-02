# Omnitrix

A powerful AI coding agent built for the terminal. Fully open source, privacy-first, with support for local and self-hosted models.

## Vision

Build the most powerful coding assistant that:
- Works entirely with open source models
- Runs locally for complete privacy
- Provides deep code intelligence via LSP
- Offers a beautiful terminal experience
- Beats SWE-bench with specialized fine-tuning

## Architecture

Omnitrix is built as a modular system with clean separation:

```
Terminal UI (Bubble Tea)
         ↓
   Agent Orchestrator
         ↓
    ┌────┴────┐
    ↓         ↓
  Tools     LLM Providers
    ↓         ↓
  LSP    Open Source Models
```

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed breakdown.

## Supported Models

### Local (Ollama)
- DeepSeek Coder V2 (33B, 6.7B)
- Qwen 2.5 Coder (32B, 7B)
- CodeLlama (34B, 13B)

### Hosted (HuggingFace Inference)
- DeepSeek-Coder-V2-Instruct
- Qwen2.5-Coder-32B-Instruct
- CodeLlama variants

### Self-Hosted (vLLM)
- Any model compatible with OpenAI API

### Optional Premium
- OpenAI GPT-4
- Anthropic Claude

## Features

### Current
- Model abstractions for multiple providers
- Configuration system
- Data models for sessions/messages

### Planned (In Order)
1. SQLite database with migrations
2. Ollama provider implementation
3. Basic tool system (read, write, bash)
4. LSP client for code intelligence
5. Agent orchestration with tool calling
6. Basic TUI with chat interface
7. Permission system for file operations
8. Session management
9. HuggingFace provider
10. Advanced features (diff viewer, file picker, etc.)

## Quick Start

### Prerequisites
```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull a model
ollama pull deepseek-coder:6.7b
```

### Installation
```bash
# Clone
git clone https://github.com/omnitrix-sh/core.sh
cd core.sh

# Build
go build -o omnitrix ./cmd/omnitrix

# Run
./omnitrix
```

## Configuration

Config file: `.omnitrix.json` (project) or `~/.config/omnitrix/config.json` (global)

```json
{
  "default_model": "deepseek-coder:33b",
  "providers": {
    "ollama": {
      "enabled": true,
      "base_url": "http://localhost:11434"
    },
    "huggingface": {
      "enabled": true,
      "api_key": "hf_..."
    }
  }
}
```

## Development Roadmap

### Phase 1: Foundation (Week 1-2)
- [x] Project structure
- [x] Data models
- [x] Configuration system
- [ ] Database layer (SQLite + sqlc)
- [ ] Ollama provider

### Phase 2: Core Agent (Week 3-4)
- [ ] Tool system (read, write, bash, grep, ls)
- [ ] Agent orchestration
- [ ] Permission system
- [ ] Session management

### Phase 3: Intelligence (Week 5-6)
- [ ] LSP integration
- [ ] Code diagnostics
- [ ] Smart context building

### Phase 4: UI (Week 7-8)
- [ ] Basic TUI
- [ ] Chat interface
- [ ] Permission dialogs
- [ ] Session picker

### Phase 5: Advanced (Week 9-12)
- [ ] HuggingFace provider
- [ ] File picker
- [ ] Diff viewer
- [ ] Command palette
- [ ] Themes

### Phase 6: SaaS (Week 13+)
- [ ] Benchmark harness
- [ ] Fine-tuning pipeline
- [ ] vLLM integration
- [ ] REST API
- [ ] Web client

## Why Open Source Models?

1. **Privacy**: Code never leaves your machine
2. **Cost**: No API fees, unlimited usage
3. **Customization**: Fine-tune for your needs
4. **Performance**: Recent models (DeepSeek, Qwen) rival GPT-4 on coding
5. **Control**: Host your own inference

## Benchmarks

Target: Beat SWE-bench with specialized fine-tuning

Current SOTA (as of 2024):
- Claude Sonnet 3.5: ~49% pass@1
- GPT-4: ~42% pass@1
- DeepSeek Coder V2: ~45% pass@1

Our goal: 55%+ through:
- Specialized prompting
- Multi-step reasoning
- LSP-guided code understanding
- Test-driven iteration

## Contributing

We welcome contributions! Areas of focus:

1. Provider implementations (Ollama, HF, vLLM)
2. Tool development (new file operations)
3. LSP integrations (more languages)
4. TUI components (themes, widgets)
5. Benchmarking (SWE-bench, HumanEval)

## License

MIT

## Credits

Inspired by:
- [OpenCode](https://github.com/opencode-ai/opencode)
- [Aider](https://github.com/paul-gauthier/aider)
- [Continue](https://github.com/continuedev/continue)

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Ollama](https://ollama.com) - Local model serving
- [SQLite](https://www.sqlite.org) - Database
- [sqlc](https://sqlc.dev) - Type-safe SQL
