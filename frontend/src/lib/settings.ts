import type { Lang, ReportFormat } from './types'

export interface Settings {
  lang: Lang
  // 0 prints a line at once, 10 is the slowest typewriter
  textSpeed: number
  eventCgs: boolean
  skipTitle: boolean
  // one host, host:port or URL per line, as typed
  policyTargets: string
  eicarWaitSec: number
  allowElevation: boolean
  reportFormat: ReportFormat
}

export const DEFAULTS: Readonly<Settings> = {
  lang: 'ru',
  textSpeed: 3,
  eventCgs: true,
  skipTitle: false,
  policyTargets: '',
  eicarWaitSec: 15,
  allowElevation: true,
  reportFormat: 'html',
}

export const TEXT_SPEED_MAX = 10
export const EICAR_WAIT_MIN = 5
export const EICAR_WAIT_MAX = 60

const KEY = 'mamori.settings.v1'

function clampInt(v: unknown, min: number, max: number, fallback: number): number {
  if (typeof v !== 'number' || !Number.isFinite(v)) return fallback
  return Math.min(max, Math.max(min, Math.round(v)))
}

export function sanitize(raw: unknown): Settings {
  const r = (typeof raw === 'object' && raw !== null ? raw : {}) as Partial<Record<keyof Settings, unknown>>
  return {
    lang: r.lang === 'en' ? 'en' : 'ru',
    textSpeed: clampInt(r.textSpeed, 0, TEXT_SPEED_MAX, DEFAULTS.textSpeed),
    eventCgs: typeof r.eventCgs === 'boolean' ? r.eventCgs : DEFAULTS.eventCgs,
    skipTitle: typeof r.skipTitle === 'boolean' ? r.skipTitle : DEFAULTS.skipTitle,
    policyTargets:
      typeof r.policyTargets === 'string' ? r.policyTargets.slice(0, 4000) : DEFAULTS.policyTargets,
    eicarWaitSec: clampInt(r.eicarWaitSec, EICAR_WAIT_MIN, EICAR_WAIT_MAX, DEFAULTS.eicarWaitSec),
    allowElevation: typeof r.allowElevation === 'boolean' ? r.allowElevation : DEFAULTS.allowElevation,
    reportFormat:
      r.reportFormat === 'txt' || r.reportFormat === 'json' || r.reportFormat === 'html'
        ? r.reportFormat
        : DEFAULTS.reportFormat,
  }
}

// localStorage itself throws in some webview and privacy configurations, not only its methods
export function storage(): Storage | null {
  try {
    return globalThis.localStorage ?? null
  } catch {
    return null
  }
}

// whether a write really sticks: some webviews hand out a storage object that throws on use
export function writable(store: Storage | null): boolean {
  if (!store) return false
  try {
    store.setItem('mamori.probe', '1')
    store.removeItem('mamori.probe')
    return true
  } catch {
    return false
  }
}

export function loadSettings(store: Storage | null = storage()): Settings {
  try {
    const raw = store?.getItem(KEY)
    return sanitize(raw ? JSON.parse(raw) : null)
  } catch {
    return sanitize(null)
  }
}

export function saveSettings(s: Settings, store: Storage | null = storage()): boolean {
  try {
    store?.setItem(KEY, JSON.stringify(s))
    return store !== null
  } catch {
    return false
  }
}

export function policyList(text: string): string[] {
  return text
    .split(/\r?\n/)
    .map((l) => l.trim())
    .filter((l) => l !== '' && !l.startsWith('#'))
}

// milliseconds per character of the typewriter
export function msPerChar(speed: number): number {
  return speed <= 0 ? 0 : speed * 7
}
