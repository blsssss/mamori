import { flushSync, mount, unmount } from 'svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { FakeBackend, memoryStorage } from '../../test/fixtures'
import { Store, storeContext } from '../state.svelte'
import EventCg from './EventCg.svelte'

let target: HTMLElement
let cg: ReturnType<typeof mount>
let store: Store

const shown = () => target.querySelector<HTMLButtonElement>('button.cg')

// jsdom runs no CSS animations, the end of the fade out is sent by hand
function fadeOut(): void {
  const el = shown()
  el?.dispatchEvent(Object.assign(new Event('animationend'), { animationName: 'svelte-x-cg-out' }))
  flushSync()
}

beforeEach(() => {
  vi.useFakeTimers()
  store = new Store(new FakeBackend(), memoryStorage())
  target = document.createElement('div')
  document.body.append(target)
  cg = mount(EventCg, { target, context: storeContext(store) })
  flushSync()
})

afterEach(() => {
  unmount(cg)
  target.remove()
  store.destroy()
  vi.useRealTimers()
})

describe('an event CG', () => {
  it('closes by itself after the hold time when it came after a check', () => {
    store.showCg('barrier')
    flushSync()
    expect(shown()).not.toBeNull()
    expect(target.querySelector('[role="status"]')?.textContent).toContain('Барьер держит')
    vi.advanceTimersByTime(2500)
    flushSync()
    expect(shown()?.classList.contains('closing')).toBe(false)
    vi.advanceTimersByTime(200)
    flushSync()
    expect(shown()?.classList.contains('closing')).toBe(true)
    fadeOut()
    expect(store.cg).toBeNull()
    expect(shown()).toBeNull()
  })

  it('opened from a thumbnail has no timer, takes the focus and gives it back', () => {
    const thumb = document.createElement('button')
    document.body.append(thumb)
    thumb.focus()
    store.showCg('exorcism', true)
    flushSync()
    expect(document.activeElement).toBe(shown())
    vi.advanceTimersByTime(60_000)
    flushSync()
    expect(store.cg?.id).toBe('exorcism')
    expect(shown()?.classList.contains('closing')).toBe(false)
    shown()?.click()
    flushSync()
    fadeOut()
    expect(store.cg).toBeNull()
    expect(document.activeElement).toBe(thumb)
    thumb.remove()
  })
})
