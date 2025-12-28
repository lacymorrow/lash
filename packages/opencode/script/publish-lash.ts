#!/usr/bin/env bun
import { $ } from "bun"
import { createInterface } from "readline"
import pkg from "../package.json"

const version = process.env["LASH_VERSION"] || process.env["OPENCODE_VERSION"] || pkg.version
const dry = process.env["DRY_RUN"] === "true"

const prompt = (query: string) => new Promise<string>((resolve) => {
  const rl = createInterface({
    input: process.stdin,
    output: process.stdout,
  })
  rl.question(query, (answer) => {
    rl.close()
    resolve(answer.trim())
  })
})


console.log(`🚀 Publishing lash v${version}`)
if (dry) console.log("(DRY RUN - no actual publishing)")

const GOARCH: Record<string, string> = {
  arm64: "arm64",
  x64: "amd64",
  "x64-baseline": "amd64",
}

const targets = [
  ["windows", "x64"],
  ["linux", "arm64"],
  ["linux", "x64"],
  ["linux", "x64-baseline"],
  ["darwin", "x64"],
  ["darwin", "x64-baseline"],
  ["darwin", "arm64"],
]

// Resolve packages/opencode relative to script location
const opencodeDir = new URL("..", import.meta.url).pathname

// Clean dist directory relative to opencodeDir
await $`rm -rf dist/lash-*`.cwd(opencodeDir).nothrow()

const optionalDependencies: Record<string, string> = {}

// Build for each target platform
for (const [os, arch] of targets) {
  const name = `lash-cli-${os}-${arch}`
  console.log(`📦 Building ${name}`)

  await $`mkdir -p dist/${name}/bin`.cwd(opencodeDir)

  // Build TUI component
  // Resolve packages/tui relative to packages/opencode (../tui)
  const tuiDir = new URL("../../tui", import.meta.url).pathname
  await $`CGO_ENABLED=0 GOOS=${os} GOARCH=${GOARCH[arch]} go build -ldflags="-s -w -X main.Version=${version}" -o ../opencode/dist/${name}/bin/tui ./cmd/opencode/main.go`.cwd(
    tuiDir,
  )

  // Build lash binary with embedded TUI
  // Dynamic externals logic:
  // We must externalize packages we don't have (to avoid build errors).
  // We should bundle packages we DO have (to make the binary work).
  // We typically only have the package for the current host platform.
  // So: Externalize ALL platform packages EXCEPT the one for the current host (if target matches host).

  // List of all platform components we know of
  const opentuiPlatforms = [
    "@opentui/core-win32-x64",
    "@opentui/core-linux-x64",
    "@opentui/core-linux-arm64",
    "@opentui/core-darwin-x64",
    "@opentui/core-darwin-arm64",
  ]
  const watcherPlatforms = [
    "@parcel/watcher-win32-x64",
    "@parcel/watcher-linux-x64-glibc",
    "@parcel/watcher-linux-arm64-glibc",
    "@parcel/watcher-darwin-x64",
    "@parcel/watcher-darwin-arm64",
  ]
  const humblePlatforms = [
    "@humble/core-darwin-arm64",
    "@humble/core-darwin-x64",
    "@humble/core-linux-arm64",
    "@humble/core-linux-x64",
    "@humble/core-win32-x64",
  ]

  // Helper to check if a package matches current host
  // "darwin-arm64" matching "@opentui/core-darwin-arm64"
  // Note: watcher uses glibc suffix for linux.
  const hostStr = `${process.platform}-${process.arch}` // e.g. darwin-arm64
  const isHost = (pkg: string) => pkg.includes(hostStr)

  const externals = [
    ...opentuiPlatforms,
    ...watcherPlatforms,
    ...humblePlatforms
  ].filter(pkg => {
    // If we are building for the host platform, we DON'T want to externalize the host package (we want to bundle it).
    // If we are building for another platform, we externalize everything (since we don't have deps).
    // Actually, even if building for another platform, externalizing host pkg is fine (it won't be used).
    // So distinct logic: 
    // KEEP (bundle) the package IF it matches valid installed deps.
    // We only have installed deps for `hostStr`.
    // So: If package matches `hostStr`, KEEP it (don't externalize).
    // logic: filter returns true to KEEP in externals listing (i.e. exclude from bundle).
    // So return true if NOT host.
    return !pkg.includes(process.platform === 'win32' ? 'win32' : process.platform) || !pkg.includes(process.arch)
  }).flatMap(pkg => ["--external", pkg])

  // Note: The logic above `!pkg.includes(...)` is simplistic.
  // Better: identify specifically the installed package.
  // But strict exclusion of the ONE matching package is safer.
  // Let's rely on exact match if possible, but names vary.
  // Simple heuristic: If `pkg` contains both platform and arch of HOST, keep it (bundle it). Else externalize.
  // BUT watcher linux has glibc. `linux-x64` vs `linux-x64-glibc`.
  // `process.platform` linux, `process.arch` x64.
  // If host is linux-x64, we have `watcher-linux-x64-glibc`.
  // My heuristic: `pkg.includes('linux') && pkg.includes('x64')` -> true.
  // So we bundle it. Good.

  await $`bun build --define OPENCODE_TUI_PATH="'../../../dist/${name}/bin/tui'" --define OPENCODE_VERSION="'${version}'" --compile --target=bun-${os}-${arch} --outfile=dist/${name}/bin/lash ./src/index.ts --embed ./dist/${name}/bin/tui ${externals}`.cwd(opencodeDir)

  // Run smoke test on current platform
  if (
    process.platform === (os === "windows" ? "win32" : os) &&
    (process.arch === arch || (process.arch === "x64" && arch === "x64-baseline"))
  ) {
    console.log(`✓ Smoke test: running dist/${name}/bin/lash --version`)
    await $`./dist/${name}/bin/lash --version`.cwd(opencodeDir)
    console.log(`✓ Smoke test: checking TUI binary is embedded`)
    await $`./dist/${name}/bin/lash --help | head -5`.cwd(opencodeDir)
  }

  // Clean up TUI binary (now embedded in lash binary)
  await $`rm -rf ./dist/${name}/bin/tui`.cwd(opencodeDir)

  // Create package.json for platform-specific package
  // Note: Bun.file(path) needs absolute path or relative to process.cwd(). Since opencodeDir is absolute, we should construct path there.
  // However, Bun.file() doesn't chain like $. 
  // We will assume Bun.file uses process.cwd(). We need to be careful.
  // The original script was writing to `dist/${name}/package.json` relative to CWD.
  // If we want consistency, we should use path.join(opencodeDir, ...) or assume running from repo root?
  // Currently the build puts files in `packages/opencode/dist` (verified).
  // But mkdir (line 50 now fixed) puts them in `packages/opencode/dist`.
  // So Bun.file should write to `packages/opencode/dist/...`.
  // If run from root, `dist` is root/dist. That's WRONG.
  // The `mkdir` correction handles the directory creation.
  // We must fix `Bun.file` paths to be absolute using `opencodeDir`.

  const pkgPath = new URL(`../dist/${name}/package.json`, import.meta.url).pathname
  await Bun.file(pkgPath).write(
    JSON.stringify(
      {
        name,
        version,
        os: [os === "windows" ? "win32" : os],
        cpu: [arch],
        bin: {
          lash: os === "windows" ? "./bin/lash.exe" : "./bin/lash"
        }
      },
      null,
      2,
    ),
  )

  // Publish platform-specific package
  if (!dry) {
    const otp = await prompt(`Enter NPM OTP for ${name} (leave empty to skip): `)
    if (otp) {
      await $`cd dist/${name} && chmod -R 755 . && npm publish --access public --otp=${otp}`.cwd(opencodeDir)
    } else {
      console.log(`Skipping publish for ${name} (no OTP provided)`)
    }
  }

  optionalDependencies[name] = version
}

// Create main lash-cli package
console.log("📦 Creating main lash-cli package")
await $`mkdir -p ./dist/lash-cli/bin`.cwd(opencodeDir)

// Create wrapper script for Unix systems
const unixWrapper = `#!/bin/sh
set -e

if [ -n "$LASH_BIN_PATH" ]; then
    resolved="$LASH_BIN_PATH"
else
    # Get the real path of this script, resolving any symlinks
    script_path="$0"
    while [ -L "$script_path" ]; do
        link_target="$(readlink "$script_path")"
        case "$link_target" in
            /*) script_path="$link_target" ;;
            *) script_path="$(dirname "$script_path")/$link_target" ;;
        esac
    done
    script_dir="$(dirname "$script_path")"
    script_dir="$(cd "$script_dir" && pwd)"
    
    # Map platform names
    case "$(uname -s)" in
        Darwin) platform="darwin" ;;
        Linux) platform="linux" ;;
        MINGW*|CYGWIN*|MSYS*) platform="win32" ;;
        *) platform="$(uname -s | tr '[:upper:]' '[:lower:]')" ;;
    esac
    
    # Map architecture names  
    case "$(uname -m)" in
        x86_64|amd64) arch="x64" ;;
        aarch64) arch="arm64" ;;
        armv7l) arch="arm" ;;
        *) arch="$(uname -m)" ;;
    esac
    
    name="lash-cli-\${platform}-\${arch}"
    binary="lash"
    [ "$platform" = "win32" ] && binary="lash.exe"
    
    # Search for the binary starting from real script location
    resolved=""
    current_dir="$script_dir"
    while [ "$current_dir" != "/" ]; do
        candidate="$current_dir/node_modules/$name/bin/$binary"
        if [ -f "$candidate" ]; then
            resolved="$candidate"
            break
        fi
        current_dir="$(dirname "$current_dir")"
    done
    
    if [ -z "$resolved" ]; then
        printf "It seems that your package manager failed to install the right version of the lash CLI for your platform. You can try manually installing the \\"%s\\" package\\n" "$name" >&2
        exit 1
    fi
fi

# Handle SIGINT gracefully
trap '' INT

# Execute the binary with all arguments
exec "$resolved" "$@"
`

await Bun.file(new URL("../dist/lash-cli/bin/lash", import.meta.url)).write(unixWrapper)
await $`chmod +x ./dist/lash-cli/bin/lash`.cwd(opencodeDir)

// Create Windows wrapper
const windowsWrapper = `@IF EXIST "%~dp0\\node.exe" (
  "%~dp0\\node.exe" "%~dp0\\..\\node_modules\\lash-cli-windows-x64\\bin\\lash.exe" %*
) ELSE (
  @SETLOCAL
  @SET PATHEXT=%PATHEXT:;.JS;=;%
  node "%~dp0\\..\\node_modules\\lash-cli-windows-x64\\bin\\lash.exe" %*
)`

await Bun.file(new URL("../dist/lash-cli/bin/lash.cmd", import.meta.url)).write(windowsWrapper)

// Create postinstall script
const postinstallScript = `#!/usr/bin/env node
import fs from "fs"
import path from "path"
import os from "os"
import { fileURLToPath } from "url"
import { createRequire } from "module"

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const require = createRequire(import.meta.url)

function detectPlatformAndArch() {
  // Map platform names
  let platform
  switch (os.platform()) {
    case "darwin":
      platform = "darwin"
      break
    case "linux":
      platform = "linux"
      break
    case "win32":
      platform = "windows"
      break
    default:
      platform = os.platform()
      break
  }

  // Map architecture names
  let arch
  switch (os.arch()) {
    case "x64":
      arch = "x64"
      break
    case "arm64":
      arch = "arm64"
      break
    case "arm":
      arch = "arm"
      break
    default:
      arch = os.arch()
      break
  }

  return { platform, arch }
}

function findBinary() {
  const { platform, arch } = detectPlatformAndArch()
  const packageName = \`lash-cli-\${platform}-\${arch}\`
  const binary = platform === "windows" ? "lash.exe" : "lash"

  try {
    // Use require.resolve to find the package
    const packageJsonPath = require.resolve(\`\${packageName}/package.json\`)
    const packageDir = path.dirname(packageJsonPath)
    const binaryPath = path.join(packageDir, "bin", binary)

    if (!fs.existsSync(binaryPath)) {
      throw new Error(\`Binary not found at \${binaryPath}\`)
    }

    return binaryPath
  } catch (error) {
    throw new Error(\`Could not find package \${packageName}: \${error.message}\`)
  }
}

function main() {
  try {
    const binaryPath = findBinary()
    const binScript = path.join(__dirname, "bin", "lash")

    // Remove existing bin script if it exists
    if (fs.existsSync(binScript)) {
      fs.unlinkSync(binScript)
    }

    // Create symlink to the actual binary
    fs.symlinkSync(binaryPath, binScript, 'file')
    fs.chmodSync(binScript, '755')
    console.log(\`lash binary symlinked: \${binScript} -> \${binaryPath}\`)
  } catch (error) {
    console.error("Failed to create lash binary symlink:", error.message)
    process.exit(1)
  }
}

main()
`

await Bun.file(new URL("../dist/lash-cli/postinstall.mjs", import.meta.url)).write(postinstallScript)

// Create main package.json
await Bun.file(new URL("../dist/lash-cli/package.json", import.meta.url)).write(
  JSON.stringify(
    {
      name: "lash-cli",
      version,
      description: "lash - The AI coding agent built for the terminal",
      bin: {
        lash: "./bin/lash"
      },
      scripts: {
        postinstall: "node ./postinstall.mjs"
      },
      keywords: ["ai", "cli", "coding", "assistant", "terminal", "lash"],
      author: "",
      license: "MIT",
      repository: {
        type: "git",
        url: "https://github.com/lacymorrow/lash"
      },
      optionalDependencies,
      engines: {
        node: ">=18.0.0"
      }
    },
    null,
    2,
  ),
)

// Publish main package
if (!dry) {
  const otp = await prompt(`Enter NPM OTP for lash-cli (leave empty to skip): `)
  if (otp) {
    await $`cd ./dist/lash-cli && npm publish --access public --otp=${otp}`.cwd(opencodeDir)
  } else {
    console.log(`Skipping publish for lash-cli (no OTP provided)`)
  }
}

console.log(`✨ lash v${version} published successfully!`)

// Create zip files for GitHub releases and compute checksums
if (!dry) {
  console.log("📦 Creating zip files for GitHub release")
  for (const [os, arch] of targets) {
    const name = `lash-cli-${os}-${arch}`
    await $`cd dist/${name}/bin && zip -r ../../${name}.zip *`.cwd(opencodeDir)
  }

  // Compute SHA256 checksums (use shasum for macOS compatibility)
  const shaOf = async (path: string) =>
    await $`shasum -a 256 ${path} | cut -d' ' -f1`.cwd(opencodeDir).text().then((x) => x.trim())

  const linuxArm64Zip = new URL("../dist/lash-cli-linux-arm64.zip", import.meta.url).pathname
  const linuxX64Zip = new URL("../dist/lash-cli-linux-x64.zip", import.meta.url).pathname
  const darwinX64Zip = new URL("../dist/lash-cli-darwin-x64.zip", import.meta.url).pathname
  const darwinArm64Zip = new URL("../dist/lash-cli-darwin-arm64.zip", import.meta.url).pathname

  const arm64Sha = (await Bun.file(linuxArm64Zip).exists()) ? await shaOf('./dist/lash-cli-linux-arm64.zip') : ""
  const x64Sha = (await Bun.file(linuxX64Zip).exists()) ? await shaOf('./dist/lash-cli-linux-x64.zip') : ""
  const macX64Sha = (await Bun.file(darwinX64Zip).exists()) ? await shaOf('./dist/lash-cli-darwin-x64.zip') : ""
  const macArm64Sha = (await Bun.file(darwinArm64Zip).exists()) ? await shaOf('./dist/lash-cli-darwin-arm64.zip') : ""

  console.log("\n📋 SHA256 Checksums:")
  if (arm64Sha) console.log(`lash-cli-linux-arm64: ${arm64Sha}`)
  if (x64Sha) console.log(`lash-cli-linux-x64: ${x64Sha}`)
  if (macX64Sha) console.log(`lash-cli-darwin-x64: ${macX64Sha}`)
  if (macArm64Sha) console.log(`lash-cli-darwin-arm64: ${macArm64Sha}`)

  // Create Homebrew formula and push to tap if GITHUB_TOKEN is available
  const tapToken = process.env["GITHUB_TOKEN"]
  if (tapToken) {
    const shortVersion = version.split("-")[0]
    const homebrewFormula = [
      "# typed: false",
      "# frozen_string_literal: true",
      "",
      "# Generated by publish-lash.ts",
      "class Lash < Formula",
      `  desc "The AI coding agent built for the terminal."`,
      `  homepage "https://github.com/lacymorrow/lash"`,
      `  version "${shortVersion}"`,
      "",
      "  on_macos do",
      "    if Hardware::CPU.intel?",
      `      url "https://github.com/lacymorrow/lash/releases/download/v${version}/lash-cli-darwin-x64.zip"`,
      macX64Sha ? `      sha256 "${macX64Sha}"` : "      sha256 :no_check",
      "",
      "      def install",
      '        bin.install "lash"',
      "      end",
      "    end",
      "    if Hardware::CPU.arm?",
      `      url "https://github.com/lacymorrow/lash/releases/download/v${version}/lash-cli-darwin-arm64.zip"`,
      macArm64Sha ? `      sha256 "${macArm64Sha}"` : "      sha256 :no_check",
      "",
      "      def install",
      '        bin.install "lash"',
      "      end",
      "    end",
      "  end",
      "",
      "  on_linux do",
      "    if Hardware::CPU.intel? and Hardware::CPU.is_64_bit?",
      `      url "https://github.com/lacymorrow/lash/releases/download/v${version}/lash-cli-linux-x64.zip"`,
      x64Sha ? `      sha256 "${x64Sha}"` : "      sha256 :no_check",
      "      def install",
      '        bin.install "lash"',
      "      end",
      "    end",
      "    if Hardware::CPU.arm? and Hardware::CPU.is_64_bit?",
      `      url "https://github.com/lacymorrow/lash/releases/download/v${version}/lash-cli-linux-arm64.zip"`,
      arm64Sha ? `      sha256 "${arm64Sha}"` : "      sha256 :no_check",
      "      def install",
      '        bin.install "lash"',
      "      end",
      "    end",
      "  end",
      "end",
      "",
    ].join("\n")

    await $`rm -rf ./dist/homebrew-tap`.cwd(opencodeDir)
    await $`git clone https://${tapToken}@github.com/lacymorrow/homebrew-tap.git ./dist/homebrew-tap`.cwd(opencodeDir)
    await Bun.file(new URL("../dist/homebrew-tap/Formula/lash.rb", import.meta.url)).write(homebrewFormula)
    await $`cd ./dist/homebrew-tap && git add Formula/lash.rb`.cwd(opencodeDir)
    await $`cd ./dist/homebrew-tap && git commit -m "Update lash to v${version}"`.cwd(opencodeDir)
    await $`cd ./dist/homebrew-tap && git push`.cwd(opencodeDir)
  }

  // Create GitHub release and upload assets if GITHUB_TOKEN is available
  const ghToken = process.env["GITHUB_TOKEN"]
  if (ghToken) {
    const owner = "lacymorrow"
    const repo = "lash"
    const res = await fetch(`https://api.github.com/repos/${owner}/${repo}/releases`, {
      method: "POST",
      headers: {
        Authorization: `token ${ghToken}`,
        "Content-Type": "application/json",
        Accept: "application/vnd.github+json",
      },
      body: JSON.stringify({
        tag_name: `v${version}`,
        name: `v${version}`,
        draft: false,
        prerelease: version.includes("-")
      })
    })

    if (res.ok) {
      const json = await res.json()
      const releaseId = json.id as number
      const upload = async (filePath: string, name: string) => {
        if (!await Bun.file(filePath).exists()) return
        const buf = await Bun.file(filePath).arrayBuffer()
        const uploadUrl = `https://uploads.github.com/repos/${owner}/${repo}/releases/${releaseId}/assets?name=${encodeURIComponent(name)}`
        await fetch(uploadUrl, {
          method: "POST",
          headers: {
            Authorization: `token ${ghToken}`,
            "Content-Type": "application/zip",
            Accept: "application/vnd.github+json",
          },
          body: buf,
        })
      }

      await upload(linuxArm64Zip, "lash-cli-linux-arm64.zip")
      await upload(linuxX64Zip, "lash-cli-linux-x64.zip")
      await upload(darwinX64Zip, "lash-cli-darwin-x64.zip")
      await upload(darwinArm64Zip, "lash-cli-darwin-arm64.zip")
    }
  }
}