import { afterAll, describe, expect, test } from "bun:test"
import { $ } from "bun"
import { mkdtemp, rm } from "fs/promises"
import os from "os"
import path from "path"

const script = path.resolve(import.meta.dir, "../install")
const root = await mkdtemp(path.join(os.tmpdir(), "lash-install-test-"))
afterAll(() => rm(root, { recursive: true, force: true }))

// Stands in for curl: answers the lash release API and release downloads, and
// refuses every other host so a wrong-repo URL fails the test instead of
// reaching the network. HEAD requests for a release asset report ASSET_STATUS
// (200 by default).
const curl = `#!/usr/bin/env bash
echo "$*" >> "$CURL_LOG"
out=""
url=""
head=false
while [ $# -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    -w|--max-time) shift 2 ;;
    -I|-sI|-sIL) head=true; shift ;;
    -*) shift ;;
    *) url="$1"; shift ;;
  esac
done
case "$url" in
  https://api.github.com/repos/lacymorrow/lash/releases/latest) printf '%s' "$GITHUB_BODY" ;;
  https://github.com/lacymorrow/lash/releases/download/*)
    if [ "$head" = true ]; then printf '%s' "\${ASSET_STATUS:-200}"; else cp "$ARCHIVE" "$out"; fi ;;
  *) exit 22 ;;
esac
`

const bin = path.join(root, "bin")
await Bun.write(path.join(bin, "curl"), curl)
await $`chmod +x ${path.join(bin, "curl")}`

// The installer picks its extractor from the host OS: tar on linux, unzip
// everywhere else. Build whichever archive this platform will reach for so the
// download path is exercised on macOS too, not just in linux CI.
const payload = path.join(root, "payload")
await Bun.write(path.join(payload, "lash"), "#!/bin/sh\necho installed\n")
const archive = path.join(root, process.platform === "linux" ? "lashcode.tar.gz" : "lashcode.zip")
if (process.platform === "linux") await $`tar -czf ${archive} -C ${payload} lash`
else await $`zip -q ${archive} lash`.cwd(payload)

let run = 0
async function install(input: { env?: Record<string, string>; args?: string[] }) {
  const home = path.join(root, `home-${run++}`)
  const log = path.join(home, "curl.log")
  await Bun.write(log, "")
  const proc = Bun.spawn(["bash", script, "--no-modify-path", ...(input.args ?? [])], {
    env: {
      PATH: `${bin}:/usr/bin:/bin`,
      HOME: home,
      SHELL: "/bin/bash",
      TMPDIR: home,
      CURL_LOG: log,
      ARCHIVE: archive,
      GITHUB_BODY: '{"tag_name": "v1.7.14"}',
      ...input.env,
    },
    stdout: "pipe",
    stderr: "pipe",
  })
  const [stdout, stderr, code] = await Promise.all([
    new Response(proc.stdout).text(),
    new Response(proc.stderr).text(),
    proc.exited,
  ])
  const installDir = input.env?.LASH_INSTALL_DIR ?? path.join(home, ".lash", "bin")
  return {
    code,
    output: stdout + stderr,
    curl: (await Bun.file(log).text()).trim().split("\n").filter(Boolean),
    installed: await Bun.file(path.join(installDir, "lash")).exists(),
  }
}

describe("install script", () => {
  test("installs an explicit version", async () => {
    const result = await install({ env: { VERSION: "1.7.14" } })
    expect(result.code).toBe(0)
    expect(result.installed).toBe(true)
    expect(result.curl.at(-1)).toContain("releases/download/v1.7.14/lashcode-")
  })

  test("accepts a v-prefixed version", async () => {
    const result = await install({ args: ["--version", "v1.7.14"] })
    expect(result.code).toBe(0)
    expect(result.installed).toBe(true)
    expect(result.curl.at(-1)).toContain("releases/download/v1.7.14/")
  })

  test("reads the latest version from the lash release API", async () => {
    const result = await install({ env: { GITHUB_BODY: '{"tag_name": "v1.7.9"}' } })
    expect(result.code).toBe(0)
    expect(result.installed).toBe(true)
    expect(result.curl.at(-1)).toContain("releases/download/v1.7.9/")
  })

  // Guards the 2026-10-19 upstream merge regression: upstream rewrote release
  // discovery to gate OpenCode 1 against OpenCode 2, and the non-conflicting
  // half of that hunk merged a hard-coded anomalyco/opencode download URL into
  // this installer, re-breaking what PR #48 had just fixed. The stub curl exits
  // 22 on any other host, so a wrong repo also fails the assertions above; this
  // test says why out loud.
  test("never contacts the upstream opencode repo", async () => {
    for (const result of [await install({}), await install({ env: { VERSION: "1.7.14" } })]) {
      expect(result.code).toBe(0)
      expect(result.curl.join("\n")).not.toContain("anomalyco")
      expect(result.curl.join("\n")).not.toContain("opencode")
      expect(result.curl.every((line) => line.includes("lacymorrow/lash"))).toBe(true)
    }
  })

  test("honors LASH_INSTALL_DIR", async () => {
    const dir = path.join(root, "custom-bin")
    const result = await install({ env: { VERSION: "1.7.14", LASH_INSTALL_DIR: dir } })
    expect(result.code).toBe(0)
    expect(result.installed).toBe(true)
  })

  test("reports a missing explicit release", async () => {
    const result = await install({ env: { VERSION: "1.7.99", ASSET_STATUS: "404" } })
    expect(result.code).toBe(1)
    expect(result.output).toContain("Release v1.7.99 not found")
    expect(result.installed).toBe(false)
  })

  test("reports a release API that returns nothing", async () => {
    const result = await install({ env: { GITHUB_BODY: "" } })
    expect(result.code).toBe(1)
    expect(result.output).toContain("Failed to fetch version information")
    expect(result.installed).toBe(false)
  })
})
