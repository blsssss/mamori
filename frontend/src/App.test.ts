import { flushSync, mount, unmount } from 'svelte'
import { afterEach, expect, it, vi } from 'vitest'

vi.mock('../wailsjs/go/main/App', () => ({
  Info: vi.fn().mockResolvedValue({
    version: 'dev',
    commit: '',
    elevated: false,
    system: {
      product: 'Windows 11 Pro',
      version: '24H2',
      build: '26100',
      arch: 'amd64',
      host: 'pc',
      user: 'pc\\u',
    },
  }),
  Run: vi.fn().mockResolvedValue({
    check: 'internet',
    status: 'pass',
    code: 'net.verdict.online',
    findings: [{ code: 'net.http.ok', status: 'pass', params: { ms: '12' } }],
    started: '',
    elapsedMs: 12,
  }),
  Summarize: vi.fn(),
  SaveReport: vi.fn(),
  Quit: vi.fn(),
}))
vi.mock('../wailsjs/runtime/runtime', () => ({ EventsOn: vi.fn(() => () => {}) }))

const { default: App } = await import('./App.svelte')

let target: HTMLElement
let app: ReturnType<typeof mount>

afterEach(() => {
  unmount(app)
  target.remove()
})

it('shows the four checks and the output module', async () => {
  target = document.createElement('div')
  document.body.append(target)
  app = mount(App, { target })
  flushSync()
  const headings = [...target.querySelectorAll('h2')].map((h) => h.textContent)
  expect(headings).toHaveLength(5)
  expect(headings[4]).toContain('Вывод результатов')
})

it('runs a check and lists its findings', async () => {
  target = document.createElement('div')
  document.body.append(target)
  app = mount(App, { target })
  flushSync()
  const button = target.querySelectorAll('section button')[0] as HTMLButtonElement
  button.click()
  await vi.waitFor(() => expect(target.textContent).toContain('net.verdict.online'))
  expect(target.textContent).toContain('net.http.ok')
})
