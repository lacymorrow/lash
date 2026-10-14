/**
 * Version selection for a lash release, kept pure so it can be tested without a network or a
 * git checkout. `release-lash.ts` gathers the inputs and calls this.
 */

export interface NextVersionInput {
  /** Newest version on npm, from registry.npmjs.org/lashcode/latest. */
  published?: string
  /** Tag of the newest GitHub release, without the leading v. */
  released?: string
  /** Every tag that exists in the repo, with their leading v. */
  tags: Iterable<string>
}

/**
 * Patch-bump whichever of npm and GitHub is further ahead, then walk past any version whose tag
 * already exists. v1.7.14 was tagged by a release run that died before it published anything, and
 * reusing that tag would ship an older commit than HEAD.
 */
export function nextVersion(input: NextVersionInput) {
  const base = [input.published, input.released]
    .filter((v): v is string => !!v)
    .sort(Bun.semver.order)
    .at(-1)
  if (!base) throw new Error("Could not read the last released version from npm or from GitHub releases")

  const tags = new Set(input.tags)
  const [major, minor, patch] = base.split(".").map((part) => Number(part) || 0)

  for (let candidate = patch + 1; candidate <= patch + 50; candidate++) {
    const version = `${major}.${minor}.${candidate}`
    if (!tags.has(`v${version}`)) return version
  }
  throw new Error(`Could not find an unused version after ${base}; pass OPENCODE_VERSION explicitly`)
}
