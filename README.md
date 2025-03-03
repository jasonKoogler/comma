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
- 🔒 **Security Scanning**: Detects sensitive data in your changes to prevent credential leaks
- 📊 **Repository Analysis**: Get insights into your commit patterns and conventions
- 💼 **Team Settings**: Share templates and enforce commit conventions across your team
- 🎨 **Terminal UIs**: Interactive interfaces for browsing changes, editing commit messages, and configuring the application

## Installation

### Prerequisites

- Go 1.19 or higher
- Git
- API key for OpenAI or Anthropic (if using those providers)

### From Source

```bash
# Clone the repository
git clone https://github.com/username/comma.git
cd comma

# Build and install
go build -o comma
mv comma /usr/local/bin/
```

### Go Install

```bash
go install github.com/username/comma@latest
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

### Use the interactive terminal UIs

```bash
# Interactive commit message UI
$ comma tui

# Interactive configuration UI
$ comma setup
```

### Analyze repository commit patterns

```bash
$ comma analyze
Analyzing repository commit patterns...

Repository Statistics:
---------------------
Total commits: 283
Contributors: 7
Conventional commits: 87.5%
Average message length: 68.1 chars

Commit Types:
  feat: 42 (14.8%)
  fix: 78 (27.6%)
  docs: 15 (5.3%)
  refactor: 28 (9.9%)
  test: 14 (4.9%)

Suggestions:
- Add scopes to your commits for better organization
```

## Command Reference

| Command              | Description                                     |
| -------------------- | ----------------------------------------------- |
| `comma generate`     | Generate a commit message based on your changes |
| `comma config view`  | View current configuration                      |
| `comma config set`   | Update configuration values                     |
| `comma install-hook` | Install Comma as a prepare-commit-msg hook      |
| `comma tui`          | Launch interactive terminal UI                  |
| `comma analyze`      | Analyze repository commit patterns              |
| `comma setup`        | Interactive configuration UI                    |
| `comma enterprise`   | Enterprise management features                  |
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
| `--skip-scan`   | Skip security scanning                    | false              |
| `--no-cache`    | Bypass commit cache                       | false              |

## Configuration

Comma uses a configuration file located at `~/.comma/config.yaml`. You can also specify a different config file using the `--config` flag.

### Interactive Configuration

You can configure Comma interactively using the setup command, which provides a user-friendly terminal UI:

```bash
$ comma setup
```

This opens an interactive configuration interface where you can navigate through different settings categories, modify values, and save your configuration without having to edit YAML files directly.

![Config TUI Screenshot](https://via.placeholder.com/600x400?text=Config+TUI+Screenshot)

### Sample Configuration File

```yaml
llm:
  provider: openai
  api_key: sk-... # Not recommended - use env var instead
  endpoint: https://api.openai.com/v1/chat/completions
  max_tokens: 500
  temperature: 0.7
  model: gpt-4
  use_local_fallback: true
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
analysis:
  enable_smart_detection: true
  suggest_scopes: true
security:
  scan_for_sensitive_data: true
  enable_audit_logging: true
cache:
  enabled: true
  max_age_hours: 24
ui:
  syntax_highlight: true
  theme: monokai
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

## Advanced Features

### Secure Credential Management

Comma stores API keys securely using your system's credential storage or encrypted files as a fallback. This ensures that your API keys are never stored in plaintext.

### Sensitive Data Detection

Before sending changes to an LLM provider, Comma scans for sensitive information such as API keys, passwords, and connection strings to prevent accidental disclosure of secrets.

```bash
$ git add config.json
$ comma generate

⚠️  Security Warning: Sensitive data detected in changes!
The following issues were found:
1. AWS Key (HIGH)
   Line: AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
   Suggestion: Store AWS credentials using environment variables or AWS credential providers

Do you want to continue with these issues? (y/n):
```

### Smart Commit Type Detection

Comma analyzes your changes to automatically detect the most appropriate commit type and scope based on file patterns, content changes, and repository context.

### Team Templates and Conventions

Enable team settings to share templates and enforce commit message conventions across your team:

```bash
# Create a team configuration
$ comma enterprise team create --name engineering --description "Engineering Team"

# Import existing team configuration
$ comma enterprise team import team-config.json

# Use team settings for commit generation
$ comma generate --team
```

### Local LLM Support

Comma can use local LLMs like Ollama for offline usage or as a fallback when API providers are unavailable:

```bash
# Configure local LLM provider
$ comma config set --provider=local --model=llama3

# Enable local fallback
$ comma config set --use-local-fallback=true
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
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the terminal UI

---

Built with ❤️ using Comma itself
