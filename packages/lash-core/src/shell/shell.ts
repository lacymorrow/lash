import { Shell } from "opencode/shell/shell"
import { Log } from "opencode/util/log"

const log = Log.create({ service: "lash-shell" })

export enum ExecutionMode {
    Shell = "Shell",
    Agent = "Agent",
    Auto = "Auto",
}

import { createSignal } from "solid-js"

class ModeController {
    private _mode = createSignal(ExecutionMode.Shell)

    getMode() {
        return this._mode[0]()
    }

    setMode(mode: ExecutionMode) {
        this._mode[1](mode)
        log.info("mode set", { mode })
    }

    toggleMode() {
        const value = this.getMode()
        const modes = Object.values(ExecutionMode)
        const index = modes.indexOf(value)
        const next = modes[(index + 1) % modes.length]!
        this.setMode(next)
        return next
    }
}

const modeController = new ModeController()

export const LashShell = {
    mode: modeController,

    async execute(params: any, ctx: any) {
        const effectiveMode = params.mode
            ? (params.mode.charAt(0).toUpperCase() + params.mode.slice(1) as ExecutionMode)
            : modeController.getMode()

        if (effectiveMode === ExecutionMode.Agent) {
            log.info("forced agent mode", { command: params.command })
            return {
                title: params.description,
                metadata: {
                    command: params.command,
                    workingDir: process.cwd(),
                    exitCode: 0,
                    description: params.description,
                    output: `[Agent mode: ${params.command}]`
                },
                output: `[Agent mode: ${params.command}]`
            }
        }
        return null
    }
}
