import { cmd } from "../cmd"
import { tui } from "./app"
import { Instance } from "@/project/instance"

export const AttachCommand = cmd({
  command: "attach <url>",
  describe: "attach to a running opencode server",
  builder: (yargs) =>
    yargs
      .positional("url", {
        type: "string",
        describe: "http://localhost:4096",
        demandOption: true,
      })
      .option("dir", {
        type: "string",
        description: "directory to run in",
      })
      .option("session", {
        alias: ["s"],
        type: "string",
        describe: "session id to continue",
      }),
  handler: async (args) => {
    if (args.dir) process.chdir(args.dir)
    await Instance.provide({
      directory: args.dir ? process.cwd() : process.cwd(),
      fn: () =>
        tui({
          url: args.url,
          args: { sessionID: args.session },
          directory: args.dir ? process.cwd() : undefined,
        }),
    })
  },
})
