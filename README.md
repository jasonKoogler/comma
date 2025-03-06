# COMMA

Comma is an AI-powered git commit message generator that analyzes your staged
changes and produces meaningful, conventional commit messages.

## FEATURES

- AI-Powered Generation: Uses OpenAI or Anthropic APIs to generate meaningful commit messages
- Multiple LLM Support: Works with OpenAI, Anthropic, and local LLMs
- Secure Credential Storage: Securely stores API keys in your system's keyring
- Conventional Commits: Follows the Conventional Commits specification
- Repository Analysis: Analyzes your repository for patterns and style consistency
- Custom Templates: Define your own prompt templates for message generation
- Command-line Interface: Simple, intuitive CLI for all operations

## INSTALLATION

### From Source:

```bash
  # Clone the repository
  git clone https://github.com/jasonKoogler/comma.git
  cd comma

  # Build the application
  make build

  # Install globally (optional)
  make install
```

### Binary Releases:

Download the appropriate binary for your system from the Releases page:
https://github.com/jasonKoogler/comma/releases

## USAGE

### Setup:

Configure Comma with your preferred LLM provider:

```bash
  comma setup
```

This interactive setup will guide you through configuring your provider, API key, and model.

Generate Commit Messages:

```bash
  # Stage your changes first
  git add .

  # Generate a commit message
  comma generate

  # Use with specific model
  comma generate --model gpt-4

  # Include diff details
  comma generate --with-diff
```

Repository Analysis:

```bash
  # Analyze last 100 commits
  comma analyze

  # Analyze commits from the last 30 days
  comma analyze --days 30
```

Configuration Management:

```bash
  # View current configuration
  comma config view

  # Set configuration values
  comma config set --provider openai
  comma config set --model gpt-4-turbo
```

## SECURITY

Comma prioritizes the security of your API keys:

- API keys are stored in your system's secure keyring when available
- Falls back to encrypted local storage when system keyring isn't accessible
- Environment variables are supported (e.g., OPENAI_API_KEY, ANTHROPIC_API_KEY)
- Avoids storing keys in plain text

## CONFIGURATION

Configuration is stored in ~/.comma/config.yaml. You can edit this file directly
or use the `comma config set` command.

### Default Template:

```
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
```

## SUPPORTED LLM PROVIDERS AND MODELS

#### OpenAI:

- gpt-4o
- gpt-4-turbo
- gpt-4
- gpt-3.5-turbo

#### Anthropic:

- claude-3-opus-20240229
- claude-3-sonnet-20240229
- claude-3-haiku-20240307
- claude-3.5-sonnet
- claude-3-7-sonnet-latest

#### Local (requires setup):

- llama3
- llama2
- mixtral
- mistral
- phi3
