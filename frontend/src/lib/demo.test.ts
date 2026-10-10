import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { goCodes } from '../test/gocodes'
import { demoBackend, SCENARIOS, scenarioFromUrl, script, summaryVerdict } from './demo'
import { CHECK_IDS, type FindingEvent, type RunOptions } from './types'

const OPTS: RunOptions[] = [
  { policyTargets: [], eicarWaitSec: 15, allowElevation: true },
  {
    policyTargets: ['forbidden.example', 'https://blocked.example/'],
    eicarWaitSec: 20,
    allowElevation: false,
  },
]

describe('demo scripts', () => {
  const codes = new Set(goCodes())

  it('use only codes the Go checks emit', () => {
    for (const s of SCENARIOS) {
      for (const id of CHECK_IDS) {
        for (const opts of OPTS) {
          const sc = script(id, s, opts)
          expect(codes.has(sc.code), `${s} ${id} ${sc.code}`).toBe(true)
          for (const step of sc.steps)
            expect(codes.has(step.finding.code), `${s} ${id} ${step.finding.code}`).toBe(true)
        }
      }
    }
  })

  it('tell the scenarios apart', () => {
    const opts = OPTS[0]
    expect(script('firewall', 'protected', opts).code).toBe('fw.verdict.works')
    expect(script('firewall', 'no-firewall', opts).code).toBe('fw.verdict.not_enforced')
    expect(script('antivirus', 'no-antivirus', opts).code).toBe('av.verdict.not_working')
    expect(script('internet', 'offline', opts).code).toBe('net.verdict.offline')
    expect(script('firewall', 'no-admin', opts).code).toBe('fw.verdict.unverified')
    expect(script('firewall', 'protected', OPTS[1]).steps.map((s) => s.finding.code)).toContain(
      'fw.rule.needs_admin',
    )
  })
})

describe('scenarioFromUrl', () => {
  it('reads ?scenario= and defaults to protected', () => {
    expect(scenarioFromUrl('?scenario=offline')).toBe('offline')
    expect(scenarioFromUrl('?scenario=evil')).toBe('protected')
    expect(scenarioFromUrl('')).toBe('protected')
  })
})

describe('summaryVerdict', () => {
  const r = (check: (typeof CHECK_IDS)[number], status: 'pass' | 'fail' | 'skip' | 'warn' | 'error') => ({
    check,
    status,
    code: '',
    findings: [],
    started: '',
    elapsedMs: 0,
  })

  it('mirrors internal/report', () => {
    expect(summaryVerdict([]).code).toBe('report.none')
    expect(summaryVerdict(CHECK_IDS.map((id) => r(id, 'pass'))).code).toBe('report.protected')
    expect(summaryVerdict([r('internet', 'fail'), r('firewall', 'pass')]).code).toBe('report.degraded')
    expect(summaryVerdict([r('internet', 'skip'), r('firewall', 'pass')]).code).toBe('report.partial')
    expect(summaryVerdict([r('firewall', 'fail')]).code).toBe('report.unprotected')
    expect(summaryVerdict([r('firewall', 'error')]).code).toBe('report.error')
  })
})

describe('demoBackend', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('streams findings and resolves with all of them', async () => {
    const b = demoBackend('protected')
    const seen: FindingEvent[] = []
    b.onFinding((e) => seen.push(e))
    const run = b.run('internet', OPTS[0])
    await vi.runAllTimersAsync()
    const res = await run
    expect(res.code).toBe('net.verdict.online')
    expect(seen.map((e) => e.finding)).toEqual(res.findings)
    expect(seen.every((e) => e.check === 'internet')).toBe(true)
    expect((await b.info()).version).toBe('demo')
  })

  it('refuses a second run and stops on cancel', async () => {
    const b = demoBackend('protected')
    const run = b.run('inventory', OPTS[0])
    await expect(b.run('internet', OPTS[0])).rejects.toThrow('another check is running')
    await vi.advanceTimersByTimeAsync(700)
    await b.cancel()
    await vi.runAllTimersAsync()
    const res = await run
    expect(res.code).toBe('common.cancelled')
    expect(res.status).toBe('skip')
    expect(res.findings.length).toBeGreaterThan(0)
  })
})
