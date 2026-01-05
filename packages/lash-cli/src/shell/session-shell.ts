import { spawn, type ChildProcessWithoutNullStreams } from "child_process"
import path from "path"
import { Log } from "opencode/src/util/log"
import { Instance } from "opencode/src/project/instance"
import { Bus } from "opencode/src/bus/index"
import { ulid } from "ulid"

const log = Log.create({ service: "session-shell" })

type ExecOptions = {
  sessionID: string
  command: string
  signal: AbortSignal
  onData?: (chunk: string) => void
}

export type ExecResult = {
  output: string
  exitCode: number
  cwd?: string
}

type Running = {
  id: string
  sentinel: string
  output: string
  streamed: number
  resolve: (result: ExecResult) => void
  reject: (error: Error) => void
  onData?: (chunk: string) => void
  abortHandler: () => void
}

class ShellProcess {
  readonly shellPath: string
  readonly shellName: string
  private proc: ChildProcessWithoutNullStreams
  private running: Running | undefined
  private chain: Promise<void>
  private closed: boolean

  constructor(shellPath: string) {
    this.shellPath = shellPath
    this.shellName = path.basename(shellPath)
    this.proc = spawnShell(shellPath)
    this.running = undefined
    this.chain = Promise.resolve()
    this.closed = false

    this.proc.once("exit", (code, signal) => {
      this.closed = true
      const active = this.running
      if (!active) return
      this.running = undefined
      active.reject(
        new Error(
          `Shell exited while running command (code=${code ?? "null"}, signal=${signal ?? "null"})`,
        ),
      )
    })
  }

  execute(command: string, options: { signal: AbortSignal; onData?: (chunk: string) => void }) {
    const current = this.chain.then(() => this.executeInternal(command, options))
    this.chain = current.then(() => undefined).catch(() => undefined)
    return current
  }

  dispose() {
    this.closed = true
    const active = this.running
    if (active) {
      this.running = undefined
      active.reject(new Error("Shell disposed"))
    }
    killProcessTree(this.proc)
  }

  get isClosed() {
    return this.closed
  }

  private executeInternal(command: string, options: { signal: AbortSignal; onData?: (chunk: string) => void }) {
    if (this.closed) {
      return Promise.reject(new Error("Shell is closed"))
    }
    if (options.signal.aborted) {
      return Promise.reject(new Error("Command aborted"))
    }

    const id = ulid()
    const sentinel = `__OPENCODE_DONE__${id}__:`

    const abortHandler = () => {
      const active = this.running
      if (!active || active.id !== id) return
      this.closed = true
      this.running = undefined
      active.reject(new Error("Command aborted"))
      killProcessTree(this.proc)
    }

    return new Promise<ExecResult>((resolve, reject) => {
      const running: Running = {
        id,
        sentinel,
        output: "",
        streamed: 0,
        resolve,
        reject,
        onData: options.onData,
        abortHandler,
      }
      this.running = running
      options.signal.addEventListener("abort", abortHandler, { once: true })

      const onStdout = (chunk: unknown) => this.onChunk(running, String(chunk))
      const onStderr = (chunk: unknown) => this.onChunk(running, String(chunk))

      this.proc.stdout.on("data", onStdout)
      this.proc.stderr.on("data", onStderr)

      const cleanup = () => {
        this.proc.stdout.off("data", onStdout)
        this.proc.stderr.off("data", onStderr)
        options.signal.removeEventListener("abort", abortHandler)
      }

      const finish = (result: ExecResult) => {
        cleanup()
        if (this.running?.id === id) {
          this.running = undefined
        }
        resolve(result)
      }

      const fail = (error: Error) => {
        cleanup()
        if (this.running?.id === id) {
          this.running = undefined
        }
        reject(error)
      }

      running.resolve = finish
      running.reject = fail

      const payload = wrapCommand(this.shellName, command, sentinel)
      const ok = this.proc.stdin.write(payload)
      if (ok) return
      this.proc.stdin.once("drain", () => {
        if (!this.running || this.running.id !== id) return
        // Try writing again after drain event
        try {
          const writeOk = this.proc.stdin.write(payload)
          if (!writeOk) {
            // If write still fails after drain, reject the command
            fail(new Error("Failed to write command to shell after drain"))
          }
        } catch (error) {
          fail(error instanceof Error ? error : new Error("Failed to write command to shell"))
        }
      })
    })
  }

  private onChunk(running: Running, chunk: string) {
    if (!this.running || this.running.id !== running.id) return

    running.output += chunk

    const index = running.output.indexOf(running.sentinel)
    if (running.onData) {
      if (index === -1) {
        const hold = running.sentinel.length
        const end = Math.max(running.output.length - hold, running.streamed)
        const next = running.output.slice(running.streamed, end)
        if (next) {
          running.streamed = end
          running.onData(next)
        }
      }
      if (index !== -1) {
        const next = running.output.slice(running.streamed, index)
        if (next) {
          running.streamed = index
          running.onData(next)
        }
      }
    }

    if (index === -1) return

    const after = running.output.slice(index + running.sentinel.length)
    const newline = after.indexOf("\n")
    if (newline === -1) return

    const meta = after.slice(0, newline).trim()
    const tab = meta.indexOf("\t")
    const exitCodeText = (tab === -1 ? meta : meta.slice(0, tab)).trim()
    const cwd = tab === -1 ? undefined : meta.slice(tab + 1).trim()
    const exitCode = Number(exitCodeText)
    const output = running.output.slice(0, index)
    running.resolve({
      output,
      exitCode: Number.isFinite(exitCode) ? exitCode : 1,
      cwd: cwd || undefined,
    })
  }
}

function spawnShell(shellPath: string) {
  const shellName = path.basename(shellPath)
  const args = resolveShellArgs(shellName)
  const proc = spawn(shellPath, args, {
    cwd: Instance.directory,
    detached: process.platform !== "win32",
    stdio: ["pipe", "pipe", "pipe"],
    env: {
      ...process.env,
      TERM: "dumb",
    },
  })
  proc.stdin.setDefaultEncoding("utf8")
  proc.stdout.setEncoding("utf8")
  proc.stderr.setEncoding("utf8")
  proc.stdin.write(shellInit(shellName))
  return proc
}

function resolveShellArgs(shellName: string) {
  if (shellName === "zsh") return ["-l", "-s"]
  if (shellName === "bash") return ["-l", "-s"]
  return ["-s"]
}

function shellInit(shellName: string) {
  if (shellName === "zsh") {
    return `
[[ -f ~/.zshenv ]] && source ~/.zshenv >/dev/null 2>&1 || true
[[ -f "\${ZDOTDIR:-$HOME}/.zshrc" ]] && source "\${ZDOTDIR:-$HOME}/.zshrc" >/dev/null 2>&1 || true
`
  }
  if (shellName === "bash") {
    return `
[[ -f ~/.bashrc ]] && source ~/.bashrc >/dev/null 2>&1 || true
shopt -s expand_aliases >/dev/null 2>&1 || true
`
  }
  return ""
}

function wrapCommand(shellName: string, command: string, sentinel: string) {
  const end = sentinel.replaceAll("'", "'\\''")
  if (shellName === "zsh" || shellName === "bash") {
    return `
{
${command}
}
__opencode_status__=$?
__opencode_cwd__="$(pwd -P 2>/dev/null || pwd)"
printf '\\n${end}%s\\t%s\\n' "$__opencode_status__" "$__opencode_cwd__"
`
  }
  return `
${command}
__opencode_status__=$?
__opencode_cwd__="$(pwd -P 2>/dev/null || pwd)"
printf '\\n${end}%s\\t%s\\n' "$__opencode_status__" "$__opencode_cwd__"
`
}

function killProcessTree(proc: ChildProcessWithoutNullStreams) {
  const pid = proc.pid
  if (!pid) return

  if (process.platform === "win32") {
    proc.kill("SIGTERM")
    return
  }

  try {
    process.kill(-pid, "SIGTERM")
  } catch (_e) {
    // ignore
  }

  try {
    proc.kill("SIGTERM")
  } catch (_e) {
    // ignore
  }

  const timer = setTimeout(() => {
    try {
      process.kill(-pid, "SIGKILL")
      return
    } catch (_e) {
      // ignore
    }
    try {
      proc.kill("SIGKILL")
    } catch (_e) {
      // ignore
    }
  }, 200)
  timer.unref?.()
}

const state = Instance.state(
  () => {
    const shells = new Map<string, ShellProcess>()
    const timers = new Map<string, Timer>()
    const lastUsed = new Map<string, number>()
    const ready: { init: boolean; unsub?: () => void } = { init: false }
    return { shells, timers, lastUsed, ready }
  },
  async (current) => {
    for (const shell of current.shells.values()) {
      shell.dispose()
    }
    current.shells.clear()
    for (const timer of current.timers.values()) {
      clearTimeout(timer)
    }
    current.timers.clear()
    current.lastUsed.clear()
    current.ready.unsub?.()
    current.ready.init = false
    current.ready.unsub = undefined
  },
)

type Timer = ReturnType<typeof setTimeout>

function initOnce() {
  const ready = state().ready
  if (ready.init) return
  ready.init = true

  ready.unsub = Bus.subscribeAll((event) => {
    if (event.type !== "session.idle") return
    const sessionID = event.properties?.sessionID as string | undefined
    if (!sessionID) return
    scheduleDispose(sessionID)
  })
}

function getShellPath() {
  const input = process.env["SHELL"]
  if (input) return input
  return "bash"
}

function getOrCreate(sessionID: string) {
  const shells = state().shells
  const existing = shells.get(sessionID)
  if (existing && !existing.isClosed) return existing

  if (existing) {
    shells.delete(sessionID)
  }

  const shellPath = getShellPath()
  const next = new ShellProcess(shellPath)
  shells.set(sessionID, next)
  return next
}

function touch(sessionID: string) {
  state().lastUsed.set(sessionID, Date.now())
  const timer = state().timers.get(sessionID)
  if (!timer) return
  clearTimeout(timer)
  state().timers.delete(sessionID)
}

function scheduleDispose(sessionID: string) {
  const existing = state().timers.get(sessionID)
  if (existing) return

  const timer = setTimeout(() => {
    // Check if shell still exists before attempting disposal
    const shells = state().shells
    if (!shells.has(sessionID)) {
      // Already disposed, just clean up timer reference
      state().timers.delete(sessionID)
      state().lastUsed.delete(sessionID)
      return
    }

    const last = state().lastUsed.get(sessionID) ?? 0
    const idleForMs = Date.now() - last
    if (idleForMs < 2 * 60 * 1000) {
      state().timers.delete(sessionID)
      scheduleDispose(sessionID)
      return
    }
    dispose(sessionID)
  }, 2 * 60 * 1000)
  timer.unref?.()

  state().timers.set(sessionID, timer)
}

export function dispose(sessionID: string) {
  const shells = state().shells
  const shell = shells.get(sessionID)
  if (shell) {
    shell.dispose()
    shells.delete(sessionID)
  }
  const timer = state().timers.get(sessionID)
  if (timer) {
    clearTimeout(timer)
    state().timers.delete(sessionID)
  }
  state().lastUsed.delete(sessionID)
}

export async function execute(options: ExecOptions) {
  initOnce()
  touch(options.sessionID)

  const shell = getOrCreate(options.sessionID)
  const result = await shell.execute(options.command, {
    signal: options.signal,
    onData: options.onData,
  })

  touch(options.sessionID)
  log.debug("executed", {
    sessionID: options.sessionID,
    shell: shell.shellName,
    exitCode: result.exitCode,
  })

  return result
}
