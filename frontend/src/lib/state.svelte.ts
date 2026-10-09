import { getContext } from 'svelte'
import { type Backend, errorText } from './api'
import type { CgId, Expression } from './assets'
import { meta } from './checks'
import { codeText, type Style, type UiKey, ui } from './i18n'
import { buildReport, buildText, checkTitle, normalizeSummary, reportName } from './report'
import {
  loadSettings,
  msPerChar,
  policyList,
  type Settings,
  sanitize,
  saveSettings,
  storage,
  writable,
} from './settings'
import {
  type AppInfo,
  CHECK_IDS,
  type CheckId,
  type CheckState,
  checkStatus,
  type FindingEvent,
  failedCall,
  idleCheck,
  type NoticeId,
  type ReportFormat,
  type Result,
  type RunOptions,
  type Summary,
} from './types'

export type Screen = 'title' | 'main' | 'log' | 'config'
export type EventCg = Exclude<CgId, 'title'>
export type Dialog = 'save' | 'quit' | null

export interface CgShow {
  id: EventCg
  // opened from a thumbnail: stays until closed, no timer
  manual: boolean
  seq: number
}

export interface Fx {
  kind: 'shake' | 'flash'
  seq: number
}

export interface Toast {
  text: string
  error: boolean
  seq: number
}

// how long Reimu's notice stays before the check starts, on top of the time to type it
const NOTICE_HOLD_MS = 3500
// AUTO lets the verdict of one check be read before the next one starts: the text box first has to
// get to it, then it stays this long
const AUTO_GAP_MS = 1500
// a text box that does not get to the verdict (the user paged back, slowest text speed) holds AUTO
// back at most this long
const AUTO_READ_CAP_MS = 15_000
// on quit Go still has to remove the temporary firewall rule and the EICAR folder of a stopped check
const QUIT_WAIT_MS = 10_000
const POLL_MS = 100

export class Store {
  readonly backend: Backend

  screen = $state<Screen>('title')
  returnTo = $state<Screen>('main')
  selected = $state<CheckId>('internet')
  checks = $state<Record<CheckId, CheckState>>({
    internet: idleCheck(),
    inventory: idleCheck(),
    firewall: idleCheck(),
    antivirus: idleCheck(),
  })
  settings = $state<Settings>(sanitize(null))
  readonly persistent: boolean
  info = $state<AppInfo | null>(null)
  infoError = $state<string | null>(null)
  summary = $state<Summary | null>(null)
  summaryVersion = $state(-1)
  summarizing = $state(false)
  // bumped whenever the set of results changes, a summary belongs to one version
  version = $state(0)
  auto = $state(false)
  cg = $state<CgShow | null>(null)
  fx = $state<Fx | null>(null)
  toast = $state<Toast | null>(null)
  announcement = $state('')
  now = $state(Date.now())
  reducedMotion = $state(false)
  dialog = $state<Dialog>(null)
  // quit was confirmed and waits for the stopped check to clean up
  quitting = $state(false)

  readonly lang = $derived(this.settings.lang)
  readonly running = $derived(
    CHECK_IDS.find((id) => this.checks[id].phase === 'running' || this.checks[id].phase === 'notice') ?? null,
  )
  readonly busy = $derived(this.running !== null || this.auto)
  readonly charMs = $derived(this.reducedMotion ? 0 : msPerChar(this.settings.textSpeed))
  readonly autoCg = $derived(this.settings.eventCgs && !this.reducedMotion)
  readonly results = $derived(
    CHECK_IDS.flatMap((id): Result[] => {
      const st = this.checks[id]
      if (st.result) return [st.result]
      if (st.phase === 'done' && st.error !== null) return [failedCall(id, st)]
      return []
    }),
  )
  readonly overall = $derived(this.summaryVersion === this.version ? this.summary : null)
  readonly expression = $derived.by((): Expression => {
    const st = this.checks[this.selected]
    if (st.phase === 'notice' || st.phase === 'running') return 'running'
    return checkStatus(st) ?? 'idle'
  })
  readonly unlocked = $derived({
    barrier: this.checks.firewall.result?.status === 'pass',
    exorcism: this.checks.antivirus.result?.status === 'pass',
    tea: this.overall?.code === 'report.protected',
  })

  private seq = 0
  private cgQueue: CgShow[] = []
  private lastOverall: string | null = null
  private pending: { version: number; promise: Promise<Summary | null> } | null = null
  private stopRequested = false
  private release: (() => void) | null = null
  // the Run call in flight, quit waits for it
  private inflight: Promise<unknown> | null = null
  // whether a text box is on screen, and the checks whose verdict it has got to since they ran
  private readers = 0
  private read = new Set<CheckId>()
  // event CGs a check has earned while the text box is still typing up to its verdict; a CG that
  // plays before the verdict is read gives the result away and hides the findings behind it
  private earned = new Map<CheckId, EventCg[]>()
  private ticker: ReturnType<typeof setInterval> | null = null
  private unsubscribe: (() => void) | null = null

  private readonly storage: Storage | null

  constructor(backend: Backend, store: Storage | null = storage()) {
    this.backend = backend
    this.storage = store
    this.settings = loadSettings(store)
    this.persistent = writable(store)
  }

  // reads every setting, so an effect that calls it saves again on any change
  persist(): void {
    saveSettings($state.snapshot(this.settings) as Settings, this.storage)
  }

  t = (key: UiKey, params?: Record<string, string | number>): string => ui(this.lang, key, params)

  code = (code: string, params?: Record<string, string>, style: Style = 'line'): string =>
    codeText(this.lang, code, params, style)

  async init(): Promise<void> {
    this.unsubscribe = this.backend.onFinding((e) => this.onFinding(e))
    if (this.settings.skipTitle) this.screen = 'main'
    try {
      this.info = await this.backend.info()
    } catch (e) {
      this.infoError = errorText(e)
    }
  }

  destroy(): void {
    this.unsubscribe?.()
    this.unsubscribe = null
    if (this.ticker) clearInterval(this.ticker)
    this.ticker = null
  }

  go(screen: Screen): void {
    if ((screen === 'log' || screen === 'config') && (this.screen === 'title' || this.screen === 'main')) {
      this.returnTo = this.screen
    }
    this.screen = screen
    if (screen === 'log') void this.summarize()
  }

  back(): void {
    this.screen = this.returnTo
  }

  select(id: CheckId): void {
    this.selected = id
  }

  runOptions(): RunOptions {
    return {
      policyTargets: policyList(this.settings.policyTargets),
      eicarWaitSec: this.settings.eicarWaitSec,
      allowElevation: this.settings.allowElevation,
    }
  }

  async run(id: CheckId): Promise<void> {
    if (this.busy) {
      this.notify(this.t('run.busy'))
      return
    }
    this.stopRequested = false
    await this.runOne(id)
  }

  async runAll(): Promise<void> {
    if (this.busy) return
    this.stopRequested = false
    this.auto = true
    try {
      for (const [i, id] of CHECK_IDS.entries()) {
        if (i > 0 && !this.stopRequested) {
          const prev = CHECK_IDS[i - 1]
          await this.wait(AUTO_GAP_MS, () => this.read.has(prev) || !this.reading(), AUTO_READ_CAP_MS)
        }
        if (this.stopRequested) break
        await this.runOne(id)
      }
    } finally {
      this.auto = false
    }
  }

  async stop(): Promise<void> {
    if (!this.busy) return
    this.stopRequested = true
    this.release?.()
    try {
      await this.backend.cancel()
    } catch {
      // the check still ends on its own timeout
    }
  }

  // the user read the notice and moved on, there is no need to wait for the timer
  skipNotice(): void {
    if (this.running && this.checks[this.running].phase === 'notice') this.release?.()
  }

  elapsed(id: CheckId): number | null {
    const st = this.checks[id]
    if (st.result) return st.result.elapsedMs
    if (st.phase === 'running') return Math.max(0, this.now - st.startedAt)
    if (st.phase === 'done') return Math.max(0, st.endedAt - st.startedAt)
    return null
  }

  // a minimized window stops the typewriter, there is nobody to wait for then
  private reading(): boolean {
    return this.readers > 0 && globalThis.document?.hidden !== true
  }

  // a text box on screen registers itself, AUTO then waits for it to show each verdict
  attachReader(): () => void {
    this.readers++
    return () => {
      this.readers--
    }
  }

  // the text box has shown the verdict of a finished check
  verdictShown(id: CheckId): void {
    if (this.checks[id].phase !== 'done') return
    this.read.add(id)
    const held = this.earned.get(id) ?? []
    this.earned.delete(id)
    for (const cg of held) this.showCg(cg)
  }

  private async runOne(id: CheckId): Promise<void> {
    const previous = $state.snapshot(this.checks[id]) as CheckState
    this.read.delete(id)
    this.earned.delete(id)
    this.selected = id
    const notices: NoticeId[] = []
    if (id === 'antivirus') notices.push('eicar')
    if (id === 'firewall' && this.settings.allowElevation && this.info?.elevated !== true) notices.push('uac')
    this.checks[id] = {
      ...idleCheck(),
      phase: notices.includes('eicar') ? 'notice' : 'running',
      notices,
      startedAt: Date.now(),
    }
    // the old result is gone, so is any summary that counted it
    this.version++
    this.startTicker()
    if (this.checks[id].phase === 'notice') {
      await this.wait(this.t('say.notice.eicar').length * this.charMs + NOTICE_HOLD_MS)
      if (this.stopRequested) {
        this.checks[id] = previous
        this.version++
        this.stopTicker()
        return
      }
      this.checks[id].phase = 'running'
      this.checks[id].startedAt = Date.now()
    }
    let call: Promise<Result>
    try {
      call = this.backend.run(id, this.runOptions())
    } catch (e) {
      call = Promise.reject(e)
    }
    this.inflight = call
    try {
      const result = await call
      this.checks[id].result = result
      this.checks[id].live = result.findings ?? []
    } catch (e) {
      this.checks[id].error = errorText(e)
    } finally {
      if (this.inflight === call) this.inflight = null
    }
    this.checks[id].endedAt = Date.now()
    this.checks[id].phase = 'done'
    this.version++
    this.stopTicker()
    this.finished(id)
  }

  // a pause of ms that STOP or the user ends early. It counts once ready() holds, or after capMs
  // in any case; an event CG on screen hides the text box, so the count starts again after it
  private wait(ms: number, ready: () => boolean = () => true, capMs = 0): Promise<void> {
    return new Promise((resolve) => {
      const t0 = Date.now()
      let since: number | null = null
      const release = () => {
        clearInterval(timer)
        this.release = null
        resolve()
      }
      const tick = () => {
        const now = Date.now()
        if (this.cg || !(ready() || now - t0 >= capMs)) since = null
        else since ??= now
        if (since !== null && now - since >= ms) release()
      }
      const timer = setInterval(tick, POLL_MS)
      this.release = release
      tick()
    })
  }

  private finished(id: CheckId): void {
    const st = this.checks[id]
    const status = checkStatus(st)
    const title = checkTitle(this.lang, id)
    this.announcement = st.result
      ? `${title}. ${this.code(st.result.code, st.result.params)}`
      : `${title}. ${this.t('say.runError')} ${st.error ?? ''}`
    if (id === this.selected && !this.reducedMotion) {
      if (status === 'fail') this.fx = { kind: 'shake', seq: ++this.seq }
      if (status === 'error') this.fx = { kind: 'flash', seq: ++this.seq }
    }
    // a text box types out only the selected check, and instant text is out at once: nothing to wait for then
    if (id === this.selected && this.charMs > 0 && this.reading()) this.earned.set(id, [])
    const cg = meta(id).cg
    if (cg && status === 'pass') this.earn(id, cg)
    if (this.results.length === CHECK_IDS.length) {
      void this.summarize().then((s) => {
        if (!s) return
        if (s.code === 'report.protected' && this.lastOverall !== s.code) this.earn(id, 'tea')
        this.lastOverall = s.code
      })
    }
  }

  // the summary can come back before or after the text box gets to the verdict, either way it waits
  private earn(id: CheckId, cg: EventCg): void {
    const held = this.earned.get(id)
    if (held) held.push(cg)
    else this.showCg(cg)
  }

  private onFinding(e: FindingEvent): void {
    const st = this.checks[e.check]
    if (st?.phase !== 'running') return
    st.live.push(e.finding)
    this.announcement = this.code(e.finding.code, e.finding.params)
  }

  private startTicker(): void {
    this.now = Date.now()
    this.ticker ??= setInterval(() => {
      this.now = Date.now()
    }, 100)
  }

  private stopTicker(): void {
    if (this.running === null && this.ticker) {
      clearInterval(this.ticker)
      this.ticker = null
    }
  }

  summarize(): Promise<Summary | null> {
    if (this.summary && this.summaryVersion === this.version) return Promise.resolve(this.summary)
    if (this.pending && this.pending.version === this.version) return this.pending.promise
    const version = this.version
    const promise = this.fetchSummary(version).finally(() => {
      if (this.pending?.version === version) this.pending = null
      this.summarizing = this.pending !== null
    })
    this.pending = { version, promise }
    this.summarizing = true
    return promise
  }

  private async fetchSummary(version: number): Promise<Summary | null> {
    try {
      const s = normalizeSummary(await this.backend.summarize($state.snapshot(this.results) as Result[]))
      if (version === this.version) {
        this.summary = s
        this.summaryVersion = version
      }
      return s
    } catch (e) {
      this.notify(this.t('summary.failed', { error: errorText(e) }), true)
      return null
    }
  }

  clearResults(): void {
    if (this.busy) return
    for (const id of CHECK_IDS) this.checks[id] = idleCheck()
    this.earned.clear()
    this.summary = null
    this.summaryVersion = -1
    this.lastOverall = null
    this.version++
    this.notify(this.t('clear.done'))
  }

  showCg(id: EventCg, manual = false): void {
    if (!manual && !this.autoCg) return
    const show = { id, manual, seq: ++this.seq }
    if (this.cg) this.cgQueue.push(show)
    else this.cg = show
  }

  closeCg(): void {
    this.cg = this.cgQueue.shift() ?? null
  }

  notify(text: string, error = false): void {
    this.toast = { text, error, seq: ++this.seq }
  }

  async saveReport(format: ReportFormat): Promise<void> {
    this.settings.reportFormat = format
    this.dialog = null
    const s = await this.summarize()
    if (!s) return
    const content = buildReport(format, s, this.lang, { demo: this.backend.kind === 'demo' })
    try {
      const path = await this.backend.saveReport(reportName(format), content)
      if (path) this.notify(this.t('save.done', { path }))
    } catch (e) {
      this.notify(this.t('save.failed', { error: errorText(e) }), true)
    }
  }

  async copyReport(): Promise<void> {
    const s = await this.summarize()
    if (!s) return
    let ok = false
    try {
      ok = await this.backend.copy(buildText(s, this.lang, { demo: this.backend.kind === 'demo' }))
    } catch {
      ok = false
    }
    this.notify(this.t(ok ? 'copy.done' : 'copy.failed'), !ok)
  }

  requestQuit(): void {
    if (this.busy) this.dialog = 'quit'
    else void this.quit()
  }

  async quit(): Promise<void> {
    if (this.quitting) return
    if (this.busy) {
      this.quitting = true
      await this.stop()
      // the cancelled check removes its firewall rule and EICAR folder before Run returns
      const call = this.inflight
      if (call) await Promise.race([call.catch(() => {}), delay(QUIT_WAIT_MS)])
    }
    this.dialog = null
    if (this.backend.kind === 'demo') {
      this.quitting = false
      this.notify(this.t('quit.demo'))
      return
    }
    await this.backend.quit()
  }
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

const KEY = Symbol('mamori.store')

// handed to mount() as its context, so every component finds the store with useStore
export function storeContext(store: Store): Map<symbol, Store> {
  return new Map([[KEY, store]])
}

export function useStore(): Store {
  return getContext<Store>(KEY)
}
