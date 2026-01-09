import { DialogAgent } from "@tui/component/dialog-agent"
import { LashShell } from "./shell/shell"

export const LashUI = {
    getCommands: (ctx: { local: any, dialog: any, theme: any }) => {
        const { local, dialog, theme } = ctx
        const { mode, setMode } = theme

        return [
            {
                title: "Agent cycle",
                value: "agent.cycle",
                keybind: "agent_cycle",
                category: "Agent",
                onSelect: () => {
                    local.agent.move(1)
                },
            },
            {
                title: "Agent cycle reverse",
                value: "agent.cycle.reverse",
                keybind: "agent_cycle_reverse",
                category: "Agent",
                onSelect: () => {
                    local.agent.move(-1)
                },
            },
            {
                title: `Switch to ${mode() === "dark" ? "light" : "dark"} mode`,
                value: "theme.switch_mode",
                onSelect: () => {
                    setMode(mode() === "dark" ? "light" : "dark")
                },
                category: "System",
            },
            {
                title: "Switch agent",
                value: "agent.list",
                keybind: "agent_list",
                category: "Agent",
                onSelect: () => {
                    dialog.replace(() => <DialogAgent />)
                },
            }
        ]
    },
    getStatus: (ctx: any) => {
        return [() => <StatusItem ctx={ctx} />]
    }
}

function StatusItem(props: { ctx: any }) {
    const { theme } = props.ctx.theme
    const mode = () => LashShell.mode.getMode()

    return (
        <box flexDirection="row" gap={1}>
            <text fg={mode() === "AGENT" ? theme.primary : theme.textMuted}>
                {mode() === "AGENT" ? "AGENT" : "AUTO"}
            </text>
        </box>
    )
}
