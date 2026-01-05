import { createCli } from "opencode";
import { AttachCommand } from "./tui/attach";
import { TuiThreadCommand } from "./tui/thread";
import { TuiSpawnCommand } from "./tui/spawn";
import { ToolRegistry } from "opencode/src/tool/registry";
import { ShellTool } from "./shell/shell";

// Register default overrides
ToolRegistry.register(ShellTool);

// Initialize CLI with upstream logic but exclude TUI commands we want to override
const cli = createCli(process.argv, {
    exclude: ["tui/thread", "tui/attach", "tui/spawn"]
});

// Register Lash-specific TUI commands
cli.command(AttachCommand);
cli.command(TuiThreadCommand);
cli.command(TuiSpawnCommand);

try {
    await cli.parse();
} catch (e) {
    console.error("Fatal error:", e);
    process.exit(1);
}
