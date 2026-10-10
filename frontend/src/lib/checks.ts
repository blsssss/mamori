import type { BgId, CgId } from './assets'
import type { CheckId, Status } from './types'

export interface CheckMeta {
  id: CheckId
  module: number
  numeral: string
  jp: string
  bg: BgId
  // event CG shown when the check passes
  cg?: Exclude<CgId, 'title' | 'tea'>
}

export const CHECKS: readonly CheckMeta[] = [
  { id: 'internet', module: 1, numeral: '一', jp: '参道', bg: 'sandou' },
  { id: 'inventory', module: 2, numeral: '二', jp: '結界の点検', bg: 'shrine' },
  { id: 'firewall', module: 3, numeral: '三', jp: '博麗大結界', bg: 'barrier', cg: 'barrier' },
  { id: 'antivirus', module: 4, numeral: '四', jp: '妖怪退治', bg: 'interior', cg: 'exorcism' },
]

export function meta(id: CheckId): CheckMeta {
  return CHECKS.find((c) => c.id === id) ?? CHECKS[0]
}

// hanko seal of each status, empty while a check has not run
export const SEAL: Record<Status, string> = {
  pass: '良',
  warn: '注',
  fail: '否',
  skip: '休',
  error: '乱',
}
