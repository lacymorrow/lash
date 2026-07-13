import { Effect } from "effect"
import { AppRuntime, type AppServices } from "../../src/effect/app-runtime"
import { InstanceRef } from "../../src/effect/instance-ref"
import { context as instanceContext } from "../../src/project/instance-context"

/**
 * Run an AppRuntime effect with InstanceRef taken from the ambient test instance.
 * `attach()` only reads InstanceRef off the current Effect fiber, so plain async
 * test code inside `provideTestInstance` must provide it explicitly.
 *
 * Kept separate from fixture.ts so tests that never touch AppRuntime don't load
 * the full app module graph.
 */
export function runTestApp<A, E>(effect: Effect.Effect<A, E, AppServices>): Promise<A> {
  return AppRuntime.runPromise(effect.pipe(Effect.provideService(InstanceRef, instanceContext.use())))
}
