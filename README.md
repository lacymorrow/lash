<p align="center">
  <a href="https://opencode.ai">
    <picture>
      <source srcset="packages/console/app/src/asset/logo-ornate-dark.svg" media="(prefers-color-scheme: dark)">
      <source srcset="packages/console/app/src/asset/logo-ornate-light.svg" media="(prefers-color-scheme: light)">
      <img src="packages/console/app/src/asset/logo-ornate-light.svg" alt="lash logo">
    </picture>
  </a>
</p>
<p align="center">AI coding agent, built for the terminal.</p>
<p align="center">
  <a href="https://opencode.ai/discord"><img alt="Discord" src="https://img.shields.io/discord/1391832426048651334?style=flat-square&label=discord" /></a>
  <a href="https://www.npmjs.com/package/opencode-ai"><img alt="npm" src="https://img.shields.io/npm/v/opencode-ai?style=flat-square" /></a>
  <a href="https://github.com/sst/opencode/actions/workflows/publish.yml"><img alt="Build status" src="https://img.shields.io/github/actions/workflow/status/sst/opencode/publish.yml?style=flat-square&branch=dev" /></a>
</p>

[![Terminal UI](packages/web/src/assets/lander/screenshot.png)](https://opencode.ai)

AI-powered terminal assistant for software developers.

## Why This Fork?

This is a maintained fork of [anomalyco/opencode](https://github.com/anomalyco/opencode). We forked to extend OpenCode with shell-native features we need for our own workflows that haven't landed upstream yet:

- **Shell mode** — a first-class execution mode (`ctrl+space` or `!` prefix) that runs commands in a live shell instead of routing through the AI, with a distinct color bar in the TUI prompt
- **Working directory tracking** — `getCwd()` / `setCwd()` persists the shell cwd across commands; a sentinel in bash output propagates `cd` changes back to the UI footer
- **Natural language detection** — automatically routes input to AI when it reads like a question or instruction, to shell when it reads like a command — no prefix needed in `auto` mode
- **Double left border UI** — two side bars in the TUI prompt: outer bar = agent color, inner bar = execution mode color, using a row layout that keeps both bars at equal height
- **Tab → shell autocomplete** — Tab triggers shell completion; Shift+Tab cycles agents (upstream maps Tab to agent cycle)
- **`EventCwdUpdated` SDK type** — exposes working directory change events through the JS SDK
- **Independent versioning** — lash uses `1.7.x` versioning and is published independently of upstream releases

We keep the fork in sync with upstream (`git fetch upstream && git merge upstream/dev`) and document merge conflicts in [UPSTREAM_SYNC.md](./UPSTREAM_SYNC.md).

## Installation

### One-line install (Recommended)
```bash
curl -fsSL https://raw.githubusercontent.com/lacymorrow/lash/main/install | bash
```

### Homebrew
```bash
brew install lacymorrow/tap/lash
```

### Documentation

For more info on how to configure OpenCode [**head over to our docs**](https://opencode.ai/docs).

### Contributing

OpenCode is an opinionated tool so any fundamental feature needs to go through a
design process with the core team.

> [!IMPORTANT]
> We do not accept PRs for core features.

However we still merge a ton of PRs - you can contribute:

- Bug fixes
- Improvements to LLM performance
- Support for new providers
- Fixes for env specific quirks
- Missing standard behavior
- Documentation

Take a look at the git history to see what kind of PRs we end up merging.

> [!NOTE]
> If you do not follow the above guidelines we might close your PR.

To run OpenCode locally you need.

- Bun 1.3 or higher
- Golang 1.24.x

And run.
```bash
npm install -g lash-cli
```

### From Source
```bash
git clone https://github.com/lacymorrow/lash
cd lash/packages/opencode
bun install
bun run dev
```

**API Client**: After making changes to the TypeScript API endpoints in `packages/opencode/src/server/server.ts`, you will need the OpenCode team to generate a new stainless sdk for the clients.

## Usage

```bash
lash --help
```

## Features

- 100% open source
- Not coupled to any provider. Although Anthropic is recommended, OpenCode can be used with OpenAI, Google or even local models. As models evolve the gaps between them will close and pricing will drop so being provider-agnostic is important.
- Out of the box LSP support
- A focus on TUI. OpenCode is built by neovim users and the creators of [terminal.shop](https://terminal.shop); we are going to push the limits of what's possible in the terminal.
- A client/server architecture. This for example can allow OpenCode to run on your computer, while you can drive it remotely from a mobile app. Meaning that the TUI frontend is just one of the possible clients.

## License

MIT

## License

The other confusingly named repo has no relation to this one. You can [read the story behind it here](https://x.com/thdxr/status/1933561254481666466).

---

**Join our community** [Discord](https://discord.gg/opencode) | [X.com](https://x.com/opencode)
