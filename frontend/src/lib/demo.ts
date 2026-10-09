// a deterministic stand-in for the Go side, loaded only by `npm run dev` in a plain browser (see
// connect in api.ts). It streams the same codes the real checks emit, never anything else
import type { Backend } from './api'
import type {
  AppInfo,
  CheckId,
  Finding,
  FindingEvent,
  Params,
  Result,
  RunOptions,
  Status,
  Summary,
} from './types'
import { CHECK_IDS } from './types'

export const SCENARIOS = ['protected', 'no-firewall', 'no-antivirus', 'offline', 'no-admin'] as const
export type Scenario = (typeof SCENARIOS)[number]

export function scenarioFromUrl(search: string): Scenario {
  const value = new URLSearchParams(search).get('scenario')
  return (SCENARIOS as readonly string[]).includes(value ?? '') ? (value as Scenario) : 'protected'
}

interface Step {
  // milliseconds after the previous step
  after: number
  finding: Finding
}

interface Script {
  steps: Step[]
  status: Status
  code: string
  data?: unknown
}

const f = (status: Status, code: string, params?: Params): Finding =>
  params ? { code, status, params } : { code, status }

const NCSI_URL = 'http://www.msftconnecttest.com/connecttest.txt'
const CONTROL = '77.88.8.8:443'
const RULE = 'mamori-probe-3f9a1c2e'
const WSAEACCES =
  'connectex: An attempt was made to access a socket in a way forbidden by its access permissions.'

const SYSTEM = {
  product: 'Windows 11 Pro',
  version: '24H2',
  build: '26100.4351',
  arch: 'amd64',
  host: 'DEMO-PC',
  user: 'DEMO-PC\\reimu',
}

function internet(s: Scenario): Script {
  if (s === 'offline') {
    const timeouts = ['1.1.1.1', '8.8.8.8', '77.88.8.8']
    return {
      steps: [
        {
          after: 180,
          finding: f('pass', 'net.adapter.ok', {
            name: 'Wi-Fi',
            ipv4: '192.168.0.105',
            gateway: '192.168.0.1',
          }),
        },
        {
          after: 420,
          finding: f('fail', 'net.dns.fail', {
            host: 'www.msftconnecttest.com',
            error: 'lookup www.msftconnecttest.com: no such host',
          }),
        },
        {
          after: 120,
          finding: f('fail', 'net.ncsi_dns.fail', {
            host: 'dns.msftncsi.com',
            error: 'lookup dns.msftncsi.com: no such host',
          }),
        },
        {
          after: 1900,
          finding: f('warn', 'net.icmp.fail', {
            target: timeouts.join(', '),
            error: timeouts.map((t) => `${t}: IP_REQ_TIMED_OUT (11010)`).join('; '),
          }),
        },
        {
          after: 140,
          finding: f('fail', 'net.tcp.fail', {
            target: timeouts.map((t) => `${t}:443`).join(', '),
            error: timeouts.map((t) => `${t}:443: i/o timeout`).join('; '),
          }),
        },
        {
          after: 90,
          finding: f('fail', 'net.http.fail', {
            url: NCSI_URL,
            error: 'dial tcp: lookup www.msftconnecttest.com: no such host',
          }),
        },
      ],
      status: 'fail',
      code: 'net.verdict.offline',
    }
  }
  return {
    steps: [
      {
        after: 200,
        finding: f('pass', 'net.adapter.ok', {
          name: 'Ethernet',
          ipv4: '192.168.1.24',
          gateway: '192.168.1.1',
        }),
      },
      {
        after: 320,
        finding: f('pass', 'net.dns.ok', { host: 'www.msftconnecttest.com', addrs: '13.107.4.52', ms: '18' }),
      },
      {
        after: 260,
        finding: f('pass', 'net.ncsi_dns.ok', { host: 'dns.msftncsi.com', addr: '131.107.255.255' }),
      },
      { after: 300, finding: f('pass', 'net.icmp.ok', { target: '77.88.8.8', rtt_ms: '9', ttl: '58' }) },
      { after: 220, finding: f('pass', 'net.tcp.ok', { target: CONTROL, ms: '11' }) },
      { after: 380, finding: f('pass', 'net.http.ok', { url: NCSI_URL, ms: '64' }) },
    ],
    status: 'pass',
    code: 'net.verdict.online',
  }
}

function inventory(s: Scenario): Script {
  const avOff = s === 'no-antivirus'
  const fwOff = s === 'no-firewall'
  const profiles = [
    { name: 'domain', active: false },
    { name: 'private', active: true },
    { name: 'public', active: false },
  ].map((p) => ({ ...p, enabled: !fwOff, inbound: 'block', outbound: 'allow' }))
  const profileStatus = (p: (typeof profiles)[number]): Status =>
    p.enabled ? 'pass' : p.active ? 'fail' : 'warn'
  const av = {
    name: 'Windows Defender',
    kind: 'antivirus',
    state: avOff ? 'off' : 'on',
    signature: 'up_to_date',
    path: 'windowsdefender://',
    timestamp: 'Thu, 09 Oct 2026 18:02:11 GMT',
    source: 'wsc',
  }
  const defender = {
    service: 'running',
    mode: 'Normal',
    realtime: !avOff,
    antivirus: true,
    signatureAge: 0,
    signatureVersion: '1.437.112.0',
  }
  const files = [
    {
      vendor: 'Microsoft Defender',
      kind: 'av',
      evidence: 'path',
      value: 'C:\\Program Files\\Windows Defender',
    },
    { vendor: 'Microsoft Defender', kind: 'av', evidence: 'process', value: 'MsMpEng.exe' },
    { vendor: 'Microsoft Defender', kind: 'av', evidence: 'service', value: 'WinDefend' },
  ]
  const steps: Step[] = [
    {
      after: 650,
      finding: f(avOff ? 'fail' : 'pass', 'inv.av.product', {
        name: av.name,
        state: av.state,
        signature: av.signature,
        source: av.source,
        path: av.path,
        timestamp: av.timestamp,
      }),
    },
    { after: 420, finding: f('pass', 'inv.winfw.service', { state: 'running' }) },
    ...profiles.map((p, i) => ({
      after: i === 0 ? 260 : 60,
      finding: f(profileStatus(p), 'inv.winfw.profile', {
        profile: p.name,
        enabled: String(p.enabled),
        active: String(p.active),
        inbound: p.inbound,
        outbound: p.outbound,
      }),
    })),
    {
      after: 700,
      finding: f(avOff ? 'fail' : 'pass', 'inv.defender.status', {
        realtime: String(defender.realtime),
        antivirus: String(defender.antivirus),
        service: defender.service,
        mode: defender.mode,
        signature_age: String(defender.signatureAge),
        signature_version: defender.signatureVersion,
      }),
    },
    ...files.map((h, i) => ({ after: i === 0 ? 500 : 40, finding: f('pass', 'inv.files.found', { ...h }) })),
  ]
  const data = {
    source: 'wsc',
    antivirus: [av],
    firewalls: [],
    windowsFirewall: { service: 'running', profiles },
    defender,
    files,
  }
  if (avOff) return { steps, status: 'fail', code: 'inv.verdict.no_av', data }
  if (fwOff) return { steps, status: 'fail', code: 'inv.verdict.no_fw', data }
  return { steps, status: 'pass', code: 'inv.verdict.ok', data }
}

function firewall(s: Scenario, opts: RunOptions): Script {
  const fwOff = s === 'no-firewall'
  const steps: Step[] = [
    {
      after: 500,
      finding: fwOff
        ? f('fail', 'fw.state.off', { profile: 'private' })
        : f('pass', 'fw.state.enforcing', { profiles: 'private' }),
    },
  ]
  const targets = opts.policyTargets.map((t) => t.trim()).filter(Boolean)
  if (targets.length === 0) {
    steps.push({ after: 120, finding: f('skip', 'fw.policy.none') })
  }
  for (const [i, target] of targets.entries()) {
    const after = i === 0 ? 900 : 30
    if (fwOff) {
      steps.push({ after, finding: f('fail', 'fw.policy.reachable', { target, ms: '41' }) })
    } else if (s === 'offline') {
      steps.push({
        after,
        finding: f('warn', 'fw.policy.inconclusive', {
          target,
          error: `dial tcp: lookup ${hostOf(target)}: no such host`,
        }),
      })
    } else {
      steps.push({
        after,
        finding: f('pass', 'fw.policy.blocked', {
          target,
          error: `dial tcp ${hostOf(target)}:443: ${WSAEACCES}`,
          reason: 'wsaeacces',
        }),
      })
    }
  }
  if (!opts.allowElevation) {
    steps.push({ after: 150, finding: f('skip', 'fw.rule.needs_admin') })
  } else if (s === 'no-admin') {
    steps.push({ after: 2600, finding: f('skip', 'fw.rule.uac_declined') })
  } else if (s === 'offline') {
    const control = ['1.1.1.1:443', '8.8.8.8:443', '77.88.8.8:443']
    steps.push({
      after: 2600,
      finding: f('skip', 'fw.rule.control_failed', {
        target: control.join(', '),
        error: control.map((t) => `dial tcp ${t}: i/o timeout`).join('; '),
      }),
    })
  } else {
    // the elevated copy reports all its findings at once, after the UAC prompt
    steps.push({ after: 2600, finding: f('pass', 'fw.rule.added', { name: RULE, target: CONTROL }) })
    if (fwOff) {
      steps.push({ after: 0, finding: f('fail', 'fw.rule.not_enforced', { target: CONTROL }) })
      steps.push({ after: 0, finding: f('pass', 'fw.rule.removed', { name: RULE }) })
    } else {
      steps.push({
        after: 0,
        finding: f('pass', 'fw.rule.blocked', {
          target: CONTROL,
          error: `dial tcp ${CONTROL}: ${WSAEACCES}`,
          ms: '2',
        }),
      })
      steps.push({ after: 0, finding: f('pass', 'fw.rule.removed', { name: RULE }) })
      steps.push({ after: 0, finding: f('pass', 'fw.rule.restored', { target: CONTROL }) })
    }
  }
  const all = steps.map((x) => x.finding)
  const has = (code: string) => all.some((x) => x.code === code)
  if (has('fw.rule.not_enforced') || has('fw.policy.reachable')) {
    return { steps, status: 'fail', code: 'fw.verdict.not_enforced' }
  }
  if ((has('fw.rule.blocked') && has('fw.rule.restored')) || has('fw.policy.blocked')) {
    return { steps, status: 'pass', code: 'fw.verdict.works' }
  }
  if (fwOff) return { steps, status: 'fail', code: 'fw.verdict.disabled' }
  return { steps, status: 'warn', code: 'fw.verdict.unverified' }
}

function hostOf(target: string): string {
  try {
    return target.includes('://') ? new URL(target).hostname : (target.split(':')[0] ?? target)
  } catch {
    return target
  }
}

function antivirus(s: Scenario, opts: RunOptions): Script {
  if (s === 'no-antivirus') {
    const wait = Math.max(5, opts.eicarWaitSec)
    return {
      steps: [
        // the demo waits a few seconds and reports the configured wait, as the real check would
        { after: 3500, finding: f('fail', 'av.eicar.not_detected', { seconds: wait.toFixed(1) }) },
        { after: 160, finding: f('fail', 'av.amsi.not_detected', { result: '1' }) },
      ],
      status: 'fail',
      code: 'av.verdict.not_working',
    }
  }
  return {
    steps: [
      { after: 1300, finding: f('pass', 'av.eicar.removed', { seconds: '1.3' }) },
      { after: 420, finding: f('pass', 'av.amsi.detected', { result: '32768' }) },
    ],
    status: 'pass',
    code: 'av.verdict.works',
  }
}

export function script(id: CheckId, s: Scenario, opts: RunOptions): Script {
  switch (id) {
    case 'internet':
      return internet(s)
    case 'inventory':
      return inventory(s)
    case 'firewall':
      return firewall(s, opts)
    case 'antivirus':
      return antivirus(s, opts)
  }
}

// the verdict of internal/report, so that the demo log reads like a real one
export function summaryVerdict(results: Result[]): { status: Status; code: string } {
  const severity: Record<Status, number> = { pass: 0, skip: 1, warn: 2, fail: 3, error: 4 }
  let status: Status = 'pass'
  let ran = 0
  for (const r of results) {
    if (r.status === 'skip') continue
    ran++
    const s: Status = r.check === 'internet' && r.status === 'fail' ? 'warn' : r.status
    if (severity[s] > severity[status]) status = s
  }
  if (ran === 0) return { status: 'skip', code: 'report.none' }
  if (status === 'pass' && ran < results.length) return { status: 'warn', code: 'report.partial' }
  if (status === 'pass') return { status: 'pass', code: 'report.protected' }
  if (status === 'warn') return { status: 'warn', code: 'report.degraded' }
  if (status === 'error') return { status: 'error', code: 'report.error' }
  return { status: 'fail', code: 'report.unprotected' }
}

export function demoBackend(scenario: Scenario): Backend {
  const listeners = new Set<(e: FindingEvent) => void>()
  let busy = false
  let cancelled = false
  let wake: (() => void) | null = null

  const sleep = (ms: number) =>
    new Promise<void>((resolve) => {
      const t = setTimeout(done, ms)
      function done() {
        clearTimeout(t)
        wake = null
        resolve()
      }
      wake = done
    })

  const info: AppInfo = { version: 'demo', commit: '', elevated: false, system: { ...SYSTEM } }

  return {
    kind: 'demo',
    scenario,
    info: () => Promise.resolve(structuredClone(info)),
    async run(id, opts) {
      if (!CHECK_IDS.includes(id)) throw new Error(`unknown check "${id}"`)
      if (busy) throw new Error('another check is running')
      busy = true
      cancelled = false
      const start = Date.now()
      const sc = script(id, scenario, opts)
      const findings: Finding[] = []
      try {
        for (const step of sc.steps) {
          if (step.after > 0) await sleep(step.after)
          if (cancelled) break
          findings.push(step.finding)
          for (const l of listeners) l({ check: id, finding: structuredClone(step.finding) })
        }
        if (!cancelled) await sleep(150)
        const result: Result = {
          check: id,
          status: cancelled ? 'skip' : sc.status,
          code: cancelled ? 'common.cancelled' : sc.code,
          findings,
          started: new Date(start).toISOString(),
          elapsedMs: Date.now() - start,
        }
        if (!cancelled && sc.data !== undefined) result.data = sc.data
        return structuredClone(result)
      } finally {
        busy = false
      }
    },
    cancel() {
      if (busy) {
        cancelled = true
        wake?.()
      }
      return Promise.resolve()
    },
    summarize(results) {
      const v = summaryVerdict(results)
      const summary: Summary = {
        version: info.version,
        system: { ...SYSTEM },
        generated: new Date().toISOString(),
        status: v.status,
        code: v.code,
        results: structuredClone(results),
      }
      return Promise.resolve(summary)
    },
    saveReport(name, content) {
      // a browser cannot ask for a path, a download is the closest thing
      const type = name.endsWith('.html')
        ? 'text/html'
        : name.endsWith('.json')
          ? 'application/json'
          : 'text/plain'
      const url = URL.createObjectURL(new Blob([content], { type: `${type};charset=utf-8` }))
      const a = document.createElement('a')
      a.href = url
      a.download = name
      a.click()
      setTimeout(() => URL.revokeObjectURL(url), 1000)
      return Promise.resolve(name)
    },
    async copy(text) {
      try {
        await navigator.clipboard.writeText(text)
        return true
      } catch {
        return false
      }
    },
    quit: () => Promise.resolve(),
    onFinding(cb) {
      listeners.add(cb)
      return () => listeners.delete(cb)
    },
  }
}
