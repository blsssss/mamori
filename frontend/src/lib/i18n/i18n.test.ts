import { describe, expect, it } from 'vitest'
import { goCodes, goParams } from '../../test/gocodes'
import { CALL_FAILED, type Lang } from '../types'
import { codeText, DICTS, duration, fallback, render, type UiKey, ui } from './index'

const LANGS: Lang[] = ['ru', 'en']
const placeholders = (s: string) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort()
const same = (_: string, v: string) => v

describe('codes from the Go checks', () => {
  const codes = goCodes()

  it('are found in the sources', () => {
    expect(codes.length).toBeGreaterThan(60)
    expect(codes).toContain('net.verdict.online')
    expect(codes).toContain('report.protected')
  })

  for (const lang of LANGS) {
    it(`all have a ${lang} text`, () => {
      const missing = codes.filter((c) => !(c in DICTS[lang].codes))
      expect(missing).toEqual([])
    })
  }

  it('have no stale translations', () => {
    const own = [CALL_FAILED]
    const stale = Object.keys(DICTS.ru.codes).filter((c) => !codes.includes(c) && !own.includes(c))
    expect(stale).toEqual([])
  })
})

describe('params from the Go checks', () => {
  const params = goParams()
  const keys = (code: string) => [...(params.get(code) ?? [])].sort()

  it('are found at every kind of call site', () => {
    expect(keys('net.dns.ok')).toEqual(['addrs', 'host', 'ms'])
    expect(keys('net.icmp.ok')).toEqual(['rtt_ms', 'target', 'ttl'])
    expect(keys('net.adapter.ok')).toEqual(['gateway', 'ipv4', 'name'])
    expect(keys('fw.rule.blocked')).toEqual(['error', 'ms', 'target'])
    expect(keys('av.eicar.not_detected')).toEqual(['error', 'seconds'])
    expect(keys('inv.defender.status')).toEqual([
      'antivirus',
      'mode',
      'realtime',
      'service',
      'signature_age',
      'signature_version',
    ])
  })

  it('fill every placeholder of the texts, optional segments included', () => {
    for (const lang of LANGS) {
      const d = DICTS[lang]
      for (const [code, text] of [...Object.entries(d.codes), ...Object.entries(d.lines)]) {
        const emitted = params.get(code) ?? new Set<string>()
        const missing = placeholders(text).filter((p) => !emitted.has(p))
        expect(missing, `${lang} ${code}`).toEqual([])
      }
    }
  })

  it('are all shown, except the raw OS error that goes under the text', () => {
    for (const [code, emitted] of params) {
      const shown = placeholders(DICTS.ru.codes[code] ?? '')
      const hidden = [...emitted].filter((k) => k !== 'error' && !shown.includes(k))
      expect(hidden, code).toEqual([])
    }
  })
})

describe('dictionaries', () => {
  it('have the same keys in both languages', () => {
    expect(Object.keys(DICTS.en.codes).sort()).toEqual(Object.keys(DICTS.ru.codes).sort())
    expect(Object.keys(DICTS.en.ui).sort()).toEqual(Object.keys(DICTS.ru.ui).sort())
    for (const lang of LANGS) {
      const extra = Object.keys(DICTS[lang].lines).filter((k) => !(k in DICTS[lang].codes))
      expect(extra).toEqual([])
    }
  })

  it('use the same placeholders in both languages and both styles', () => {
    for (const [code, ru] of Object.entries(DICTS.ru.codes)) {
      expect(placeholders(DICTS.en.codes[code]), code).toEqual(placeholders(ru))
      for (const lang of LANGS) {
        const line = DICTS[lang].lines[code]
        if (line) expect(placeholders(line), `${lang} ${code}`).toEqual(placeholders(DICTS[lang].codes[code]))
      }
    }
    for (const [key, ru] of Object.entries(DICTS.ru.ui)) {
      expect(placeholders(DICTS.en.ui[key as UiKey]), key).toEqual(placeholders(ru))
    }
  })

  it('keep the raw OS error out of the text, it is shown under it', () => {
    for (const lang of LANGS) {
      for (const [code, text] of Object.entries({ ...DICTS[lang].codes, ...DICTS[lang].lines })) {
        expect(text.includes('{error}'), `${lang} ${code}`).toBe(false)
      }
    }
  })

  it('contain no em dash, no emoji and no exclamation marks', () => {
    for (const lang of LANGS) {
      const d = DICTS[lang]
      const all = [
        ...Object.values(d.ui),
        ...Object.values(d.codes),
        ...Object.values(d.lines),
        ...Object.values(d.status),
        ...Object.values(d.values).flatMap((m) => Object.values(m)),
      ]
      for (const s of all) {
        expect(s, s).not.toMatch(/[\u2014\u2015!]|\p{Emoji_Presentation}/u)
      }
    }
  })
})

describe('render', () => {
  it('drops an optional segment with an empty param', () => {
    const t = 'IPv4 {ipv4}[[, gateway {gateway}]].'
    expect(render(t, { ipv4: '10.0.0.2', gateway: '' }, same)).toBe('IPv4 10.0.0.2.')
    expect(render(t, { ipv4: '10.0.0.2', gateway: '10.0.0.1' }, same)).toBe(
      'IPv4 10.0.0.2, gateway 10.0.0.1.',
    )
  })

  it('marks a missing required param', () => {
    expect(render('to {target}', {}, same)).toBe('to ?')
  })
})

describe('codeText', () => {
  it('speaks in the first person in the text box and stays impersonal in reports', () => {
    const p = { host: 'www.msftconnecttest.com', addrs: '13.107.4.52', ms: '12' }
    expect(codeText('ru', 'net.dns.ok', p)).toBe(
      'Проверила DNS: www.msftconnecttest.com разрешается в 13.107.4.52 за 12 мс.',
    )
    expect(codeText('ru', 'net.dns.ok', p, 'text')).toBe(
      'DNS: www.msftconnecttest.com разрешается в 13.107.4.52 за 12 мс.',
    )
    expect(codeText('en', 'net.dns.ok', p, 'text')).toBe(
      'DNS: www.msftconnecttest.com resolves to 13.107.4.52 in 12 ms.',
    )
  })

  it('translates enum params, lists of them and decimal seconds', () => {
    const profile = {
      profile: 'private',
      enabled: 'false',
      active: 'true',
      inbound: 'block',
      outbound: 'allow',
    }
    expect(codeText('ru', 'inv.winfw.profile', profile, 'text')).toBe(
      'Брандмауэр Windows, профиль «частный» (активный): выключен, входящие по умолчанию: блокировать, исходящие: разрешать.',
    )
    expect(codeText('ru', 'fw.state.enforcing', { profiles: 'domain, private' }, 'text')).toBe(
      'Брандмауэр Windows включён на активных профилях сети: доменный, частный.',
    )
    expect(codeText('ru', 'av.eicar.removed', { seconds: '1.3' }, 'text')).toContain('через 1,3 с')
    expect(codeText('en', 'av.eicar.removed', { seconds: '1.3' }, 'text')).toContain('1.3 s')
    expect(codeText('ru', 'inv.files.unavailable', { evidence: 'process', error: 'x' }, 'text')).toContain(
      'список процессов',
    )
  })

  it('writes the Security Center timestamp as a local date, not as English text', () => {
    const p = {
      name: 'Microsoft Defender',
      state: 'on',
      signature: 'up_to_date',
      source: 'wmi',
      path: '',
      timestamp: 'Thu, 09 Oct 2026 18:02:11 GMT',
    }
    const local = new Date(Date.UTC(2026, 9, 9, 18, 2, 11))
    const pad = (n: number) => String(n).padStart(2, '0')
    const want = `${local.getFullYear()}-${pad(local.getMonth() + 1)}-${pad(local.getDate())} ${pad(local.getHours())}:02:11`
    expect(codeText('ru', 'inv.av.product', p, 'text')).toContain(`состояние на ${want}.`)
    expect(codeText('en', 'inv.av.product', { ...p, timestamp: 'not a date' }, 'text')).toContain(
      'state as of not a date.',
    )
  })

  it('keeps unknown enum values as they are', () => {
    expect(codeText('en', 'inv.fw.product', { name: 'X', state: 'weird', source: 'wsc' }, 'text')).toBe(
      'Third-party firewall “X”: weird (source: Windows Security Center, WSC API).',
    )
  })

  it('falls back to the code and its params', () => {
    expect(codeText('ru', 'net.new_probe.ok', { target: '1.1.1.1', ms: '5' })).toBe(
      'net.new_probe.ok {target: 1.1.1.1, ms: 5}',
    )
    expect(fallback('x.y')).toBe('x.y')
  })
})

describe('ui and duration', () => {
  it('fills params', () => {
    expect(ui('ru', 'box.counter', { n: 3, total: 7 })).toBe('3 / 7')
    expect(ui('en', 'save.done', { path: 'C:\\r.txt' })).toBe('Report saved: C:\\r.txt')
  })

  it('formats elapsed time', () => {
    expect(duration('ru', 1250)).toBe('1,3 с')
    expect(duration('en', 400)).toBe('0.4 s')
    expect(duration('ru', 125_000)).toBe('2 мин 05 с')
  })
})
