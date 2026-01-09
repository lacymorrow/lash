
import { lashCore } from "@lash-cli/core"

console.log("Imported lashCore:", lashCore)

try {
    const hooks = await lashCore({} as any)
    console.log("Initialized hooks:", Object.keys(hooks))

    if (hooks["ui.status"]) {
        const out = { components: [] }
        await hooks["ui.status"]({ theme: { theme: {}, mode: () => "AGENT" } } as any, out)
        console.log("Status output:", out.components)
    }

} catch (e) {
    console.error("Error running lashCore:", e)
}
