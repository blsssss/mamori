import { formatTime } from '../time'
import type { Lang, Params, Status } from '../types'
import { en } from './en'
import { ru } from './ru'

export type UiKey = keyof typeof ru.ui

export interface Dict {
  decimal: string
  status: Record<Status, string>
  ui: Record<UiKey, string>
  values: Record<string, Record<string, string>>
  codes: Record<string, string>
  lines: Record<string, string>
}

export const DICTS: Record<Lang, Dict> = { ru, en }

// how a code reads: 'line' is Reimu's dialogue, 'text' the impersonal wording of the reports
export type Style = 'line' | 'text'

const PLACEHOLDER = /\{(\w+)\}/g
const OPTIONAL = /\[\[(.*?)\]\]/g

export function render(
  template: string,
  params: Params,
  value: (key: string, raw: string) => string,
): string {
  const kept = template.replace(OPTIONAL, (_, segment: string) => {
    const keys = [...segment.matchAll(PLACEHOLDER)].map((m) => m[1])
    return keys.every((k) => (params[k] ?? '') !== '') ? segment : ''
  })
  return kept.replace(PLACEHOLDER, (_, k: string) => (k in params ? value(k, params[k]) : '?'))
}

function paramValue(dict: Dict, code: string, key: string, raw: string): string {
  const map = dict.values[`${code}:${key}`] ?? dict.values[key]
  if (map) {
    if (raw in map) return map[raw]
    // lists such as "domain, private" are mapped item by item
    if (raw.includes(', ')) {
      return raw
        .split(', ')
        .map((item) => map[item] ?? item)
        .join(', ')
    }
  }
  if (key === 'seconds') return raw.replace('.', dict.decimal)
  // Security Center gives an English RFC 1123 date, such as "Thu, 09 Oct 2026 18:02:11 GMT"
  if (key === 'timestamp') return formatTime(raw)
  return raw
}

export function fallback(code: string, params: Params = {}): string {
  const entries = Object.entries(params)
  if (entries.length === 0) return code
  return `${code} {${entries.map(([k, v]) => `${k}: ${v}`).join(', ')}}`
}

export function hasCode(lang: Lang, code: string): boolean {
  return code in DICTS[lang].codes
}

export function codeText(lang: Lang, code: string, params: Params = {}, style: Style = 'line'): string {
  const dict = DICTS[lang]
  const template = (style === 'line' ? dict.lines[code] : undefined) ?? dict.codes[code]
  if (template === undefined) return fallback(code, params)
  return render(template, params, (k, v) => paramValue(dict, code, k, v))
}

export function ui(lang: Lang, key: UiKey, params: Record<string, string | number> = {}): string {
  const template = DICTS[lang].ui[key] ?? DICTS.ru.ui[key] ?? key
  const p: Params = {}
  for (const [k, v] of Object.entries(params)) p[k] = String(v)
  return render(template, p, (_, v) => v)
}

export function statusWord(lang: Lang, status: Status): string {
  return DICTS[lang].status[status]
}

export function decimal(lang: Lang, n: number, digits = 1): string {
  return n.toFixed(digits).replace('.', DICTS[lang].decimal)
}

export function duration(lang: Lang, ms: number): string {
  if (ms < 60_000) return ui(lang, 'time.seconds', { n: decimal(lang, ms / 1000) })
  const total = Math.round(ms / 1000)
  return ui(lang, 'time.minutes', { m: Math.floor(total / 60), s: String(total % 60).padStart(2, '0') })
}
