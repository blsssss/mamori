import { flushSync, mount, unmount } from 'svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { FakeBackend, memoryStorage, result } from '../../test/fixtures'
import { codeText, ui } from '../i18n'
import { Store, storeContext } from '../state.svelte'
import type { Finding } from '../types'
import TextBox from './TextBox.svelte'

let target: HTMLElement
let box: ReturnType<typeof mount>
let backend: FakeBackend
let store: Store

const text = (sel: string) => target.querySelector(sel)?.textContent?.replace(/\s+/g, ' ').trim() ?? ''

// time passes in small steps, so that the effects scheduled by one step run before the next
async function pass(ms: number): Promise<void> {
  for (let t = 0; t < ms; t += 50) {
    await vi.advanceTimersByTimeAsync(50)
    flushSync()
  }
}

const findings: Finding[] = [
  {
    code: 'net.adapter.ok',
    status: 'pass',
    params: { name: 'Ethernet', ipv4: '10.0.2.15', gateway: '10.0.2.2' },
  },
  {
    code: 'net.dns.ok',
    status: 'pass',
    params: { host: 'www.msftconnecttest.com', addrs: '13.107.4.52', ms: '18' },
  },
  { code: 'net.icmp.ok', status: 'pass', params: { target: '1.1.1.1', rtt_ms: '12', ttl: '57' } },
  { code: 'net.tcp.ok', status: 'pass', params: { target: '1.1.1.1:443', ms: '14' } },
  {
    code: 'net.http.ok',
    status: 'pass',
    params: { url: 'http://www.msftconnecttest.com/connecttest.txt', ms: '40' },
  },
]

beforeEach(async () => {
  vi.useFakeTimers()
  backend = new FakeBackend()
  store = new Store(backend, memoryStorage())
  await store.init()
  store.screen = 'main'
  target = document.createElement('div')
  document.body.append(target)
  box = mount(TextBox, { target, context: storeContext(store) })
  flushSync()
})

afterEach(() => {
  unmount(box)
  target.remove()
  store.destroy()
  vi.useRealTimers()
})

describe('the text box', () => {
  it('follows a burst of findings and ends on the verdict, typed out in full', async () => {
    const done = store.run('internet')
    flushSync()
    for (const f of findings) backend.emit('internet', f)
    backend.runs[0].resolve(result('internet', 'pass', 'net.verdict.online', findings))
    await done
    flushSync()
    await pass(12_000)
    expect(text('.counter')).toBe('7 / 7')
    expect(target.querySelector('.say')?.classList.contains('verdict')).toBe(true)
    expect(text('.say .text')).toBe(codeText('ru', 'net.verdict.online', {}, 'line'))
  })

  it('keeps a line finished by a click, the typewriter does not take it back', async () => {
    const intro = ui('ru', 'say.intro.internet')
    await pass(100)
    expect(text('.say .text').length).toBeLessThan(20)
    target.querySelector<HTMLElement>('.say')?.click()
    flushSync()
    expect(text('.say .text')).toBe(`${intro}▼`)
    await pass(300)
    expect(text('.say .text')).toBe(`${intro}▼`)
  })

  it('holds AUTO until the verdict is on screen, then a moment longer', async () => {
    const all = store.runAll()
    flushSync()
    for (const f of findings) backend.emit('internet', f)
    backend.runs[0].resolve(result('internet', 'pass', 'net.verdict.online', findings))
    const verdict = codeText('ru', 'net.verdict.online', {}, 'line')
    let shown = -1
    let t = 0
    for (; t < 20_000 && store.selected === 'internet'; t += 50) {
      if (shown < 0 && text('.say .text') === verdict) shown = t
      await pass(50)
    }
    // the fixed gap of AUTO alone would have moved on after 1.8 s, before the box got there
    expect(shown).toBeGreaterThan(1800)
    expect(store.selected).toBe('inventory')
    expect(t - shown).toBeGreaterThanOrEqual(1500)
    expect(t - shown).toBeLessThan(1800)
    await store.stop()
    backend.runs[1].resolve(result('inventory', 'skip', 'common.cancelled'))
    await all
  })

  it('pages with the wheel only when the line does not scroll that way', () => {
    store.settings.textSpeed = 0
    flushSync()
    const say = target.querySelector<HTMLElement>('.say')
    if (!say) throw new Error('no text area')
    Object.defineProperties(say, {
      scrollHeight: { configurable: true, value: 300 },
      clientHeight: { configurable: true, value: 100 },
      scrollTop: { configurable: true, writable: true, value: 0 },
    })
    say.dispatchEvent(new WheelEvent('wheel', { deltaY: 100 }))
    flushSync()
    expect(text('.counter')).toBe('1 / 2')
    vi.advanceTimersByTime(300)
    say.scrollTop = 200
    say.dispatchEvent(new WheelEvent('wheel', { deltaY: 100 }))
    flushSync()
    expect(text('.counter')).toBe('2 / 2')
  })
})
