import type { Plugin } from "@opencode-ai/plugin"
import { LashShell } from "./shell/shell"
import { LashUI } from "./ui"

export const lashCore: Plugin = async (input) => {
    console.log("---- LASH CORE PLUGIN INITIALIZING ----")
    return {
        "shell.execute": async (inp, out) => {
            const result = await LashShell.execute(inp, input)
            if (result) {
                out.executed = true
                out.result = result
            }
        },
        "ui.command": async (inp, out) => {
            out.commands = LashUI.getCommands(inp as any)
        },
        "ui.status": async (inp, out) => {
            out.components = LashUI.getStatus(inp as any)
        }
    }
}
