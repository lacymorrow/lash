#!/usr/bin/env bun

import { $ } from "bun"
import { appendFile } from "fs/promises"
import path from "path"
import { buildNotes, getLatestRelease } from "./changelog-lash"
import { verifyNpmAuth } from "../packages/opencode/script/npm-auth"
import { nextVersion } from "../packages/opencode/script/release-version"

const REPO = "lacymorrow/lash"

/** Actions collects step outputs by appending to this file, so never truncate it. */
async function writeOutput(output: string) {
  const file = process.env["GITHUB_OUTPUT"]
  if (!file) return
  await appendFile(file, output)
}

/** `OPENCODE_VERSION` wins so a caller can ship a minor or a major; otherwise patch bump. */
async function resolveVersion() {
  const override = process.env["OPENCODE_VERSION"]
  if (override) return override

  const [published, released] = await Promise.all([
    fetch("https://registry.npmjs.org/lashcode/latest")
      .then((res) => (res.ok ? res.json() : undefined))
      .then((data: any) => data?.version as string | undefined)
      .catch(() => undefined),
    getLatestRelease().catch(() => undefined),
  ])
  console.log(`last release: npm ${published ?? "unknown"}, github ${released ?? "unknown"}`)

  const tags = await $`git tag --list`.text().then((text) => text.split("\n").map((line) => line.trim()))
  return nextVersion({ published, released, tags })
}

// Releases are cut from lash and nowhere else. Stating that once here is simpler than reasoning
// about preview channels, and it stops a stray branch from publishing to npm by accident.
const branch = await $`git branch --show-current`.text().then((t) => t.trim())
if (branch !== "lash") {
  throw new Error(`Releases are cut from the lash branch. You are on ${branch || "a detached HEAD"}.`)
}

const version = await resolveVersion()
// Script reads both of these at import time, so they have to be set before it loads.
process.env["OPENCODE_VERSION"] = version
process.env["OPENCODE_CHANNEL"] = "latest"

const { Script } = await import("@opencode-ai/script")

console.log(`=== publishing lash v${Script.version} ===\n`)

// Check the npm credential before a build that takes a quarter of an hour.
await verifyNpmAuth()

// A dry run answers "is the token good and what would ship" without touching npm, the tags or
// the releases. Actions passes booleans through as the strings "true" and "false".
if (process.env["LASH_RELEASE_DRY_RUN"] === "true") {
  console.log(`dry run: would release v${Script.version} from ${branch}. Nothing was published.`)
  await writeOutput(`version=${Script.version}\n`)
  process.exit(0)
}

const previous = await getLatestRelease()
const notes = await buildNotes(previous, "HEAD")

const pkgjsons = await Array.fromAsync(
  new Bun.Glob("**/package.json").scan({
    absolute: true,
  }),
).then((arr) => arr.filter((x) => !x.includes("node_modules") && !x.includes("dist")))

// Version bumping logic - keeping strict sync with Script.version
for (const file of pkgjsons) {
  let pkg = await Bun.file(file).text()
  pkg = pkg.replaceAll(/"version": "[^"]+"/g, `"version": "${Script.version}"`)
  console.log("updated:", file)
  await Bun.file(file).write(pkg)
}

// Update Extension toml if needed (optional for Lash, but safe)
try {
  const extensionToml = new URL("../packages/extensions/zed/extension.toml", import.meta.url).pathname
  let toml = await Bun.file(extensionToml).text()
  toml = toml.replace(/^version = "[^"]+"/m, `version = "${Script.version}"`)
  toml = toml.replaceAll(/releases\/download\/v[^/]+\//g, `releases/download/v${Script.version}/`)
  console.log("updated:", extensionToml)
  await Bun.file(extensionToml).write(toml)
} catch (e) {
  // ignore if missing
}

await $`bun install`

console.log("\n=== opencode (lash) ===\n")
// Call our new lash publish script, but DEFER the registry/npm steps until we have released on GitHub
process.env.DEFER_PUBLISH_TASKS = "true"
const publishModule = await import(`../packages/opencode/script/publish-lash.ts`)

const dir = new URL("..", import.meta.url).pathname
process.chdir(dir)

const tag = `v${Script.version}`
let output = `version=${Script.version}\n`

// Commit and Tag (tolerate "nothing to commit" if versions were already bumped)
await $`git commit -am "release: ${tag}"`.nothrow()

// A tag left over from a rerun is fine as long as it names this commit, otherwise the release
// would ship an older tree than the one we just built.
const head = await $`git rev-parse HEAD`.text().then((t) => t.trim())
const tagged = await $`git rev-list -n 1 ${tag}`.nothrow().quiet()
if (tagged.exitCode !== 0) {
  await $`git tag ${tag}`
} else if (tagged.stdout.toString().trim() !== head) {
  throw new Error(`Tag ${tag} already points at another commit. Pass OPENCODE_VERSION to pick a free version.`)
}

await $`git fetch origin`
await $`git push origin HEAD --tags --no-verify --force-with-lease`

// Validating presence of artifacts before creating release
console.log("Verifying artifacts for release...")
const releaseFiles = await Array.fromAsync(new Bun.Glob("./packages/opencode/dist/*.{zip,tar.gz}").scan())
if (releaseFiles.length === 0) {
  throw new Error("No distribution archives found in packages/opencode/dist, refusing to cut an empty release")
}
console.log(`Found ${releaseFiles.length} archives to upload`)

// Create the release, or fill in one that a previous run left half finished
const existing = await $`gh release view ${tag} --repo ${REPO}`.nothrow().quiet()
if (existing.exitCode === 0) {
  console.log(`Release ${tag} already exists, uploading archives...`)
  await $`gh release upload ${tag} --repo ${REPO} --clobber ${releaseFiles}`
} else {
  await $`gh release create ${tag} --repo ${REPO} --title ${tag} --notes ${notes.join("\n") || "No notable changes"} ${releaseFiles}`
}

const release = await $`gh release view ${tag} --repo ${REPO} --json id,tagName`.json()

output += `release=${release.id}\n`
output += `tag=${release.tagName}\n`
await writeOutput(output)

console.log("GitHub Release verified. Publishing to npm...")

await publishModule.publishPackages(publishModule.publishQueue, "latest")

// Homebrew last: it needs the release assets above, and a failure here must not undo the npm
// publish. Pushing to lacymorrow/homebrew-tap is cross-repo, so Actions' GITHUB_TOKEN is not enough.
// updateRegistries reads the archives through paths like ./dist/lashcode-linux-x64.tar.gz, so it has
// to run from packages/opencode rather than from the root we chdir'd back to.
try {
  process.chdir(path.join(dir, "packages", "opencode"))
  await publishModule.updateRegistries(publishModule.lashBinaries)
  process.chdir(dir)
} catch (e) {
  process.chdir(dir)
  console.error("Homebrew tap update failed. The GitHub release and the npm publish are done.")
  console.error("Set a HOMEBREW_TAP_TOKEN secret with write access to lacymorrow/homebrew-tap, then rerun.")
  throw e
}

console.log(`\nReleased lash ${tag}.`)
