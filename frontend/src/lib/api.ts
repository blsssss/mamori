import * as App from '../../wailsjs/go/main/App'
import * as runtime from '../../wailsjs/runtime/runtime'
import type { AppInfo, CheckId, FindingEvent, Result, RunOptions, Summary } from './types'

// everything the interface needs from Go; the real one wraps the Wails bindings, the demo one in
// demo.ts fakes them for `npm run dev` in a plain browser
export interface Backend {
  readonly kind: 'wails' | 'demo' | 'none'
  readonly scenario?: string
  info(): Promise<AppInfo>
  run(id: CheckId, opts: RunOptions): Promise<Result>
  cancel(): Promise<void>
  summarize(results: Result[]): Promise<Summary>
  // resolves to the chosen path, or '' when the user closed the dialog
  saveReport(defaultName: string, content: string): Promise<string>
  copy(text: string): Promise<boolean>
  quit(): Promise<void>
  onFinding(cb: (e: FindingEvent) => void): () => void
}

type Bindings = { main?: { App?: unknown } }

export function hasWails(w: unknown = globalThis): boolean {
  const go = (w as { go?: Bindings }).go
  return typeof go?.main?.App === 'object' && go.main.App !== null
}

export function wailsBackend(): Backend {
  return {
    kind: 'wails',
    info: () => App.Info() as Promise<AppInfo>,
    run: (id, opts) => App.Run(id, opts) as unknown as Promise<Result>,
    cancel: () => App.Cancel(),
    summarize: (results) =>
      App.Summarize(results as unknown as Parameters<typeof App.Summarize>[0]) as unknown as Promise<Summary>,
    saveReport: (name, content) => App.SaveReport(name, content),
    copy: (text) => runtime.ClipboardSetText(text),
    quit: () => App.Quit(),
    onFinding: (cb) => runtime.EventsOn('finding', (e: FindingEvent) => cb(e)),
  }
}

// a production build opened outside the app: every call fails with a clear message
export function missingBackend(): Backend {
  const fail = () => Promise.reject(new Error('Wails runtime is not available'))
  return {
    kind: 'none',
    info: fail,
    run: fail,
    cancel: () => Promise.resolve(),
    summarize: fail,
    saveReport: fail,
    copy: () => Promise.resolve(false),
    quit: () => Promise.resolve(),
    onFinding: () => () => {},
  }
}

export async function connect(): Promise<Backend> {
  if (hasWails()) return wailsBackend()
  // import.meta.env.DEV is false in `vite build`, so the demo is not even in the bundle Wails embeds
  if (import.meta.env.DEV) {
    const demo = await import('./demo')
    return demo.demoBackend(demo.scenarioFromUrl(globalThis.location?.search ?? ''))
  }
  return missingBackend()
}

export function errorText(e: unknown): string {
  if (e instanceof Error) return e.message
  if (typeof e === 'string') return e
  try {
    return JSON.stringify(e)
  } catch {
    return String(e)
  }
}
