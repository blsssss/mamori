import { afterEach, describe, expect, it, vi } from 'vitest'

const go = vi.hoisted(() => ({
  Info: vi.fn(() => Promise.resolve({ version: '1.0.0', commit: '', elevated: true, system: {} })),
  Run: vi.fn(() =>
    Promise.resolve({ check: 'internet', status: 'pass', code: 'net.verdict.online', findings: [] }),
  ),
  Cancel: vi.fn(() => Promise.resolve()),
  Summarize: vi.fn(() => Promise.resolve({ status: 'skip', code: 'report.none', results: [] })),
  SaveReport: vi.fn(() => Promise.resolve('C:\\r.html')),
  Quit: vi.fn(() => Promise.resolve()),
}))

const rt = vi.hoisted(() => ({
  EventsOn: vi.fn((_name: string, _cb: (e: unknown) => void) => () => {}),
  ClipboardSetText: vi.fn(() => Promise.resolve(true)),
}))

vi.mock('../../wailsjs/go/main/App', () => go)
vi.mock('../../wailsjs/runtime/runtime', () => rt)

import { connect, errorText, hasWails, missingBackend, wailsBackend } from './api'

afterEach(() => {
  delete (globalThis as { go?: unknown }).go
})

describe('backend selection', () => {
  it('detects the Wails bindings', () => {
    expect(hasWails({})).toBe(false)
    expect(hasWails({ go: { main: {} } })).toBe(false)
    expect(hasWails({ go: { main: { App: {} } } })).toBe(true)
  })

  it('uses Wails inside the app', async () => {
    ;(globalThis as { go?: unknown }).go = { main: { App: {} } }
    const b = await connect()
    expect(b.kind).toBe('wails')
  })

  it('falls back to the demo in development without Wails', async () => {
    const b = await connect()
    expect(b.kind).toBe('demo')
    expect(b.scenario).toBe('protected')
  })

  it('fails clearly outside the app in production', async () => {
    const b = missingBackend()
    await expect(b.info()).rejects.toThrow('Wails runtime is not available')
    expect(await b.copy('x')).toBe(false)
  })
})

describe('wailsBackend', () => {
  it('goes through the generated bindings', async () => {
    const b = wailsBackend()
    const opts = { policyTargets: ['a'], eicarWaitSec: 15, allowElevation: true }
    await b.run('internet', opts)
    expect(go.Run).toHaveBeenCalledWith('internet', opts)
    await b.summarize([])
    expect(go.Summarize).toHaveBeenCalledWith([])
    expect(await b.saveReport('r.html', '<html>')).toBe('C:\\r.html')
    expect(go.SaveReport).toHaveBeenCalledWith('r.html', '<html>')
    await b.cancel()
    await b.quit()
    expect(go.Cancel).toHaveBeenCalled()
    expect(go.Quit).toHaveBeenCalled()
    expect(await b.copy('text')).toBe(true)
    expect(rt.ClipboardSetText).toHaveBeenCalledWith('text')
  })

  it('subscribes to the "finding" event', () => {
    const cb = vi.fn()
    wailsBackend().onFinding(cb)
    expect(rt.EventsOn).toHaveBeenCalledWith('finding', expect.any(Function))
    const handler = rt.EventsOn.mock.calls[0][1]
    handler({ check: 'internet', finding: { code: 'net.dns.ok', status: 'pass' } })
    expect(cb).toHaveBeenCalledWith({ check: 'internet', finding: { code: 'net.dns.ok', status: 'pass' } })
  })
})

describe('errorText', () => {
  it('reads Go errors, which Wails rejects as strings', () => {
    expect(errorText('another check is running')).toBe('another check is running')
    expect(errorText(new Error('boom'))).toBe('boom')
    expect(errorText({ a: 1 })).toBe('{"a":1}')
  })
})
