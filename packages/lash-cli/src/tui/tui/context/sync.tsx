import type {
  Message,
  Agent,
  Provider,
  Session,
  Part,
  Config,
  Todo,
  Command,
  Permission,
  LspStatus,
  McpStatus,
  FormatterStatus,
} from "@opencode-ai/sdk"
import { createStore, produce, reconcile } from "solid-js/store"
import { useSDK } from "@tui/context/sdk"
import { Binary } from "@/util/binary"
import { createSimpleContext } from "./helper"

export const { use: useSync, provider: SyncProvider } = createSimpleContext({
  name: "Sync",
  init: () => {
    const [store, setStore] = createStore<{
      ready: boolean
      provider: Provider[]
      agent: Agent[]
      command: Command[]
      permission: {
        [sessionID: string]: Permission[]
      }
      config: Config
      session: Session[]
      todo: {
        [sessionID: string]: Todo[]
      }
      message: {
        [sessionID: string]: Message[]
      }
      part: {
        [messageID: string]: Part[]
      }
      lsp: LspStatus[]
      mcp: {
        [key: string]: McpStatus
      }
      formatter: FormatterStatus[]
    }>({
      config: {},
      ready: false,
      agent: [],
      permission: {},
      command: [],
      provider: [],
      session: [],
      todo: {},
      message: {},
      part: {},
      lsp: [],
      mcp: {},
      formatter: [],
    })

    const sdk = useSDK()

    sdk.event.listen((e) => {
      const event = e.details
      switch (event.type) {
        case "permission.updated": {
          const permissions = store.permission[event.properties.sessionID]
          if (!permissions) {
            setStore("permission", event.properties.sessionID, [event.properties])
            break
          }
          const match = Binary.search(permissions, event.properties.id, (p) => p.id)
          setStore(
            "permission",
            event.properties.sessionID,
            produce((draft) => {
              if (match.found) {
                draft[match.index] = event.properties
                return
              }
              draft.push(event.properties)
            }),
          )
          break
        }

        case "permission.replied": {
          const permissions = store.permission[event.properties.sessionID]
          const match = Binary.search(permissions, event.properties.permissionID, (p) => p.id)
          if (!match.found) break
          setStore(
            "permission",
            event.properties.sessionID,
            produce((draft) => {
              draft.splice(match.index, 1)
            }),
          )
          break
        }

        case "todo.updated":
          setStore("todo", event.properties.sessionID, event.properties.todos)
          break

        case "session.deleted": {
          const result = Binary.search(store.session, event.properties.info.id, (s) => s.id)
          if (result.found) {
            setStore(
              "session",
              produce((draft) => {
                draft.splice(result.index, 1)
              }),
            )
          }
          break
        }
        case "session.updated":
          const result = Binary.search(store.session, event.properties.info.id, (s) => s.id)
          if (result.found) {
            setStore("session", result.index, reconcile(event.properties.info))
            break
          }
          setStore(
            "session",
            produce((draft) => {
              draft.splice(result.index, 0, event.properties.info)
            }),
          )
          break
        case "message.updated": {
          const messages = store.message[event.properties.info.sessionID]
          if (!messages) {
            setStore("message", event.properties.info.sessionID, [event.properties.info])
            break
          }
          const result = Binary.search(messages, event.properties.info.id, (m) => m.id)
          if (result.found) {
            const incoming = event.properties.info
            setStore("message", incoming.sessionID, result.index, (previous) => {
              if (!previous) return incoming
              if (previous.role !== "assistant") return reconcile(incoming)(previous)
              if (incoming.role !== "assistant") return reconcile(incoming)(previous)

              const previousCompleted = previous.time.completed
              const incomingCompleted = incoming.time.completed
              if (previousCompleted && !incomingCompleted) return previous
              if (previousCompleted && incomingCompleted && incomingCompleted < previousCompleted) return previous

              return reconcile(incoming)(previous)
            })
            break
          }
          setStore(
            "message",
            event.properties.info.sessionID,
            produce((draft) => {
              draft.splice(result.index, 0, event.properties.info)
            }),
          )
          break
        }
        case "message.removed": {
          const messages = store.message[event.properties.sessionID]
          const result = Binary.search(messages, event.properties.messageID, (m) => m.id)
          if (result.found) {
            setStore(
              "message",
              event.properties.sessionID,
              produce((draft) => {
                draft.splice(result.index, 1)
              }),
            )
          }
          break
        }
        case "message.part.updated": {
          const parts = store.part[event.properties.part.messageID]
          if (!parts) {
            setStore("part", event.properties.part.messageID, [event.properties.part])
            break
          }
          const result = Binary.search(parts, event.properties.part.id, (p) => p.id)
          if (result.found) {
            setStore("part", event.properties.part.messageID, result.index, reconcile(event.properties.part))
            break
          }
          setStore(
            "part",
            event.properties.part.messageID,
            produce((draft) => {
              draft.splice(result.index, 0, event.properties.part)
            }),
          )
          break
        }

        case "message.part.removed": {
          const parts = store.part[event.properties.messageID]
          const result = Binary.search(parts, event.properties.partID, (p) => p.id)
          if (result.found)
            setStore(
              "part",
              event.properties.messageID,
              produce((draft) => {
                draft.splice(result.index, 1)
              }),
            )
          break
        }

        case "lsp.updated": {
          sdk.client.lsp.status().then((x) => setStore("lsp", x.data!))
          break
        }
      }
    })

    // blocking
    Promise.all([
      sdk.client.config.providers().then((x) => setStore("provider", x.data!.providers)),
      sdk.client.app.agents().then((x) => setStore("agent", x.data ?? [])),
      sdk.client.config.get().then((x) => setStore("config", x.data!)),
    ]).then(() => setStore("ready", true))

    // non-blocking
    Promise.all([
      sdk.client.session.list().then((x) =>
        setStore(
          "session",
          (x.data ?? []).toSorted((a, b) => a.id.localeCompare(b.id)),
        ),
      ),
      sdk.client.command.list().then((x) => setStore("command", x.data ?? [])),
      sdk.client.lsp.status().then((x) => setStore("lsp", x.data!)),
      sdk.client.mcp.status().then((x) => setStore("mcp", x.data!)),
      sdk.client.formatter.status().then((x) => setStore("formatter", x.data!)),
    ])

    const result = {
      data: store,
      set: setStore,
      get ready() {
        return store.ready
      },
      session: {
        get(sessionID: string) {
          const match = Binary.search(store.session, sessionID, (s) => s.id)
          if (match.found) return store.session[match.index]
          return undefined
        },
        status(sessionID: string) {
          const session = result.session.get(sessionID)
          if (!session) return "idle"
          if (session.time.compacting) return "compacting"
          const messages = store.message[sessionID] ?? []
          const last = messages.at(-1)
          if (!last) return "idle"
          if (last.role === "user") return "working"
          return last.time.completed ? "idle" : "working"
        },
        async sync(sessionID: string) {
          const [session, messages, todo] = await Promise.all([
            sdk.client.session.get({ path: { id: sessionID }, throwOnError: true }),
            sdk.client.session.messages({ path: { id: sessionID } }),
            sdk.client.session.todo({ path: { id: sessionID } }),
          ])
          setStore(
            produce((draft) => {
              const match = Binary.search(draft.session, sessionID, (s) => s.id)
              if (match.found) {
                const existing = draft.session[match.index]
                if (!existing || existing.time.updated <= session.data!.time.updated) {
                  draft.session[match.index] = session.data!
                }
              }
              if (!match.found) draft.session.splice(match.index, 0, session.data!)
              draft.todo[sessionID] = todo.data ?? []
              const existing = draft.message[sessionID] ?? []
              const merged = messages.data!.map((x) => {
                const incoming = x.info
                const found = Binary.search(existing, incoming.id, (m) => m.id)
                if (!found.found) return incoming

                const previous = existing[found.index]
                if (previous.role !== "assistant") return incoming
                if (incoming.role !== "assistant") return incoming

                const previousCompleted = previous.time.completed
                const incomingCompleted = incoming.time.completed
                if (previousCompleted && !incomingCompleted) return previous
                if (previousCompleted && incomingCompleted && incomingCompleted <= previousCompleted) return previous

                return incoming
              })
              // Keep any messages we already have that the fetch didn't include yet.
              for (const msg of existing) {
                const found = Binary.search(merged, msg.id, (m) => m.id)
                if (!found.found) merged.splice(found.index, 0, msg)
              }
              draft.message[sessionID] = merged
              for (const message of messages.data!) {
                draft.part[message.info.id] = message.parts
              }
            }),
          )
        },
      },
    }
    return result
  },
})
