import path from "path"
import { describe, expect, test } from "bun:test"
import { Session } from "../../src/session/session"
import { SessionPrompt } from "../../src/session/prompt"
import { runTestApp } from "../fixture/app"
import { provideTestInstance, tmpdir } from "../fixture/fixture"

function sessionCreate(input?: Session.CreateInput) {
  return runTestApp(Session.Service.use((svc) => svc.create(input)))
}

function sessionRemove(id: Session.Info["id"]) {
  return runTestApp(Session.Service.use((svc) => svc.remove(id)))
}

function sessionPrompt(input: SessionPrompt.PromptInput) {
  return runTestApp(SessionPrompt.Service.use((svc) => svc.prompt(input)))
}

describe("session.prompt missing file", () => {
  test("does not fail the prompt when a file part is missing", async () => {
    await using tmp = await tmpdir({
      git: true,
      config: {
        agent: {
          build: {
            model: "openai/gpt-5.2",
          },
        },
      },
    })

    await provideTestInstance({
      directory: tmp.path,
      fn: async () => {
        const session = await sessionCreate({})

        const missing = path.join(tmp.path, "does-not-exist.ts")
        const msg = await sessionPrompt({
          sessionID: session.id,
          agent: "build",
          noReply: true,
          parts: [
            { type: "text", text: "please review @does-not-exist.ts" },
            {
              type: "file",
              mime: "text/plain",
              url: `file://${missing}`,
              filename: "does-not-exist.ts",
            },
          ],
        })

        if (msg.info.role !== "user") throw new Error("expected user message")

        const hasFailure = msg.parts.some(
          (part) => part.type === "text" && part.synthetic && part.text.includes("Read tool failed to read"),
        )
        expect(hasFailure).toBe(true)

        await sessionRemove(session.id)
      },
    })
  })
})
