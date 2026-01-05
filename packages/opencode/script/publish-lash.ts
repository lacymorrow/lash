
import { $ } from "bun"
import { resolve } from "path"
import { rename } from "fs/promises"

// Set CWD to packages/opencode to ensure correct resolution
const opencodeDir = resolve(__dirname, "..")
process.chdir(opencodeDir)

const version = "1.0.7" // Ensure this matches package.json if needed
const name = "lash-cli"
const os = process.platform
const arch = process.arch

try {
  // Hide bunfig.toml so compilation doesn't bake it in
  const bunfigPath = "bunfig.toml"
  const tempBunfigPath = "bunfig.dev.toml"
  let renamed = false
  try {
    await rename(bunfigPath, tempBunfigPath)
    renamed = true
  } catch (e) {
    console.warn("Could not rename bunfig.toml (maybe already renamed?)")
  }

  try {
    // Import plugin dynamically (path relative to opencodeDir/script if running there, but we chdir to opencodeDir)
    // So relative to opencodeDir: node_modules/...
    const pluginPath = "./node_modules/@opentui/solid/scripts/solid-plugin.ts"
    // @ts-ignore
    const { default: solidTransformPlugin } = await import(pluginPath);

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

    // Calculate externals for Bun.build
    const externalPkgs = [
      ...opentuiPlatforms,
      ...watcherPlatforms,
      ...humblePlatforms
    ].filter(pkg => {
      // Keep host packages, externalize others
      return !pkg.includes(process.platform === 'win32' ? 'win32' : process.platform) || !pkg.includes(process.arch)
    })

    console.log("Bundling lash with solid plugin...")
    await Bun.build({
      entrypoints: ["./src/index.ts"],
      outdir: `dist/${name}/bin`,
      naming: "lash.js",
      target: "bun",
      plugins: [solidTransformPlugin],
      define: {
        OPENCODE_TUI_PATH: `'../../../dist/${name}/bin/tui'`,
        OPENCODE_VERSION: `'${version}'`
      },
      external: externalPkgs
    })

    console.log("Compiling binary...")
    const bundleOut = `dist/${name}/bin/lash.js`
    // Use --embed to ensure tui is included as asset
    await $`bun build --compile --target=bun-${os}-${arch} --outfile=dist/${name}/bin/lash ${bundleOut} --embed ./dist/${name}/bin/tui`.cwd(opencodeDir)

    // Run smoke test
    if (
      process.platform === (os === "windows" ? "win32" : os) &&
      process.arch === arch
    ) {
      console.log(`Smoke test: running dist/${name}/bin/lash --version`)
      const smoke = await $`./dist/${name}/bin/lash --version`.cwd(opencodeDir).text()
      if (!smoke.includes(version)) {
        throw new Error(`Smoke test failed: Version mismatch. Expected ${version}, got ${smoke}`)
      }
      console.log(`✓ Smoke test passed: ${smoke.trim()}`)
    }

  } finally {
    // Restore bunfig.toml
    if (renamed) {
      await rename(tempBunfigPath, bunfigPath)
    }
  }

} catch (e) {
  console.error(e)
  process.exit(1)
}