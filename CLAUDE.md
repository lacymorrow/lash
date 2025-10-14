# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **lash** - a fork of sst/opencode, an AI-powered terminal assistant for software developers. It provides a CLI tool for AI-assisted coding with support for multiple AI providers.

## Key Commands

### Development
```bash
# Run development server (from workspace root)
bun run dev

# Run development server from opencode package
cd packages/opencode && bun run dev
```

### Testing
```bash
# Run all tests
bun test

# Run specific test file
bun test packages/opencode/test/tool/edit.test.ts
```

### Type Checking
```bash
# Type check all packages
bun run typecheck

# Type check specific package
cd packages/opencode && bun run typecheck
```

### Building
```bash
# Build production binaries
cd packages/opencode && bun run build

# Build development binary with TUI
cd packages/opencode && bun run build:dev

# Build and publish to npm
cd packages/opencode && bun run publish:lash
```

## Architecture

### Repository Structure
- **Monorepo** using Bun workspaces
- **Main CLI**: `packages/opencode/` - TypeScript-based CLI implementation
- **TUI Component**: `packages/tui/` - Go-based terminal UI (embedded into main binary)
- **Infrastructure**: `infra/` - SST configuration for cloud deployment
- **Cloud Functions**: `cloud/` - API and authentication services

### Core Components

#### packages/opencode/src/
- `tool/` - File operations, bash execution, and other development tools
- `session/` - User interaction and session management
- `provider/` - AI provider integrations (Anthropic, OpenAI, Google, etc.)
- `agent/` - Agent system for specialized tasks
- `mcp/` - Model Context Protocol integration
- `lsp/` - Language Server Protocol support
- `shell/` - Shell command execution and management
- `server/` - Local development server
- `cli/` - Command-line interface implementation

### Build Process
1. The Go TUI is compiled from `packages/tui/`
2. The TUI binary is embedded into the main TypeScript application
3. Final binary is created using Bun's compile feature with the embedded TUI

### AI Provider Integration
- Uses the `ai` SDK for provider abstraction
- Supports multiple providers through configuration
- Provider implementations in `packages/opencode/src/provider/`

### Tool System
The tool system (`packages/opencode/src/tool/`) includes:
- File reading/writing/editing
- Bash command execution
- Project file operations
- Pattern matching and search capabilities
- Screenshot and image handling

### Testing Strategy
- Test files located alongside source or in `test/` directories
- Uses Bun's built-in test runner
- Test categories: tool tests, shell tests, session tests, integration tests

## Important Considerations

1. **Binary Building**: The TUI must be built before the main application as it gets embedded
2. **Provider Configuration**: AI providers are configured through environment variables and config files
3. **Development Mode**: Use `--conditions=development` flag for development builds
4. **SST Integration**: Uses SST for infrastructure and deployment configuration