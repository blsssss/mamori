import { describe, expect, it } from 'vitest'
import { brokenStorage, memoryStorage } from '../test/fixtures'
import { DEFAULTS, loadSettings, msPerChar, policyList, sanitize, saveSettings } from './settings'

describe('settings', () => {
  it('fall back to the defaults when the stored value is missing or broken', () => {
    expect(sanitize(null)).toEqual(DEFAULTS)
    expect(sanitize('nonsense')).toEqual(DEFAULTS)
    const store = memoryStorage()
    store.setItem('mamori.settings.v1', '{not json')
    expect(loadSettings(store)).toEqual(DEFAULTS)
  })

  it('clamp and validate every field', () => {
    const s = sanitize({
      lang: 'de',
      textSpeed: 99,
      eventCgs: 'yes',
      eicarWaitSec: 1,
      allowElevation: false,
      reportFormat: 'pdf',
      policyTargets: 42,
    })
    expect(s).toEqual({ ...DEFAULTS, textSpeed: 10, eicarWaitSec: 5, allowElevation: false })
    expect(sanitize({ eicarWaitSec: 600 }).eicarWaitSec).toBe(60)
    expect(sanitize({ textSpeed: Number.NaN }).textSpeed).toBe(DEFAULTS.textSpeed)
  })

  it('round-trip through storage', () => {
    const store = memoryStorage()
    const s = { ...DEFAULTS, lang: 'en' as const, eicarWaitSec: 30, policyTargets: 'example.com' }
    expect(saveSettings(s, store)).toBe(true)
    expect(loadSettings(store)).toEqual(s)
  })

  it('work when storage throws on every call', () => {
    expect(loadSettings(brokenStorage)).toEqual(DEFAULTS)
    expect(saveSettings(DEFAULTS, brokenStorage)).toBe(false)
    expect(loadSettings(null)).toEqual(DEFAULTS)
    expect(saveSettings(DEFAULTS, null)).toBe(false)
  })
})

describe('policyList', () => {
  it('takes one target per line and skips blanks and comments', () => {
    expect(policyList(' example.com \r\n\n# lab rule\nhttps://example.org/x\n203.0.113.10:8080\n')).toEqual([
      'example.com',
      'https://example.org/x',
      '203.0.113.10:8080',
    ])
  })
})

describe('msPerChar', () => {
  it('is zero for instant text and grows with the setting', () => {
    expect(msPerChar(0)).toBe(0)
    expect(msPerChar(1)).toBeLessThan(msPerChar(10))
  })
})
