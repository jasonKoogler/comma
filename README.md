# Comma

<div align="center">
  <img src="https://via.placeholder.com/150?text=Comma" alt="Comma Logo" width="150">
  <p><i>AI-powered git commit message generation</i></p>
</div>

Comma is a robust Golang CLI tool that uses AI to analyze your git changes and generate meaningful, conventional commit messages. It integrates with multiple LLM providers and streamlines your git workflow by eliminating the mental overhead of crafting perfect commit messages.

[![Go Report Card](https://goreportcard.com/badge/github.com/username/comma)](https://goreportcard.com/report/github.com/username/comma)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

## Features

- 🤖 **Multiple LLM Providers**: Seamless integration with OpenAI, Anthropic Claude, and local LLMs like Ollama
- 📝 **Conventional Commits**: Generates messages following conventional commit format
- 🔍 **Smart Context Analysis**: Analyzes repository context for more relevant commit messages
- 🔄 **Git Hook Integration**: Install as a prepare-commit-msg hook for automated workflow
- ⚙️ **Rich Configuration**: Extensive customization via config file or command-line flags
- 🌐 **Environment Awareness**: Auto-detects project type and latest commit history

## Installation

### Prerequisites

- Go 1.19 or higher
- Git
- API key for OpenAI or Anthropic (if using those providers)

### From Source

```bash
# Clone the repository
git clone https://github.com/jasonKoogler/comma.git
cd comma

# Build and install
go build -o comma
mv comma /usr/local/bin/
```

### Go Install

```bash
go install github.com/jasonKoogler/comma@latest
```

## Quick Start

```bash
# Configure your LLM provider
comma config set --provider=openai

# Generate a commit message for staged changes
git add .
comma generate

# Or install as a git hook for automatic commit message generation
comma install-hook
```

## Usage Examples

### Generate a commit message for staged changes

```bash
$ git add src/main.go
$ comma generate
Analyzing changes...
Generating commit message...

Generated Commit Message:
-------------------
feat(main): add command-line argument parsing

Implement flag package integration to handle user-defined options
and improve CLI usability with help text and version information.
-------------------
Use this commit message? (y/n): y
✓ Changes committed successfully!
```

### Generate a commit message with specific options

```bash
$ comma generate --provider=anthropic --with-diff --max-tokens=300
```

### View current configuration

```bash
$ comma config view
Current Configuration:
---------------------
Config file: /home/user/.comma/config.yaml
LLM Provider: openai
LLM Model: gpt-4
API Endpoint: https://api.openai.com/v1/chat/completions
Max Tokens: 500
Temperature: 0.70
Include Diff: false

Template:
Generate a concise and meaningful git commit message for the changes.
Follow the conventional commit format: <type>(<scope>): <subject>
...
```

### Update configuration

```bash
$ comma config set --provider=anthropic --model=claude-3-opus-20240229 --temperature=0.5
✓ Configuration updated successfully!
```

### Install git hook

```bash
$ comma install-hook
✓ Hook installed successfully!
```

Now Comma will automatically generate commit messages when you run `git commit`.

## Command Reference

| Command              | Description                                     |
| -------------------- | ----------------------------------------------- |
| `comma generate`     | Generate a commit message based on your changes |
| `comma config view`  | View current configuration                      |
| `comma config set`   | Update configuration values                     |
| `comma install-hook` | Install Comma as a prepare-commit-msg hook      |
| `comma version`      | Show version information                        |

### Generate Command Flags

| Flag            | Description                               | Default            |
| --------------- | ----------------------------------------- | ------------------ |
| `--provider`    | LLM provider to use                       | From config        |
| `--api-key`     | API key for the LLM provider              | From config or env |
| `--template`    | Template for the commit message           | From config        |
| `--max-tokens`  | Maximum number of tokens for the response | 500                |
| `--with-diff`   | Include detailed diff in the prompt       | false              |
| `--edit-prompt` | Edit the prompt before sending to LLM     | false              |
| `--staged`      | Only consider staged changes              | true               |
| `--verbose`     | Enable verbose output                     | false              |

## Configuration

Comma uses a configuration file located at `~/.comma/config.yaml`. You can also specify a different config file using the `--config` flag.

### Sample Configuration

```yaml
llm:
  provider: openai
  api_key: sk-... # Not recommended - use env var instead
  endpoint: https://api.openai.com/v1/chat/completions
  max_tokens: 500
  temperature: 0.7
  model: gpt-4
template: |
  Generate a concise and meaningful git commit message for the changes.
  Follow the conventional commit format: <type>(<scope>): <subject>

  Types: feat, fix, docs, style, refactor, test, chore

  Rules:
  1. First line should be a short summary (max 72 chars)
  2. Use imperative, present tense (e.g., "add" not "added")
  3. Don't end the summary line with a period
  4. Optional body with more detailed explanation (after blank line)

  Changes: 
  {{ .Changes }}
include_diff: false
```

### Environment Variables

Comma respects environment variables with the prefix `COMMA_`:

- `COMMA_LLM_PROVIDER`
- `COMMA_LLM_API_KEY`
- `COMMA_LLM_ENDPOINT`
- `COMMA_LLM_MAX_TOKENS`
- `COMMA_LLM_TEMPERATURE`
- `COMMA_LLM_MODEL`

Additionally, Comma will look for provider-specific API keys:

- `OPENAI_API_KEY` (when using OpenAI)
- `ANTHROPIC_API_KEY` (when using Anthropic)

## LLM Providers

### OpenAI

OpenAI's GPT models (GPT-4 recommended) provide excellent commit message generation. To use:

```bash
comma config set --provider=openai --model=gpt-4
```

Ensure you have set either `COMMA_LLM_API_KEY` or `OPENAI_API_KEY` environment variable.

### Anthropic Claude

Anthropic's Claude models excel at understanding context and generating natural-sounding commit messages:

```bash
comma config set --provider=anthropic --model=claude-3-opus-20240229
```

Ensure you have set either `COMMA_LLM_API_KEY` or `ANTHROPIC_API_KEY` environment variable.

### Local LLMs (Ollama)

For privacy or offline usage, Comma supports local LLMs through Ollama:

```bash
# Start Ollama locally first
ollama run llama3

# Configure Comma to use local LLM
comma config set --provider=local --model=llama3 --endpoint=http://localhost:11434/api/generate
```

## Template Variables

You can customize the prompt template using variables:

- `{{ .Changes }}` - The git changes (files and diff)
- `{{ .Context.RepoName }}` - The repository name
- `{{ .Context.CurrentBranch }}` - The current branch name
- `{{ .Context.LastCommitMsg }}` - The last commit message
- `{{ .Context.FileTypes }}` - List of file extensions in the repo
- `{{ .Context.ProjectType }}` - Detected project type (Go, Python, etc.)
- `{{ .Context.CommitHistory }}` - Recent commit messages

## Advanced Usage

### Creating Custom Templates

Create a custom template focused on specific needs:

```bash
comma config set --template="
You are a commit message generator for a security-focused project.

Please analyze these changes and generate a commit message that:
1. Follows conventional commit format
2. Highlights security implications if any
3. References related security standards when applicable

Changes:
{{ .Changes }}
"
```

### Different Messages for Different Repositories

Use git aliases for different repositories:

```bash
# In your .gitconfig
[alias]
  commit-feature = "!comma generate --template=/path/to/feature-template.txt"
  commit-bugfix = "!comma generate --template=/path/to/bugfix-template.txt"
```

### Continuous Integration Integration

Use Comma for automated commits in CI pipelines:

```bash
# In your CI script
git add .
COMMIT_MSG=$(comma generate --staged --provider=openai)
git commit -m "$COMMIT_MSG"
git push
```

## Troubleshooting

### API Key Issues

If you encounter authentication errors:

```
Error: failed to generate commit message: API error: Invalid authentication
```

Check your API key is correctly set:

```bash
# For OpenAI
export OPENAI_API_KEY=sk-...

# For Anthropic
export ANTHROPIC_API_KEY=sk-...
```

### Rate Limiting

If you encounter rate limiting:

```
Error: API returned non-200 status: 429
```

Comma has built-in retries with exponential backoff, but you may need to wait or switch providers.

### Model Errors

If the model returns errors:

```
Error: API error: Model overloaded
```

Try a different model or reduce complexity:

```bash
comma config set --model=gpt-3.5-turbo --max-tokens=300
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgements

- [OpenAI](https://openai.com/) for their powerful language models
- [Anthropic](https://www.anthropic.com/) for Claude
- [Cobra](https://github.com/spf13/cobra) and [Viper](https://github.com/spf13/viper) libraries

---

Built with ❤️ using Comma itself
