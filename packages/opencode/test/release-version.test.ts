import { describe, expect, test } from "bun:test"
import { nextVersion } from "../script/release-version"

describe("nextVersion", () => {
  test("patch bumps from the last release", () => {
    expect(nextVersion({ published: "1.7.13", released: "1.7.13", tags: [] })).toBe("1.7.14")
  })

  test("skips a version whose tag already exists", () => {
    // The state LAC-4453 found: npm and the newest release both sat at 1.7.13 while v1.7.14 was
    // tagged by a run that never published, so 1.7.14 must not be reused.
    expect(nextVersion({ published: "1.7.13", released: "1.7.13", tags: ["v1.7.13", "v1.7.14"] })).toBe("1.7.15")
  })

  test("skips a run of taken tags", () => {
    const tags = ["v1.7.14", "v1.7.15", "v1.7.16"]
    expect(nextVersion({ published: "1.7.13", released: "1.7.13", tags })).toBe("1.7.17")
  })

  test("follows whichever source is ahead", () => {
    expect(nextVersion({ published: "1.7.13", released: "1.8.2", tags: [] })).toBe("1.8.3")
    expect(nextVersion({ published: "1.9.0", released: "1.8.2", tags: [] })).toBe("1.9.1")
  })

  test("works when only one source is known", () => {
    expect(nextVersion({ published: "1.7.13", tags: [] })).toBe("1.7.14")
    expect(nextVersion({ released: "1.7.13", tags: [] })).toBe("1.7.14")
  })

  test("ignores unrelated tags", () => {
    // The fork carries upstream tags like v4.7 and v2.0.0; they must not steer the lash version.
    expect(nextVersion({ published: "1.7.13", released: "1.7.13", tags: ["v4.7", "v2.0.0"] })).toBe("1.7.14")
  })

  test("refuses to guess with no known release", () => {
    expect(() => nextVersion({ tags: [] })).toThrow(/Could not read the last released version/)
  })

  test("gives up rather than bumping forever", () => {
    const tags = Array.from({ length: 60 }, (_, i) => `v1.7.${14 + i}`)
    expect(() => nextVersion({ published: "1.7.13", tags })).toThrow(/pass OPENCODE_VERSION/)
  })
})
