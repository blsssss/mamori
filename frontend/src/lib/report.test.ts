import { describe, expect, it } from 'vitest'
import { FULL, result, summary } from '../test/fixtures'
import {
  buildHtml,
  buildJson,
  buildReport,
  buildText,
  escapeHtml,
  formatTime,
  normalizeSummary,
  parseTime,
  reportName,
} from './report'

const unprotected = summary(FULL, 'fail', 'report.unprotected')
// art and metaphors belong to the interface, never to a report
const ART = /Рейму|Reimu|霊夢|ёкай|youkai|барьер|barrier|обере|ward|妖怪|結界|Проверила|I checked/i

describe('normalizeSummary', () => {
  it('turns "protected" into "partial" when some checks did not run', () => {
    const s = normalizeSummary(summary(FULL.slice(0, 2), 'pass', 'report.protected'))
    expect(s.status).toBe('warn')
    expect(s.code).toBe('report.partial')
  })

  it('keeps every other verdict', () => {
    const all = FULL.map((r) => ({ ...r, status: 'pass' as const }))
    expect(normalizeSummary(summary(all, 'pass', 'report.protected')).code).toBe('report.protected')
    expect(normalizeSummary(summary(FULL.slice(0, 1), 'fail', 'report.unprotected')).code).toBe(
      'report.unprotected',
    )
  })
})

describe('reportName', () => {
  it('follows mamori-report-YYYYMMDD-HHMM.ext in local time', () => {
    const d = new Date(2026, 9, 9, 7, 5)
    expect(reportName('txt', d)).toBe('mamori-report-20261009-0705.txt')
    expect(reportName('html', d)).toBe('mamori-report-20261009-0705.html')
    expect(reportName('json', d)).toBe('mamori-report-20261009-0705.json')
  })
})

describe('time', () => {
  it('reads Go timestamps with nanoseconds', () => {
    const d = parseTime('2026-10-09T21:55:03.123456789+03:00')
    expect(d?.toISOString()).toBe('2026-10-09T18:55:03.123Z')
    expect(parseTime('nonsense')).toBeNull()
    expect(formatTime('nonsense')).toBe('nonsense')
    expect(formatTime('2026-10-09T21:55:03Z')).toMatch(/^2026-10-(09|10) \d\d:55:03$/)
  })
})

describe('a check whose Run call failed', () => {
  const failed = {
    ...result('firewall', 'error', 'app.call_failed'),
    params: { error: 'Access <denied>' },
  }
  const s = summary([FULL[0], failed], 'error', 'report.error')

  it('is listed with its error, not as a module that did not run', () => {
    const txt = buildText(s, 'ru')
    expect(txt).toContain('Итог: [ОШИБКА] Проверку выполнить не удалось: вызов программы завершился ошибкой.')
    expect(txt).toContain('Сообщение системы: Access <denied>')
    expect(txt.match(/не выполнялась/g)).toHaveLength(2)
    const html = buildHtml(s, 'en')
    expect(html).toContain('The check could not run: the program call returned an error.')
    expect(html).toContain('<div class="err">System message: Access &lt;denied&gt;</div>')
  })
})

describe('buildText', () => {
  const ru = buildText(unprotected, 'ru')
  const en = buildText(unprotected, 'en')

  it('has the header, the overall verdict and every module', () => {
    expect(ru).toContain('mamori: Отчёт о проверке средств защиты ПК')
    expect(ru).toContain('mamori 1.0.0 (abc1234)')
    expect(ru).toContain('Windows 11 Pro 24H2, сборка 26100.4351, amd64')
    expect(ru).toContain('PC-01\\student')
    expect(ru).toContain('Общий итог: [НЕ ПРОЙДЕНО] Компьютер не защищён')
    expect(ru).toContain('Выполнено проверок: 4 / 4')
    for (const n of [1, 2, 3, 4]) expect(ru).toContain(`Модуль ${n}.`)
    expect(en).toContain('Overall verdict: [FAILED] The computer is not protected')
  })

  it('lists findings with their status and the raw OS error under them', () => {
    expect(ru).toContain('  - [пройдено] DNS: www.msftconnecttest.com разрешается в 13.107.4.52 за 18 мс.')
    expect(ru).toContain(
      '  - [не пройдено] Ресурс https://forbidden.example/, запрещённый политикой, доступен',
    )
    expect(ru).toContain('      Сообщение системы: Access is denied. <script>alert(1)</script>')
    expect(ru).toContain('Время выполнения: 1,3 с')
  })

  it('says which modules did not run', () => {
    const partial = buildText(summary(FULL.slice(0, 1), 'pass', 'report.protected'), 'ru')
    expect(partial).toContain('Общий итог: [ВНИМАНИЕ] Выполнены не все проверки')
    expect(partial.match(/Итог: не выполнялась/g)).toHaveLength(3)
  })

  it('is strictly technical', () => {
    expect(ru).not.toMatch(ART)
    expect(en).not.toMatch(ART)
    expect(ru).not.toContain(String.fromCharCode(0x2014))
  })

  it('marks demo data', () => {
    expect(buildText(unprotected, 'ru', { demo: true })).toContain('ДЕМО-РЕЖИМ')
    expect(ru).not.toContain('ДЕМО')
  })
})

describe('buildHtml', () => {
  const html = buildHtml(unprotected, 'ru')

  it('is a self-contained printable document', () => {
    expect(html.startsWith('<!doctype html>')).toBe(true)
    expect(html).toContain('<html lang="ru">')
    expect(html).toContain('<meta charset="utf-8">')
    expect(html).toContain('@media print')
    expect(html).not.toMatch(/(src|href)=|@import|url\(/)
  })

  it('escapes everything that comes from the system', () => {
    expect(html).not.toContain('<script>')
    expect(html).toContain('Access is denied. &lt;script&gt;alert(1)&lt;/script&gt;')
    expect(escapeHtml(`<a href="x">'&'</a>`)).toBe('&lt;a href=&quot;x&quot;&gt;&#39;&amp;&#39;&lt;/a&gt;')
  })

  it('shows statuses as words, not colour alone, and keeps the codes', () => {
    expect(html).toContain('<span class="badge fail">не пройдено</span>')
    expect(html).toContain('<code>fw.policy.reachable</code>')
  })

  it('is strictly technical', () => {
    expect(html).not.toMatch(ART)
    expect(buildHtml(unprotected, 'en')).not.toMatch(ART)
  })
})

describe('buildJson', () => {
  it('is the pretty printed summary', () => {
    const json = buildJson(unprotected)
    expect(JSON.parse(json)).toEqual(unprotected)
    expect(json).toContain('\n  "version": "1.0.0",')
    expect(json.endsWith('}\n')).toBe(true)
  })

  it('carries the normalized verdict and the demo flag', () => {
    const parsed = JSON.parse(
      buildJson(summary([result('internet', 'pass', 'net.verdict.online')], 'pass', 'report.protected'), {
        demo: true,
      }),
    )
    expect(parsed.code).toBe('report.partial')
    expect(parsed.demo).toBe(true)
  })
})

describe('buildReport', () => {
  it('picks the format', () => {
    expect(buildReport('txt', unprotected, 'en')).toBe(buildText(unprotected, 'en'))
    expect(buildReport('html', unprotected, 'en')).toBe(buildHtml(unprotected, 'en'))
    expect(buildReport('json', unprotected, 'en')).toBe(buildJson(unprotected))
  })
})
