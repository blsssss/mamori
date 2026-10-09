import type { Backend } from '../lib/api'
import type {
  AppInfo,
  CheckId,
  Finding,
  FindingEvent,
  Result,
  RunOptions,
  Status,
  Summary,
} from '../lib/types'

export const SYSTEM = {
  product: 'Windows 11 Pro',
  version: '24H2',
  build: '26100.4351',
  arch: 'amd64',
  host: 'PC-01',
  user: 'PC-01\\student',
}

export function result(check: CheckId, status: Status, code: string, findings: Finding[] = []): Result {
  return { check, status, code, findings, started: '2026-10-09T21:55:03.123456789+03:00', elapsedMs: 1250 }
}

export function summary(results: Result[], status: Status, code: string): Summary {
  return {
    version: '1.0.0',
    commit: 'abc1234',
    system: { ...SYSTEM },
    generated: '2026-10-09T21:56:10.5+03:00',
    status,
    code,
    results,
  }
}

export const FULL: Result[] = [
  result('internet', 'pass', 'net.verdict.online', [
    {
      code: 'net.dns.ok',
      status: 'pass',
      params: { host: 'www.msftconnecttest.com', addrs: '13.107.4.52', ms: '18' },
    },
  ]),
  result('inventory', 'pass', 'inv.verdict.ok', [
    {
      code: 'inv.winfw.profile',
      status: 'pass',
      params: { profile: 'private', enabled: 'true', active: 'true', inbound: 'block', outbound: 'allow' },
    },
  ]),
  result('firewall', 'fail', 'fw.verdict.not_enforced', [
    {
      code: 'fw.policy.reachable',
      status: 'fail',
      params: { target: 'https://forbidden.example/', ms: '41' },
    },
    {
      code: 'fw.rule.add_failed',
      status: 'error',
      params: { error: 'Access is denied. <script>alert(1)</script>' },
    },
  ]),
  result('antivirus', 'pass', 'av.verdict.works', [
    { code: 'av.eicar.removed', status: 'pass', params: { seconds: '1.3' } },
  ]),
]

interface PendingRun {
  id: CheckId
  opts: RunOptions
  resolve: (r: Result) => void
  reject: (e: unknown) => void
}

// a backend whose runs end when the test says so, with findings the test pushes
export class FakeBackend implements Backend {
  readonly kind = 'wails' as const
  listeners = new Set<(e: FindingEvent) => void>()
  runs: PendingRun[] = []
  cancelled = 0
  summarized: Result[][] = []
  saved: { name: string; content: string }[] = []
  copied: string[] = []
  quits = 0
  elevated = false

  info(): Promise<AppInfo> {
    return Promise.resolve({
      version: '1.0.0',
      commit: 'abc1234',
      elevated: this.elevated,
      system: { ...SYSTEM },
    })
  }

  run(id: CheckId, opts: RunOptions): Promise<Result> {
    return new Promise((resolve, reject) => {
      this.runs.push({ id, opts, resolve, reject })
    })
  }

  emit(check: CheckId, finding: Finding): void {
    for (const l of this.listeners) l({ check, finding })
  }

  cancel(): Promise<void> {
    this.cancelled++
    return Promise.resolve()
  }

  summarize(results: Result[]): Promise<Summary> {
    this.summarized.push(results)
    const bad = results.some((r) => r.status !== 'pass')
    return Promise.resolve(
      bad ? summary(results, 'fail', 'report.unprotected') : summary(results, 'pass', 'report.protected'),
    )
  }

  saveReport(name: string, content: string): Promise<string> {
    this.saved.push({ name, content })
    return Promise.resolve(`C:\\Users\\student\\Documents\\${name}`)
  }

  copy(text: string): Promise<boolean> {
    this.copied.push(text)
    return Promise.resolve(true)
  }

  quit(): Promise<void> {
    this.quits++
    return Promise.resolve()
  }

  onFinding(cb: (e: FindingEvent) => void): () => void {
    this.listeners.add(cb)
    return () => this.listeners.delete(cb)
  }
}

// a Storage whose every call throws, as some locked-down webviews do
export const brokenStorage = {
  getItem() {
    throw new Error('SecurityError')
  },
  setItem() {
    throw new Error('SecurityError')
  },
} as unknown as Storage

export function memoryStorage(): Storage {
  const m = new Map<string, string>()
  return {
    get length() {
      return m.size
    },
    clear: () => m.clear(),
    getItem: (k) => m.get(k) ?? null,
    key: (i) => [...m.keys()][i] ?? null,
    removeItem: (k) => {
      m.delete(k)
    },
    setItem: (k, v) => {
      m.set(k, String(v))
    },
  }
}

// lets queued promise callbacks run
export async function settle(): Promise<void> {
  for (let i = 0; i < 5; i++) await Promise.resolve()
}
