#!/usr/bin/env bun

import { $ } from "bun"
import os from "os"
import path from "path"

const REGISTRY = "registry.npmjs.org"
const TOKEN_HELP = [
  "Create a granular access token for 'lashcode' at https://www.npmjs.com/settings/lacymorrow/tokens",
  "with read and write permission on the package, then store it:",
  "  gh secret set NPM_TOKEN --repo lacymorrow/lash",
  "Granular automation tokens bypass 2FA on publish, which is what lets a release run unattended.",
].join("\n")

export const npmToken = process.env["NPM_TOKEN"] || process.env["NODE_AUTH_TOKEN"]

let userconfig: string | undefined

/**
 * Point npm at a throwaway userconfig holding the token so `npm publish` authenticates without
 * an OTP. Without a token we leave npm's own config alone and the caller falls back to prompting.
 */
export async function configureNpmAuth() {
  if (!npmToken) return
  if (userconfig) return
  userconfig = path.join(os.tmpdir(), `lash-npmrc-${process.pid}`)
  await Bun.write(userconfig, `//${REGISTRY}/:_authToken=${npmToken}\n`)
  await $`chmod 600 ${userconfig}`.nothrow().quiet()
  process.env["npm_config_userconfig"] = userconfig
}

/**
 * Check the token against the registry up front. A dead token should cost two seconds, not a
 * fifteen minute build followed by a 401.
 */
export async function verifyNpmAuth() {
  if (!npmToken) {
    console.log("npm auth: no NPM_TOKEN set, publish will prompt for an OTP")
    return
  }
  await configureNpmAuth()
  const whoami = await $`npm whoami`.nothrow().quiet()
  if (whoami.exitCode !== 0) {
    throw new Error(
      ["NPM_TOKEN was rejected by the registry.", whoami.stderr.toString().trim(), "", TOKEN_HELP].join("\n"),
    )
  }
  console.log(`npm auth: authenticated as ${whoami.stdout.toString().trim()} via NPM_TOKEN`)
}
