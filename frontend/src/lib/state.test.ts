import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { brokenStorage, FakeBackend, memoryStorage, result, settle } from '../test/fixtures'
import { Store } from './state.svelte'
import { CHECK_IDS, type CheckId, checkStatus } from './types'

let backend: FakeBackend
let store: Store

async function fresh(storage: Storage | null = memoryStorage()): Promise<void> {
  backend = new FakeBackend()
  store = new Store(backend, storage)
  await store.init()
}

function finish(id: CheckId, status: 'pass' | 'fail' = 'pass'): void {
  const run = backend.runs.find((r) => r.id === id)
  if (!run) throw new Error(`${id} did not start`)
  run.resolve(result(id, status, `${id}.verdict`))
}

beforeEach(async () => {
  vi.useFakeTimers()
  await fresh()
})

afterEach(() => {
  store.destroy()
  vi.useRealTimers()
})

describe('a single check', () => {
  it('streams findings live and keeps the result', async () => {
    const done = store.run('internet')
    expect(store.running).toBe('internet')
    expect(store.expression).toBe('running')
    backend.emit('internet', {
      code: 'net.dns.ok',
      status: 'pass',
      params: { host: 'h', addrs: 'a', ms: '1' },
    })
    backend.emit('inventory', { code: 'inv.files.none', status: 'warn' })
    expect(store.checks.internet.live).toHaveLength(1)
    expect(store.checks.inventory.live).toHaveLength(0)
    expect(store.announcement).toContain('Проверила DNS')

    backend.runs[0].resolve(result('internet', 'warn', 'net.verdict.limited'))
    await done
    expect(store.checks.internet.phase).toBe('done')
    expect(checkStatus(store.checks.internet)).toBe('warn')
    expect(store.running).toBeNull()
    expect(store.expression).toBe('warn')
    expect(store.version).toBe(2)
  })

  it('passes the settings to Go', async () => {
    store.settings.policyTargets = 'a.example\n\n# note\nhttps://b.example/'
    store.settings.eicarWaitSec = 30
    store.settings.allowElevation = false
    void store.run('internet')
    expect(backend.runs[0].opts).toEqual({
      policyTargets: ['a.example', 'https://b.example/'],
      eicarWaitSec: 30,
      allowElevation: false,
    })
  })

  it('shows a failed call as an error', async () => {
    const done = store.run('inventory')
    backend.runs[0].reject('another check is running')
    await done
    expect(store.checks.inventory.error).toBe('another check is running')
    expect(checkStatus(store.checks.inventory)).toBe('error')
    expect(store.fx?.kind).toBe('flash')
  })

  it('shakes Reimu on a failed check, not in reduced motion', async () => {
    let done = store.run('internet')
    finish('internet', 'fail')
    await done
    expect(store.fx?.kind).toBe('shake')
    store.fx = null
    store.reducedMotion = true
    backend.runs = []
    done = store.run('internet')
    finish('internet', 'fail')
    await done
    expect(store.fx).toBeNull()
  })

  it('refuses to start while another check runs', async () => {
    void store.run('internet')
    await store.run('inventory')
    expect(backend.runs.map((r) => r.id)).toEqual(['internet'])
    expect(store.toast?.text).toContain('Дождитесь')
  })
})

describe('notices', () => {
  it('explain the EICAR file before the antivirus check starts', async () => {
    const done = store.run('antivirus')
    expect(store.checks.antivirus.phase).toBe('notice')
    expect(store.checks.antivirus.notices).toEqual(['eicar'])
    expect(backend.runs).toHaveLength(0)
    await vi.advanceTimersByTimeAsync(10_000)
    expect(store.checks.antivirus.phase).toBe('running')
    expect(backend.runs).toHaveLength(1)
    finish('antivirus')
    await done
  })

  it('end early when the user moves on', async () => {
    void store.run('antivirus')
    store.skipNotice()
    await settle()
    expect(backend.runs).toHaveLength(1)
  })

  it('give way to STOP, which restores the previous result', async () => {
    let done = store.run('antivirus')
    store.skipNotice()
    await settle()
    finish('antivirus')
    await done
    backend.runs = []
    done = store.run('antivirus')
    await store.stop()
    await done
    expect(backend.runs).toHaveLength(0)
    expect(store.checks.antivirus.phase).toBe('done')
    expect(store.checks.antivirus.result?.code).toBe('antivirus.verdict')
    expect(store.running).toBeNull()
  })

  it('warn about UAC only when it will appear', async () => {
    void store.run('firewall')
    expect(store.checks.firewall.notices).toEqual(['uac'])
    finish('firewall')
    await settle()
    if (store.info) store.info = { ...store.info, elevated: true }
    backend.runs = []
    void store.run('firewall')
    expect(store.checks.firewall.notices).toEqual([])
  })
})

describe('AUTO', () => {
  it('starts the next check once the text box has shown the verdict', async () => {
    const detach = store.attachReader()
    const all = store.runAll()
    finish('internet')
    await vi.advanceTimersByTimeAsync(4000)
    expect(store.selected).toBe('internet')
    store.verdictShown('internet')
    await vi.advanceTimersByTimeAsync(1300)
    expect(store.selected).toBe('internet')
    await vi.advanceTimersByTimeAsync(300)
    expect(store.selected).toBe('inventory')
    // a text box that never gets there holds AUTO back only so long
    finish('inventory')
    await vi.advanceTimersByTimeAsync(15_000)
    expect(store.selected).toBe('inventory')
    await vi.advanceTimersByTimeAsync(1600)
    expect(store.selected).toBe('firewall')
    await store.stop()
    backend.runs[2].resolve(result('firewall', 'skip', 'common.cancelled'))
    await all
    detach()
  })

  it('does not wait for a text box nobody can see', async () => {
    const detach = store.attachReader()
    const hidden = vi.spyOn(document, 'hidden', 'get').mockReturnValue(true)
    const all = store.runAll()
    finish('internet')
    await vi.advanceTimersByTimeAsync(1700)
    expect(store.selected).toBe('inventory')
    await store.stop()
    backend.runs[1].resolve(result('inventory', 'skip', 'common.cancelled'))
    await all
    hidden.mockRestore()
    detach()
  })

  it('runs the four checks in order and stops on STOP', async () => {
    const all = store.runAll()
    expect(store.busy).toBe(true)
    finish('internet')
    await vi.advanceTimersByTimeAsync(2000)
    expect(store.selected).toBe('inventory')
    finish('inventory')
    await vi.advanceTimersByTimeAsync(2000)
    expect(store.selected).toBe('firewall')
    await store.stop()
    expect(backend.cancelled).toBe(1)
    backend.runs[2].resolve(result('firewall', 'skip', 'common.cancelled'))
    await all
    expect(backend.runs.map((r) => r.id)).toEqual(['internet', 'inventory', 'firewall'])
    expect(store.auto).toBe(false)
    expect(store.checks.antivirus.phase).toBe('idle')
  })
})

describe('event CGs', () => {
  it('play after a passed firewall check and for a protected verdict', async () => {
    for (const id of CHECK_IDS) {
      const done = store.run(id)
      store.skipNotice()
      await settle()
      finish(id)
      await done
    }
    await settle()
    expect(store.cg?.id).toBe('barrier')
    store.closeCg()
    expect(store.cg?.id).toBe('exorcism')
    store.closeCg()
    expect(store.cg?.id).toBe('tea')
    store.closeCg()
    expect(store.cg).toBeNull()
    expect(store.unlocked).toEqual({ barrier: true, exorcism: true, tea: true })
  })

  it('wait until the text box has typed out the verdict that earned them', async () => {
    const detach = store.attachReader()
    for (const id of CHECK_IDS) {
      const done = store.run(id)
      store.skipNotice()
      await settle()
      finish(id)
      await done
      await settle()
      if (id === 'firewall') {
        expect(store.cg).toBeNull()
        store.verdictShown('firewall')
        expect(store.cg?.id).toBe('barrier')
        store.closeCg()
      }
    }
    // the summary is already back, the antivirus verdict is still being typed
    expect(store.overall?.code).toBe('report.protected')
    expect(store.cg).toBeNull()
    store.verdictShown('antivirus')
    expect(store.cg?.id).toBe('exorcism')
    store.closeCg()
    expect(store.cg?.id).toBe('tea')
    store.closeCg()
    store.verdictShown('antivirus')
    expect(store.cg).toBeNull()
    detach()
  })

  it('play at once for a check the text box is not showing', async () => {
    const detach = store.attachReader()
    const done = store.run('firewall')
    store.select('internet')
    finish('firewall')
    await done
    expect(store.cg?.id).toBe('barrier')
    detach()
  })

  it('stay off in reduced motion or when disabled, a thumbnail still opens them', async () => {
    store.reducedMotion = true
    const done = store.run('firewall')
    finish('firewall')
    await done
    expect(store.cg).toBeNull()
    store.reducedMotion = false
    store.settings.eventCgs = false
    store.showCg('barrier')
    expect(store.cg).toBeNull()
    store.showCg('barrier', true)
    expect(store.cg).toMatchObject({ id: 'barrier', manual: true })
  })
})

describe('results', () => {
  it('keep a failed call as an error, in the summary and in the reports', async () => {
    const done = store.run('inventory')
    await vi.advanceTimersByTimeAsync(1200)
    backend.runs[0].reject(new Error('Wails: call timed out'))
    await done
    expect(store.results).toEqual([
      expect.objectContaining({
        check: 'inventory',
        status: 'error',
        code: 'app.call_failed',
        params: { error: 'Wails: call timed out' },
        findings: [],
      }),
    ])
    expect(store.results[0].elapsedMs).toBeGreaterThanOrEqual(1200)
    expect(store.elapsed('inventory')).toBe(store.results[0].elapsedMs)
    await store.saveReport('txt')
    expect(backend.summarized[0].map((r) => r.check)).toEqual(['inventory'])
    const txt = backend.saved[0].content
    expect(txt).toContain('Итог: [ОШИБКА] Проверку выполнить не удалось: вызов программы завершился ошибкой.')
    expect(txt).toContain('Сообщение системы: Wails: call timed out')
  })

  it('are summed up once per version and marked partial when checks are missing', async () => {
    const done = store.run('internet')
    finish('internet')
    await done
    const [a, b] = await Promise.all([store.summarize(), store.summarize()])
    expect(a).toBe(b)
    expect(backend.summarized).toHaveLength(1)
    expect(a?.code).toBe('report.partial')
    expect(store.overall?.code).toBe('report.partial')
  })

  it('are cleared on request', async () => {
    const done = store.run('internet')
    finish('internet')
    await done
    store.clearResults()
    expect(store.results).toHaveLength(0)
    expect(store.checks.internet.phase).toBe('idle')
    expect(store.overall).toBeNull()
  })

  it('are saved and copied as reports', async () => {
    const done = store.run('internet')
    finish('internet')
    await done
    await store.saveReport('txt')
    expect(backend.saved[0].name).toMatch(/^mamori-report-\d{8}-\d{4}\.txt$/)
    expect(backend.saved[0].content).toContain('Отчёт о проверке средств защиты ПК')
    expect(store.toast?.text).toContain('Отчёт сохранён')
    expect(store.settings.reportFormat).toBe('txt')
    await store.copyReport()
    expect(backend.copied[0]).toContain('Модуль 1')
  })
})

describe('screens and settings', () => {
  it('return to where the log was opened from', () => {
    store.go('main')
    store.go('log')
    store.back()
    expect(store.screen).toBe('main')
    store.go('title')
    store.go('config')
    store.back()
    expect(store.screen).toBe('title')
  })

  it('skip the title screen when asked to', async () => {
    const storage = memoryStorage()
    storage.setItem('mamori.settings.v1', JSON.stringify({ skipTitle: true }))
    await fresh(storage)
    expect(store.screen).toBe('main')
  })

  it('work when storage throws', async () => {
    await fresh(brokenStorage)
    expect(store.persistent).toBe(false)
    store.settings.lang = 'en'
    expect(() => store.persist()).not.toThrow()
    expect(store.t('title.start')).toBe('Start')
  })

  it('quit after a stopped check has cleaned up', async () => {
    void store.run('firewall')
    store.requestQuit()
    expect(store.dialog).toBe('quit')
    const quit = store.quit()
    await settle()
    expect(backend.cancelled).toBe(1)
    expect(store.quitting).toBe(true)
    expect(store.dialog).toBe('quit')
    await vi.advanceTimersByTimeAsync(5000)
    expect(backend.quits).toBe(0)
    backend.runs[0].resolve(result('firewall', 'skip', 'common.cancelled'))
    await quit
    expect(backend.quits).toBe(1)
  })

  it('quit after 10 s even when the stopped check does not return', async () => {
    void store.run('internet')
    const quit = store.quit()
    await vi.advanceTimersByTimeAsync(9900)
    expect(backend.quits).toBe(0)
    await vi.advanceTimersByTimeAsync(200)
    await quit
    expect(backend.quits).toBe(1)
  })

  it('quit at once, or ask while a check runs', async () => {
    store.requestQuit()
    await settle()
    expect(backend.quits).toBe(1)
    void store.run('internet')
    store.requestQuit()
    expect(store.dialog).toBe('quit')
  })
})
