// exported reports are strictly technical: impersonal texts, no art, no metaphors
import { meta } from './checks'
import { codeText, duration, statusWord, type UiKey, ui } from './i18n'
import { formatTime } from './time'
import type { CheckId, Finding, Lang, ReportFormat, Result, Status, Summary } from './types'
import { CHECK_IDS } from './types'

export interface ReportOptions {
  // set when the data comes from the demo backend, the report then says so on top
  demo?: boolean
}

export { formatTime, parseTime } from './time'

// Summarize in Go only sees the results it is given: a set without some of the four checks would
// read as "protected", so here it becomes "partial", as Go itself does for skipped checks. A fallback
// until verdict() in internal/report counts against check.All
export function normalizeSummary(s: Summary): Summary {
  const covered = new Set(s.results.map((r) => r.check))
  if (s.code === 'report.protected' && CHECK_IDS.some((id) => !covered.has(id))) {
    return { ...s, status: 'warn', code: 'report.partial' }
  }
  return s
}

const pad = (n: number) => String(n).padStart(2, '0')

export function reportName(format: ReportFormat, d: Date = new Date()): string {
  const stamp = `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}-${pad(d.getHours())}${pad(d.getMinutes())}`
  return `mamori-report-${stamp}.${format}`
}

export function checkTitle(lang: Lang, id: CheckId): string {
  return ui(lang, `check.${id}`)
}

export function systemLine(lang: Lang, s: Summary['system']): string {
  return ui(lang, 'top.system', { ...s })
}

function program(s: Summary): string {
  return `mamori ${s.version}${s.commit ? ` (${s.commit})` : ''}`
}

function ordered(s: Summary): { id: CheckId; result: Result | undefined }[] {
  return CHECK_IDS.map((id) => ({ id, result: s.results.find((r) => r.check === id) }))
}

function label(lang: Lang, status: Status): string {
  return statusWord(lang, status).toLocaleUpperCase(lang)
}

// the raw OS error of a finding, or of a verdict when the Run call itself failed
function rawError(f: Finding | Result): string {
  return f.params?.error ?? ''
}

export function buildText(summary: Summary, lang: Lang, opts: ReportOptions = {}): string {
  const s = normalizeSummary(summary)
  const t = (key: UiKey, p?: Record<string, string | number>) => ui(lang, key, p)
  const title = `mamori: ${t('rep.title')}`
  const out: string[] = [title, '='.repeat(title.length), '']
  const rows: [string, string][] = [
    [t('rep.program'), program(s)],
    [t('rep.generated'), formatTime(s.generated)],
    [t('rep.system'), systemLine(lang, s.system)],
    [t('rep.host'), s.system.host],
    [t('rep.user'), s.system.user],
  ]
  const width = Math.max(...rows.map(([k]) => k.length)) + 2
  for (const [k, v] of rows) out.push(`${`${k}:`.padEnd(width)}${v}`)
  out.push('')
  if (opts.demo) out.push(t('rep.demo'), '')
  out.push(`${t('rep.overall')}: [${label(lang, s.status)}] ${codeText(lang, s.code, {}, 'text')}`)
  out.push(`${t('rep.checksRun')}: ${s.results.length} / ${CHECK_IDS.length}`)

  for (const { id, result } of ordered(s)) {
    const head = `${t('rep.module', { n: meta(id).module })}. ${checkTitle(lang, id)}`
    out.push('', head, '-'.repeat(head.length))
    if (!result) {
      out.push(`${t('rep.verdict')}: ${t('rep.notRun')}`)
      continue
    }
    out.push(
      `${t('rep.verdict')}: [${label(lang, result.status)}] ${codeText(lang, result.code, result.params, 'text')}`,
    )
    if (rawError(result)) out.push(`${t('rep.error')}: ${rawError(result)}`)
    out.push(`${t('rep.elapsed')}: ${duration(lang, result.elapsedMs)}`)
    if (result.findings.length > 0) out.push(`${t('rep.findings')}:`)
    for (const f of result.findings) {
      out.push(`  - [${statusWord(lang, f.status)}] ${codeText(lang, f.code, f.params, 'text')}`)
      const err = rawError(f)
      if (err) out.push(`      ${t('rep.error')}: ${err}`)
    }
  }
  return `${out.join('\n')}\n`
}

export function escapeHtml(s: string): string {
  return s
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}

const HTML_STYLE = `
*{box-sizing:border-box}
body{margin:0 auto;max-width:960px;padding:32px 28px;background:#fff;color:#1b1b1b;
font:15px/1.5 "Segoe UI",Arial,sans-serif}
h1{font-size:24px;margin:0 0 4px}
h2{font-size:18px;margin:28px 0 8px;padding-bottom:4px;border-bottom:1px solid #999}
.sub{margin:0 0 20px;color:#555}
.demo{border:2px solid #000;padding:8px 12px;font-weight:700;margin:0 0 20px}
table{border-collapse:collapse;width:100%}
.meta th{text-align:left;font-weight:600;padding:3px 16px 3px 0;white-space:nowrap;vertical-align:top;width:1%}
.meta td{padding:3px 0}
.findings{margin-top:8px}
.findings th,.findings td{border:1px solid #bbb;padding:6px 8px;text-align:left;vertical-align:top}
.findings th{background:#eee;font-weight:600}
.findings tr{break-inside:avoid}
.verdict{margin:6px 0}
.muted{color:#555;margin:2px 0}
.badge{display:inline-block;min-width:96px;padding:1px 6px;margin-right:6px;border:1.5px solid currentColor;
border-radius:3px;font-size:12px;font-weight:700;text-transform:uppercase;letter-spacing:.04em;text-align:center}
.pass .badge,.badge.pass{color:#1d6b2a}
.warn .badge,.badge.warn{color:#8a5a00}
.fail .badge,.badge.fail{color:#a3160d}
.error .badge,.badge.error{color:#6e1414}
.skip .badge,.badge.skip{color:#555}
.err{margin-top:4px;font:12.5px/1.4 Consolas,"Cascadia Mono",monospace;color:#444;word-break:break-word}
code{font:12px Consolas,"Cascadia Mono",monospace;color:#444}
.overall{font-size:16px}
@media print{body{padding:0;max-width:none}@page{margin:16mm}h2{break-after:avoid}}
`

export function buildHtml(summary: Summary, lang: Lang, opts: ReportOptions = {}): string {
  const s = normalizeSummary(summary)
  const t = (key: UiKey, p?: Record<string, string | number>) => escapeHtml(ui(lang, key, p))
  const text = (code: string, params?: Record<string, string>) =>
    escapeHtml(codeText(lang, code, params, 'text'))
  const badge = (status: Status) =>
    `<span class="badge ${status}">${escapeHtml(statusWord(lang, status))}</span>`
  const parts: string[] = []
  parts.push(
    '<!doctype html>',
    `<html lang="${lang}">`,
    '<head>',
    '<meta charset="utf-8">',
    '<meta name="viewport" content="width=device-width, initial-scale=1">',
    `<title>mamori: ${t('rep.title')}, ${escapeHtml(formatTime(s.generated))}</title>`,
    `<style>${HTML_STYLE}</style>`,
    '</head>',
    '<body>',
    `<h1>${t('rep.title')}</h1>`,
    `<p class="sub">${escapeHtml(program(s))}</p>`,
  )
  if (opts.demo) parts.push(`<p class="demo">${t('rep.demo')}</p>`)
  const meta_: [string, string][] = [
    [t('rep.generated'), escapeHtml(formatTime(s.generated))],
    [t('rep.system'), escapeHtml(systemLine(lang, s.system))],
    [t('rep.host'), escapeHtml(s.system.host)],
    [t('rep.user'), escapeHtml(s.system.user)],
    [t('rep.checksRun'), `${s.results.length} / ${CHECK_IDS.length}`],
  ]
  parts.push(
    '<table class="meta"><tbody>',
    ...meta_.map(([k, v]) => `<tr><th>${k}</th><td>${v}</td></tr>`),
    '</tbody></table>',
  )
  parts.push(
    `<section class="overall ${s.status}">`,
    `<h2>${t('rep.overall')}</h2>`,
    `<p class="verdict">${badge(s.status)}${text(s.code)}</p>`,
    '</section>',
  )
  for (const { id, result } of ordered(s)) {
    parts.push(`<section class="check ${result?.status ?? 'none'}">`)
    parts.push(`<h2>${t('rep.module', { n: meta(id).module })}. ${escapeHtml(checkTitle(lang, id))}</h2>`)
    if (!result) {
      parts.push(`<p class="verdict">${t('rep.verdict')}: ${t('rep.notRun')}</p>`, '</section>')
      continue
    }
    parts.push(`<p class="verdict">${badge(result.status)}${text(result.code, result.params)}</p>`)
    const verdictErr = rawError(result)
    if (verdictErr) parts.push(`<div class="err">${t('rep.error')}: ${escapeHtml(verdictErr)}</div>`)
    parts.push(
      `<p class="muted">${t('rep.elapsed')}: ${escapeHtml(duration(lang, result.elapsedMs))}. ${t('rep.started')}: ${escapeHtml(formatTime(result.started))}.</p>`,
    )
    if (result.findings.length > 0) {
      parts.push(
        '<table class="findings">',
        `<thead><tr><th>${t('rep.status')}</th><th>${t('rep.text')}</th><th>${t('rep.code')}</th></tr></thead>`,
        '<tbody>',
      )
      for (const f of result.findings) {
        const err = rawError(f)
        const errHtml = err ? `<div class="err">${t('rep.error')}: ${escapeHtml(err)}</div>` : ''
        parts.push(
          `<tr><td>${badge(f.status)}</td><td>${text(f.code, f.params)}${errHtml}</td><td><code>${escapeHtml(f.code)}</code></td></tr>`,
        )
      }
      parts.push('</tbody></table>')
    }
    parts.push('</section>')
  }
  parts.push('</body>', '</html>', '')
  return parts.join('\n')
}

export function buildJson(summary: Summary, opts: ReportOptions = {}): string {
  const s = normalizeSummary(summary)
  return `${JSON.stringify(opts.demo ? { demo: true, ...s } : s, null, 2)}\n`
}

export function buildReport(
  format: ReportFormat,
  summary: Summary,
  lang: Lang,
  opts: ReportOptions = {},
): string {
  switch (format) {
    case 'txt':
      return buildText(summary, lang, opts)
    case 'html':
      return buildHtml(summary, lang, opts)
    case 'json':
      return buildJson(summary, opts)
  }
}
