/**
 * Working directory state management for shell mode.
 * Tracks the current working directory independently from Instance.directory.
 */

import { context as instanceContext } from "@/project/instance-context"
import { Bus } from "@/bus"
import { BusEvent } from "@/bus/bus-event"
import path from "path"
import os from "os"
import { Schema } from "effect"

let currentCwd: string | null = null

/**
 * Event published when the working directory changes.
 */
export const CwdEvent = {
  Updated: BusEvent.define(
    "cwd.updated",
    Schema.Struct({
      cwd: Schema.String,
    }),
  ),
}

/**
 * Get the current working directory.
 * Returns Instance.directory if not explicitly set.
 */
export function getCwd(): string {
  if (currentCwd === null) {
    try {
      return instanceContext.use().directory
    } catch {
      return process.cwd()
    }
  }
  return currentCwd
}

/**
 * Set the current working directory.
 * Handles ~ expansion and relative path resolution.
 */
export function setCwd(dir: string): void {
  let resolved = dir

  // Handle ~ expansion
  if (resolved.startsWith("~")) {
    resolved = path.join(os.homedir(), resolved.slice(1))
  }

  // Resolve relative paths against current cwd
  if (!path.isAbsolute(resolved)) {
    resolved = path.resolve(getCwd(), resolved)
  }

  const changed = currentCwd !== resolved
  currentCwd = resolved

  // Publish event if cwd changed
  if (changed) {
    try {
      void Bus.publish(instanceContext.use(), CwdEvent.Updated, { cwd: resolved })
    } catch {
      // No instance context available; skip publishing
    }
  }
}

/**
 * Reset the working directory to Instance.directory.
 */
export function resetCwd(): void {
  currentCwd = null
}
