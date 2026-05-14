# Intro (written by a human)

What happens if you ask an LLM to 'create an OpenAI chat TUI in Golang'? Almost certainly nothing, or maybe not.  
Seriously, this is my attempt to create, with the help of AI, an agent for coding from scratch. The main reasons:

- pure fun
- studying in detail how agents and LLMs work

# WARNING: From now on, content may be AI-generated.

```
 ___                    _    ___
| __|_ __  _ __   __ _ (_)  |_ _|
| _|| '  \| '  \ / _` || |   | |
|___|_|_|_|_|_|_|\__,_||_|  |___|
    Terminal AI Chat Interface
```

# EmmAI - Terminal AI Chat Interface

A terminal user interface (TUI) for chatting with any OpenAI-compatible API server.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)

## Features

- **Minimal TUI**: Clean, intuitive interface built with [tview](https://github.com/rivo/tview)
- **Streaming Responses**: Real-time streaming of AI responses with typing effect
- **Conversation Persistence**: Automatically saves and loads conversations
- **Custom Endpoints**: Compatible with any OpenAI-compatible API server (remote and local)
- **Keyboard Shortcuts**: Efficient navigation and control
- **Token Tracking**: Real-time display of token usage
- **Optimized for local llm**: Sane system prompts for efficiency without sacrificing functionality (or at least we try)

## Prerequisites

- Go 1.21 or higher
- OpenAI-compatible endpoint

## Installation

### Using Go Install

```bash
go install github.com/ilmich/emmai/cmd/emmai@latest
```

### From Source

```bash
# Clone the repository
git clone https://github.com/ilmich/emmai.git
cd emmai

# Build and install
make install

# Or just build
make build
./bin/emmai
```

## Configuration

### Method 1: Environment Variable (Recommended)

```bash
export OPENAI_API_KEY="sk-..."
emmai
```

### Method 2: Configuration File

Create a config file at `~/.emmai/config.yaml`:

```yaml
# API Key (optional if using environment variable)
api_key: sk-...

# Model selection (default: gpt-3.5-turbo)
model: gpt-4

# Temperature: creativity level 0.0-2.0 (default: 0.7)
temperature: 0.7

# Max tokens per response (default: 2048)
max_tokens: 2048

# System prompt: sets AI behavior
system_prompt: "You are a helpful assistant focused on clear, concise explanations."
```

See [configs/config.example.yaml](configs/config.example.yaml) for a template.

## Custom API Endpoints

EmmAI supports custom OpenAI-compatible API endpoints, making it compatible with:
- **llama.cpp** - Run LLMs locally
- **Ollama** - Local model hosting
- **LM Studio** - Desktop LLM runner
- **Any OpenAI-compatible API**

### Configuration

Add to your `~/.emmai/config.yaml`:

```yaml
base_url: "http://localhost:8080"  # Your custom endpoint
model: "llama2"                     # Model name from your provider
api_key: ""                         # Optional for local endpoints

# Optional: skip certificate verification (use with caution)
insecure_skip_verify: false
```

Or use environment variables:

```bash
export OPENAI_BASE_URL="http://localhost:8080"
export OPENAI_API_KEY=""  # Empty for local endpoints
emmai
```

### API Key Requirements

| Endpoint Type | API Key Required? |
|---------------|-------------------|
| OpenAI (api.openai.com) | ✅ Yes (required) |
| LocalAI, Ollama, etc. | ❌ No (optional) |

**Note**: For custom endpoints, set `OPENAI_API_KEY=""` (empty string) or omit from config.

## Usage

### Starting the Application

```bash
# With environment variable
export OPENAI_API_KEY="sk-..."
emmai

# With config file
emmai
```

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Enter` | Send message |
| `Shift+Enter` | New line in input |
| `ESC` | Stop streaming response |
| `Ctrl+L` | Clear conversation (start new) |
| `Ctrl+R` | Retry last message |
| `Ctrl+Q` | Quit application |
| `Ctrl+C` | Force quit |
| `Page Up/Down` | Scroll chat history |

## Data Storage

### Conversations

Conversations are automatically saved to:
```
~/.emmai/conversations/{conversation_id}.json
```

Each conversation includes:
- Complete message history
- Model used
- Timestamps
- Unique conversation ID

## Contributing

Contributions are welcome! Please follow Go best practices and include tests for new features.

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linters
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) for details

## Acknowledgments

- Built with [tview](https://github.com/rivo/tview) by rivo
- Uses [go-openai](https://github.com/sashabaranov/go-openai) SDK
- Follows [Standard Go Project Layout](https://github.com/golang-standards/project-layout)

## Support

- **Issues**: [GitHub Issues](https://github.com/ilmich/emmai/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ilmich/emmai/discussions)

## Roadmap

Future enhancements:
- [ ] Model switching via keyboard shortcut
- [ ] Export conversations to markdown
- [ ] Search within conversations
- [ ] Message editing and regeneration
- [ ] Cost tracking and estimates
- [ ] Syntax highlighting for code blocks
- [ ] Custom keybindings configuration

---

Made with ❤️ using Go
