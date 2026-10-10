import { flushSync, mount, unmount } from 'svelte'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import App from './App.svelte'
import { Store, storeContext } from './lib/state.svelte'
import { FakeBackend, memoryStorage, result, settle } from './test/fixtures'

let target: HTMLElement
let app: ReturnType<typeof mount>
let backend: FakeBackend
let store: Store

async function start(): Promise<void> {
  backend = new FakeBackend()
  store = new Store(backend, memoryStorage())
  target = document.createElement('div')
  document.body.append(target)
  app = mount(App, { target, context: storeContext(store) })
  await settle()
  flushSync()
}

const $ = <T extends Element = HTMLElement>(sel: string) => target.querySelector<T & HTMLElement>(sel)
const $$ = (sel: string) => [...target.querySelectorAll<HTMLElement>(sel)]
// svelte sets inert as a property, which jsdom keeps without reflecting it to the attribute
const inert = (el: Element | null) =>
  el?.hasAttribute('inert') || (el as { inert?: boolean } | null)?.inert === true
const text = (el: Element | null) => el?.textContent?.replace(/\s+/g, ' ').trim() ?? ''

function key(k: string): void {
  const el = document.activeElement ?? document.body
  el.dispatchEvent(new KeyboardEvent('keydown', { key: k, bubbles: true, cancelable: true }))
  flushSync()
}

beforeEach(start)

afterEach(() => {
  unmount(app)
  target.remove()
  store.destroy()
})

describe('title screen', () => {
  it('offers the VN title menu and the version', () => {
    expect($$('.title nav .label').map(text)).toEqual(['Начать', 'Журнал', 'Настройки', 'Выход'])
    expect(text($('.title .subtitle'))).toBe('Проверка средств защиты ПК')
    expect(text($('.title .version'))).toBe('версия 1.0.0 (abc1234)')
    expect(document.activeElement?.textContent).toContain('Начать')
  })

  it('names every title menu button by its label, the kanji beside it is decoration', () => {
    expect($$('.title nav button').map((b) => b.getAttribute('aria-label'))).toEqual([
      'Начать',
      'Журнал',
      'Настройки',
      'Выход',
    ])
    expect(text($('.title h1'))).toBe('mamori')
  })

  it('leaves the focus alone when the scene is opened with the mouse', () => {
    $$('.title nav button')[0].click()
    flushSync()
    expect(document.activeElement).not.toBe($('.say'))
  })

  it('gets the focus back on the menu entry after the config closes, so the arrows work again', () => {
    const config = $$('.title nav button')[2]
    config.focus()
    config.click()
    flushSync()
    expect(document.activeElement?.closest('.config')).not.toBeNull()
    key('Escape')
    expect(document.activeElement).toBe(config)
    key('ArrowDown')
    expect(document.activeElement?.getAttribute('aria-label')).toBe('Выход')
  })

  it('switches the language', () => {
    $$('.title .lang button')[1].click()
    flushSync()
    expect($$('.title nav .label').map(text)).toEqual(['Start', 'Log', 'Settings', 'Quit'])
    expect(document.documentElement.lang).toBe('en')
  })
})

describe('main scene', () => {
  beforeEach(() => {
    $$('.title nav button')[0].click()
    flushSync()
  })

  it('shows the four checks, module 5, the text box and the quick menu', () => {
    expect($$('.chapter')).toHaveLength(4)
    expect(text($('.chapter .title'))).toBe('Подключение к интернету')
    expect(text($('.overall .title'))).toBe('Вывод результатов')
    expect(text($('.nameplate'))).toBe('霊夢·Рейму')
    expect($$('.qm button').map((b) => b.textContent?.trim())).toEqual([
      'Auto',
      'Stop',
      'Log',
      'Save',
      'Config',
      'Title',
      'Quit',
    ])
    for (const b of $$('.qm button')) {
      expect(b.getAttribute('aria-label')).toMatch(/^\w+\. [А-Яа-яЁё]/)
      expect(b.title).not.toBe('')
    }
    expect(text($('.box .live'))).toContain('Модуль 1, проверка подключения к интернету')
    expect(text($('.counter'))).toBe('1 / 2')
  })

  it('gives the text window and module 5 stable names', () => {
    const say = $('.say')
    expect(say?.getAttribute('aria-label')).toBe('Реплика Рейму')
    expect($('.arrow.right:not(.double)')?.getAttribute('aria-label')).toBe('Следующая реплика')
    expect(say?.getAttribute('aria-describedby')).toBe('box-line box-hint')
    expect(say?.querySelector('[aria-live]')).toBeNull()
    expect($('.overall')?.getAttribute('aria-label')).toBe(
      'Модуль 5. Вывод результатов: итог появится после проверок. Открыть журнал',
    )
  })

  it('never conveys a status by colour alone', async () => {
    $$('.chapter .run')[0].click()
    flushSync()
    expect(text($('.chapter .word'))).toBe('выполняется')
    backend.runs[0].resolve(result('internet', 'fail', 'net.verdict.offline'))
    await settle()
    flushSync()
    expect(text($('.chapter .word'))).toBe('не пройдено')
    expect(text($('.chapter .kanji'))).toBe('否')
    expect($('.chapter .pick')?.getAttribute('aria-label')).toBe(
      'Модуль 1. Подключение к интернету: не пройдено',
    )
  })

  it('names each run button starting with the word it shows', async () => {
    const run = $$('.chapter .run')[0]
    const starts = () => {
      const word = text(run).replace('…', '')
      expect(run.getAttribute('aria-label')?.startsWith(word), word).toBe(true)
      expect(run.getAttribute('aria-label')).toContain('Подключение к интернету')
    }
    starts()
    run.click()
    flushSync()
    expect(text(run)).toBe('Идёт…')
    starts()
    backend.runs[0].resolve(result('internet', 'pass', 'net.verdict.online'))
    await settle()
    flushSync()
    expect(text(run)).toBe('Повторить')
    starts()
  })

  it('makes the scene inert under an event CG opened from a thumbnail', () => {
    store.showCg('barrier', true)
    flushSync()
    expect(inert($('.app'))).toBe(true)
    key('Escape')
    expect(store.cg).toBeNull()
    expect(inert($('.app'))).toBe(false)
  })

  it('puts new findings in a polite live region', async () => {
    $$('.chapter .run')[0].click()
    flushSync()
    backend.emit('internet', {
      code: 'net.tcp.ok',
      status: 'pass',
      params: { target: '1.1.1.1:443', ms: '9' },
    })
    flushSync()
    const live = $$('[aria-live="polite"]').map(text)
    expect(live).toContain('Открыла TCP-соединение с 1.1.1.1:443 за 9 мс.')
  })

  it('opens the log and the config, Esc brings the scene back', () => {
    $$('.qm button')[2].click()
    flushSync()
    expect(text($('.log h1'))).toBe('Журнал')
    expect(text($('.log .heading p'))).toBe('Модуль 5. Вывод результатов')
    expect(inert($('.scene'))).toBe(true)
    key('Escape')
    expect($('.log')).toBeNull()
    expect(inert($('.scene'))).toBe(false)

    $$('.qm button')[4].click()
    flushSync()
    expect(text($('.config h1'))).toBe('Настройки')
    expect($('.config textarea')).not.toBeNull()
    key('Escape')
    expect($('.config')).toBeNull()
  })

  it('puts a keyboard user on the text box, so Enter goes on advancing it', () => {
    store.go('title')
    flushSync()
    key('Enter')
    $$('.title nav button')[0].click()
    flushSync()
    expect(document.activeElement).toBe($('.say'))
    key('Enter')
    key('Enter')
    expect(text($('.counter'))).toBe('2 / 2')
  })

  it('gives the focus back to the control that opened the log', () => {
    const log = $$('.qm button')[2]
    // a click focuses a button in WebView2, jsdom has to be told
    log.focus()
    log.click()
    flushSync()
    expect(document.activeElement?.closest('.log')).not.toBeNull()
    key('Escape')
    expect($('.log')).toBeNull()
    expect(document.activeElement).toBe(log)
  })

  it('falls back to the selected chapter when the opener is gone', () => {
    ;(document.activeElement as HTMLElement | null)?.blur()
    store.select('firewall')
    store.go('config')
    flushSync()
    key('Escape')
    expect(document.activeElement).toBe($('[data-check="firewall"]'))
  })

  it('advances the text with Enter when nothing is focused', () => {
    ;(document.activeElement as HTMLElement | null)?.blur()
    key('Enter')
    key('Enter')
    expect(text($('.counter'))).toBe('2 / 2')
    expect(text($('.box .live'))).toContain('AUTO')
  })
})
