# Lash Release Engineering

Lash uses a specialized release pipeline designed to exist alongside the upstream OpenCode project without modifying upstream source files. This ensures easier merges and cleaner separation of concerns.

## How to cut a release

**From GitHub (preferred).** Actions → **release lash** → Run workflow. Leave the version blank for a
patch bump, or type one to ship a minor or a major. Nothing to type while it runs. The workflow is
`.github/workflows/release-lash.yml`, it is gated to `lacymorrow/lash`, and it always builds the
`lash` branch.

Tick **dry_run** first if you want to confirm the npm credential and the next version number without
publishing anything. That takes about two minutes; a real release takes closer to twenty.

**From a laptop.** On the `lash` branch:

```bash
bun run release
```

With `NPM_TOKEN` exported this runs unattended. Without it, the script prompts for an npm OTP, which
is the fallback for publishing from an account with 2FA and no automation token.

Either way the run produces, in this order: a version bump commit and a `v<version>` tag, a GitHub
release carrying all eleven archives, the npm publish, and an updated `lash.rb` in
`lacymorrow/homebrew-tap`. A rerun after a partial failure picks up where it stopped rather than
duplicating work.

## Release Scripts

The release process is driven by **Lash-specific scripts** that exist in parallel to the upstream scripts.

### 1. `script/release-lash.ts`

**The Entry Point.** It refuses to run outside the `lash` branch, picks the version, checks the npm
credential before spending time on a build, then drives the steps listed above.

Version selection, when no version is given: patch-bump whichever of npm and the newest GitHub
release is further ahead, then skip past any version whose tag already exists. The logic lives in
`packages/opencode/script/release-version.ts` and is covered by
`packages/opencode/test/release-version.test.ts`.

### 2. `packages/opencode/script/publish-lash.ts`

**The Build & Asset Transformer.**
This script is invoked by `release-lash.ts`. It:

1.  Imports `build-lash.ts` to build `lash` binaries directly.
2.  **Patches** the wrapper script (`bin/opencode` -> `bin/lash`) to use Lash branding and paths.
3.  **Packages** `lashcode` for NPM.
4.  **Updates** `lacymorrow/homebrew-tap` with a new `lash.rb` formula.

### 3. `packages/opencode/script/build-lash.ts`

**Lash-Specific Builder.**
A dedicated build script that outputs `lash` binaries directly, removing the need for renaming artifacts post-build.

### 3. Helper Scripts

- **`script/changelog-lash.ts`**: Fetches commits/releases from `lacymorrow/lash` instead of upstream.
- **`packages/opencode/script/postinstall-lash.mjs`**: A modified postinstall script that creates the correct symlinks for `lash` binaries.

## Credentials

| Name | Where it lives | What it is for |
| --- | --- | --- |
| `NPM_TOKEN` | repo secret on `lacymorrow/lash` | Publishing `lashcode`. Must be a **granular automation token** scoped to the package with read and write permission. Automation tokens bypass 2FA, which is what lets the workflow publish without a human typing an OTP. Classic tokens on a 2FA account will not work. |
| `GITHUB_TOKEN` | provided by Actions | Pushing the tag and creating the release. |
| `HOMEBREW_TAP_TOKEN` | repo secret on `lacymorrow/lash` | Pushing `lash.rb` to `lacymorrow/homebrew-tap`. That push is cross-repo, so the token Actions provides is not enough. |

A dead or missing `NPM_TOKEN` fails the run in the first couple of minutes with the exact command
needed to replace it, rather than after a full build.

Local runs also need **Bun** (v1.3.x+) and a `gh` login with access to `lacymorrow/lash` and
`lacymorrow/homebrew-tap`. An **AUR key** is optional and currently unused.

## Directory Structure

Items marked with `(*)` are Lash-specific additions.

```text
root/
├── .github/workflows/
│   └── release-lash.yml (*)     # workflow_dispatch entry point
├── script/
│   ├── release-lash.ts (*)      # MAIN ENTRY POINT
│   ├── changelog-lash.ts (*)    # Lash history logic
│   └── ...
└── packages/
    └── opencode/
        └── script/
            ├── publish-lash.ts (*)      # Build/Rename/Publish logic
            ├── release-version.ts (*)   # Which version to ship next
            ├── npm-auth.ts (*)          # NPM_TOKEN handling, OTP fallback
            ├── postinstall-lash.mjs (*) # Runtime binary finder
            ├── build.ts                 # Upstream builder (used by us)
            └── publish.ts               # Upstream publisher (ignored)
```
