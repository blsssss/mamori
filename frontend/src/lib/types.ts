// mirrors of the Go types in internal/check, internal/report, internal/sysinfo and app.go, with
// the string unions the generated wailsjs models do not have

export type Status = 'pass' | 'skip' | 'warn' | 'fail' | 'error'

export type CheckId = 'internet' | 'inventory' | 'firewall' | 'antivirus'

export const CHECK_IDS: readonly CheckId[] = ['internet', 'inventory', 'firewall', 'antivirus']

export type Params = Record<string, string>

export interface Finding {
  code: string
  status: Status
  params?: Params
}

export interface Result {
  check: CheckId
  status: Status
  code: string
  params?: Params
  findings: Finding[]
  data?: unknown
  started: string
  elapsedMs: number
}

export interface SysInfo {
  product: string
  version: string
  build: string
  arch: string
  host: string
  user: string
}

export interface AppInfo {
  version: string
  commit: string
  elevated: boolean
  system: SysInfo
}

export interface RunOptions {
  policyTargets: string[]
  eicarWaitSec: number
  allowElevation: boolean
}

export interface Summary {
  version: string
  commit?: string
  system: SysInfo
  generated: string
  status: Status
  code: string
  results: Result[]
}

export interface FindingEvent {
  check: CheckId
  finding: Finding
}

export type Lang = 'ru' | 'en'

export type ReportFormat = 'txt' | 'html' | 'json'

export const STATUSES: readonly Status[] = ['pass', 'warn', 'fail', 'skip', 'error']

export function isCheckId(v: unknown): v is CheckId {
  return typeof v === 'string' && (CHECK_IDS as readonly string[]).includes(v)
}

// idle: never ran or cleared; notice: Reimu explains what is about to happen before the call
export type Phase = 'idle' | 'notice' | 'running' | 'done'

export type NoticeId = 'eicar' | 'uac'

export interface CheckState {
  phase: Phase
  // findings streamed while the check runs, replaced by the result's own list at the end
  live: Finding[]
  result: Result | null
  // the call itself failed, there is no result
  error: string | null
  startedAt: number
  endedAt: number
  notices: NoticeId[]
}

export function idleCheck(): CheckState {
  return { phase: 'idle', live: [], result: null, error: null, startedAt: 0, endedAt: 0, notices: [] }
}

// the verdict of a check whose Run call itself failed; it is not a Go code, the interface sets it
export const CALL_FAILED = 'app.call_failed'

// such a check has no result from Go, yet the summary and the reports must still list it as an error
export function failedCall(id: CheckId, s: CheckState): Result {
  return {
    check: id,
    status: 'error',
    code: CALL_FAILED,
    params: { error: s.error ?? '' },
    findings: [],
    started: new Date(s.startedAt).toISOString(),
    elapsedMs: Math.max(0, Math.round(s.endedAt - s.startedAt)),
  }
}

// the status a check shows: its result, an error for a failed call, nothing while it has not ended
export function checkStatus(s: CheckState): Status | null {
  if (s.result) return s.result.status
  if (s.error !== null) return 'error'
  return null
}
