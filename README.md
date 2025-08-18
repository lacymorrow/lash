# Lash

<p align="center">
    <a href="https://stuff.charm.sh/crush/charm-crush.png"><img width="450" alt="Charm Crush Logo" src="https://github.com/user-attachments/assets/adc1a6f4-b284-4603-836c-59038caa2e8b" /></a><br />
    <a href="https://github.com/lacymorrow/lash/releases"><img src="https://img.shields.io/github/release/lacymorrow/lash" alt="Latest Release"></a>
    <a href="https://github.com/lacymorrow/lash/actions"><img src="https://github.com/lacymorrow/lash/workflows/build/badge.svg" alt="Build Status"></a>
</p>

Terminal-based AI assistant for developers. A login-shell-friendly fork of Charmbracelet Crush with Shell, Agent, and Auto modes, plus built-in MCP support.

### Features

- **Multi-Model:** choose from a wide range of LLMs or add your own via OpenAI- or Anthropic-compatible APIs
- **Flexible:** switch LLMs mid-session while preserving context
- **Session-Based:** maintain multiple work sessions and contexts per project
- **LSP-Enhanced:** uses LSPs for additional context, just like you do
- **Extensible:** add capabilities via MCPs (`http`, `stdio`, and `sse`)
- **Works Everywhere:** first-class support in terminals on macOS, Linux, and Windows (PowerShell and WSL)
- **Modes:** Shell, Agent, and Auto routing (first run defaults to Auto)

### Installation

NPM:

```bash
npm install -g @lacymorrow/lash
lash --version
```

Homebrew:

```bash
brew tap lacymorrow/tap
brew install lacymorrow/tap/lash
lash --version
```

Windows users:

```bash
# Winget
winget install charmbracelet.crush

# Scoop
scoop bucket add charm https://github.com/charmbracelet/scoop-bucket.git
scoop install crush
```

<details>
<summary><strong>Nix (NUR)</strong></summary>

Crush is available via [NUR](https://github.com/nix-community/NUR) in `nur.repos.charmbracelet.crush`.

You can also try out Crush via `nix-shell`:

```bash
# Add the NUR channel.
nix-channel --add https://github.com/nix-community/NUR/archive/main.tar.gz nur
nix-channel --update

# Get Crush in a Nix shell.
nix-shell -p '(import <nur> { pkgs = import <nixpkgs> {}; }).repos.charmbracelet.crush'
```

</details>

<details>
<summary><strong>Debian/Ubuntu</strong></summary>

```bash
# Download .deb package from GitHub releases
wget https://github.com/lacymorrow/lash/releases/latest/download/lash_Linux_x86_64.deb
sudo dpkg -i lash_Linux_x86_64.deb

# Or download and extract binary directly
wget https://github.com/lacymorrow/lash/releases/latest/download/lash_Linux_x86_64.tar.gz
tar -xzf lash_Linux_x86_64.tar.gz
sudo mv lash/lash /usr/local/bin/
```

</details>

<details>
<summary><strong>Fedora/RHEL</strong></summary>

```bash
echo '[charm]
name=Charm
baseurl=https://repo.charm.sh/yum/
enabled=1
gpgcheck=1
gpgkey=https://repo.charm.sh/yum/gpg.key' | sudo tee /etc/yum.repos.d/charm.repo
sudo yum install crush
```

</details>

Or, download it:

- [Packages][releases] are available in Debian and RPM formats
- [Binaries][releases] are available for Linux, macOS, Windows, FreeBSD, OpenBSD, and NetBSD

[releases]: https://github.com/lacymorrow/lash/releases

Or just install it with Go:

```
go install github.com/lacymorrow/lash@latest
```

> [!WARNING]
> Productivity may increase when using Lash and you may find yourself nerd
> sniped when first using the application. If the symptoms persist, join the
> [Discord][discord] and nerd snipe the rest of us.

### Getting Started

The quickest way to get started is to grab an API key for your preferred
provider such as Anthropic, OpenAI, Groq, or OpenRouter and run `lash`. You'll be prompted to enter your API key.

That said, you can also set environment variables for preferred providers.

| Environment Variable       | Provider                                           |
| -------------------------- | -------------------------------------------------- |
| `ANTHROPIC_API_KEY`        | Anthropic                                          |
| `OPENAI_API_KEY`           | OpenAI                                             |
| `OPENROUTER_API_KEY`       | OpenRouter                                         |
| `GEMINI_API_KEY`           | Google Gemini                                      |
| `VERTEXAI_PROJECT`         | Google Cloud VertexAI (Gemini)                     |
| `VERTEXAI_LOCATION`        | Google Cloud VertexAI (Gemini)                     |
| `GROQ_API_KEY`             | Groq                                               |
| `AWS_ACCESS_KEY_ID`        | AWS Bedrock (Claude)                               |
| `AWS_SECRET_ACCESS_KEY`    | AWS Bedrock (Claude)                               |
| `AWS_REGION`               | AWS Bedrock (Claude)                               |
| `AZURE_OPENAI_ENDPOINT`    | Azure OpenAI models                                |
| `AZURE_OPENAI_API_KEY`     | Azure OpenAI models (optional when using Entra ID) |
| `AZURE_OPENAI_API_VERSION` | Azure OpenAI models                                |

### Models Catalog

Lash uses the Catwalk model catalog from the upstream project for defaults. You can override or add providers in your configuration.

### Configuration

Lash runs great with no configuration. If you do want to customize it, configuration follows the upstream file names for compatibility and is read with the following priority:

1. `.crush.json`
2. `crush.json`
3. `$HOME/.config/crush/crush.json` (Windows: `%USERPROFILE%\AppData\Local\crush\crush.json`)

Configuration itself is stored as a JSON object:

```json
{
   "this-setting": {"this": "that"},
   "that-setting": ["ceci", "cela"]
}
```

Lash stores ephemeral data, such as application state, in this location:

```bash
# Project-relative (default)
./.lash/
```

### LSPs

Lash can use LSPs for additional context. LSPs can be added manually like so:

```json
{
  "$schema": "https://charm.land/crush.json",
  "lsp": {
    "go": {
      "command": "gopls",
      "env": {
        "GOTOOLCHAIN": "go1.24.5"
      }
    },
    "typescript": {
      "command": "typescript-language-server",
      "args": ["--stdio"]
    },
    "nix": {
      "command": "nil"
    }
  }
}
```

### MCPs

Lash supports Model Context Protocol (MCP) servers through three
transport types: `stdio` for command-line servers, `http` for HTTP endpoints,
and `sse` for Server-Sent Events. Environment variable expansion is supported
using `$(echo $VAR)` syntax.

```json
{
  "$schema": "https://charm.land/crush.json",
  "mcp": {
    "filesystem": {
      "type": "stdio",
      "command": "node",
      "args": ["/path/to/mcp-server.js"],
      "env": {
        "NODE_ENV": "production"
      }
    },
    "github": {
      "type": "http",
      "url": "https://example.com/mcp/",
      "headers": {
        "Authorization": "$(echo Bearer $EXAMPLE_MCP_TOKEN)"
      }
    },
    "streaming-service": {
      "type": "sse",
      "url": "https://example.com/mcp/sse",
      "headers": {
        "API-Key": "$(echo $API_KEY)"
      }
    }
  }
}
```

### Ignoring Files

Lash respects `.gitignore` by default. You can also create a `.crushignore` file to specify additional files and directories that should be ignored when providing context.

The `.crushignore` file uses the same syntax as `.gitignore` and can be placed
in the root of your project or in subdirectories.

### Allowing Tools

By default, Lash will ask you for permission before running tool calls. If you'd like, you can allow tools to be executed without prompting you for permissions. Use this with care.

```json
{
  "$schema": "https://charm.land/crush.json",
  "permissions": {
    "allowed_tools": [
      "view",
      "ls",
      "grep",
      "edit",
      "mcp_context7_get-library-doc"
    ]
  }
}
```

You can also skip all permission prompts entirely by running Lash with the `--yolo` flag (or setting `lash.yolo` in config). Be careful with this feature.

### Timeouts

To prevent requests or tool calls from hanging indefinitely, you can configure global caps under `options`:

```json
{
  "$schema": "https://charm.land/crush.json",
  "options": {
    "request_timeout_seconds": 300,
    "tool_call_timeout_seconds": 120
  }
}
```

- `request_timeout_seconds`: Maximum duration for a single agent request. When reached, the request is canceled.
- `tool_call_timeout_seconds`: Maximum duration for each individual tool call. Tools with their own shorter timeouts still apply; this acts as a safety cap.

Built-in tools like `bash`, `fetch`, `download`, and `sourcegraph` already enforce their own per-call timeouts; the global caps add an extra safeguard.

### Local Models

Local models can also be configured via OpenAI-compatible API. Here are two common examples:

#### Ollama

```json
{
  "providers": {
    "ollama": {
      "name": "Ollama",
      "base_url": "http://localhost:11434/v1/",
      "type": "openai",
      "models": [
        {
          "name": "Qwen 3 30B",
          "id": "qwen3:30b",
          "context_window": 256000,
          "default_max_tokens": 20000
        }
      ]
    }
  }
}
```

#### LM Studio

```json
{
  "providers": {
    "lmstudio": {
      "name": "LM Studio",
      "base_url": "http://localhost:1234/v1/",
      "type": "openai",
      "models": [
        {
          "name": "Qwen 3 30B",
          "id": "qwen/qwen3-30b-a3b-2507",
          "context_window": 256000,
          "default_max_tokens": 20000
        }
      ]
    }
  }
}
```

### Custom Providers

Lash supports custom provider configurations for both OpenAI-compatible and Anthropic-compatible APIs.

#### OpenAI-Compatible APIs

Here’s an example configuration for Deepseek, which uses an OpenAI-compatible
API. Don't forget to set `DEEPSEEK_API_KEY` in your environment.

```json
{
  "$schema": "https://charm.land/crush.json",
  "providers": {
    "deepseek": {
      "type": "openai",
      "base_url": "https://api.deepseek.com/v1",
      "api_key": "$DEEPSEEK_API_KEY",
      "models": [
        {
          "id": "deepseek-chat",
          "name": "Deepseek V3",
          "cost_per_1m_in": 0.27,
          "cost_per_1m_out": 1.1,
          "cost_per_1m_in_cached": 0.07,
          "cost_per_1m_out_cached": 1.1,
          "context_window": 64000,
          "default_max_tokens": 5000
        }
      ]
    }
  }
}
```

#### Anthropic-Compatible APIs

Custom Anthropic-compatible providers follow this format:

```json
{
  "$schema": "https://charm.land/crush.json",
  "providers": {
    "custom-anthropic": {
      "type": "anthropic",
      "base_url": "https://api.anthropic.com/v1",
      "api_key": "$ANTHROPIC_API_KEY",
      "extra_headers": {
        "anthropic-version": "2023-06-01"
      },
      "models": [
        {
          "id": "claude-sonnet-4-20250514",
          "name": "Claude Sonnet 4",
          "cost_per_1m_in": 3,
          "cost_per_1m_out": 15,
          "cost_per_1m_in_cached": 3.75,
          "cost_per_1m_out_cached": 0.3,
          "context_window": 200000,
          "default_max_tokens": 50000,
          "can_reason": true,
          "supports_attachments": true
        }
      ]
    }
  }
}
```

### Amazon Bedrock

Lash supports running Anthropic models through Bedrock, with caching disabled.

* A Bedrock provider will appear once you have AWS configured, i.e. `aws configure`
* Crush also expects the `AWS_REGION` or `AWS_DEFAULT_REGION` to be set
* To use a specific AWS profile set `AWS_PROFILE` in your environment, i.e. `AWS_PROFILE=myprofile crush`

### Vertex AI Platform

Vertex AI will appear in the list of available providers when `VERTEXAI_PROJECT` and `VERTEXAI_LOCATION` are set. You will also need to be authenticated:

```bash
gcloud auth application-default login
```

To add specific models to the configuration, configure as such:

```json
{
  "$schema": "https://charm.land/crush.json",
  "providers": {
    "vertexai": {
      "models": [
        {
          "id": "claude-sonnet-4@20250514",
          "name": "VertexAI Sonnet 4",
          "cost_per_1m_in": 3,
          "cost_per_1m_out": 15,
          "cost_per_1m_in_cached": 3.75,
          "cost_per_1m_out_cached": 0.3,
          "context_window": 200000,
          "default_max_tokens": 50000,
          "can_reason": true,
          "supports_attachments": true
        }
      ]
    }
  }
}
```

### A Note on Claude Max and GitHub Copilot

Lash only supports model providers through official, compliant APIs. We do not
support or endorse any methods that rely on personal Claude Max and GitHub Copilot
accounts or OAuth workarounds, which may violate Anthropic and Microsoft’s
Terms of Service.

We’re committed to building sustainable, trusted integrations with model
providers. If you’re a provider interested in working with us,
[reach out](mailto:vt100@charm.sh).

### Logging

Logs are stored in `./.lash/logs/lash.log` relative to your project.

The CLI also contains some helper commands to make perusing recent logs easier:

```bash
# Print the last 1000 lines
lash logs

# Print the last 500 lines
lash logs --tail 500

# Follow logs in real time
lash logs --follow
```

Want more logging? Run `lash` with the `--debug` flag, or enable it in the
config:

```json
{
  "$schema": "https://charm.land/crush.json",
  "options": {
    "debug": true,
    "debug_lsp": true
  }
}
```

### Lash-specific Configuration

Lash adds an optional `lash` namespace to configuration for mode and safety controls while remaining compatible with upstream `crush.json`:

```json
{
  "$schema": "https://charm.land/crush.json",
  "lash": {
    "mode": "Auto",
    "yolo": false,
    "safety": { "confirm_agent_exec": true }
  }
}
```

### License

FSL-1.1-MIT (MIT Future). See `LICENSE`.
