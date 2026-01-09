#!/usr/bin/env bun
import solidPlugin from "../../../node_modules/@opentui/solid/scripts/solid-plugin"
import { $ } from "bun"
import path from "path"
import { fileURLToPath } from "url"

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const dir = path.resolve(__dirname, "..")
process.chdir(dir)

await $`rm -rf dist`
await $`mkdir -p dist`

console.log("Building lash-core...")

await Bun.build({
    entrypoints: ["./src/index.ts"],
    outdir: "./dist",
    target: "bun",
    plugins: [solidPlugin],
    external: [
        "opencode",
        "@opencode-ai/plugin",
        "@opentui/solid",
        "@opentui/core",
        "@opentui/context",
        "@opentui/ui",
        "@opentui/component"
    ],
    sourcemap: "external",
})

console.log("Build complete")
