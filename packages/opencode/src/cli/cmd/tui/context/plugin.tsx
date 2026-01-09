import { createContext, useContext, type Accessor } from "solid-js"

export interface PluginContextState {
    commands: any[]
    status: any[]
}

const PluginContext = createContext<Accessor<PluginContextState>>()

export function usePlugin() {
    const ctx = useContext(PluginContext)
    if (!ctx) throw new Error("usePlugin must be used within a PluginProvider")
    return ctx
}

export const PluginProvider = PluginContext.Provider
