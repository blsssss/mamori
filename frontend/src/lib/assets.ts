import bgBarrier from '../assets/bg/barrier.webp'
import bgInterior from '../assets/bg/interior.webp'
import bgSandou from '../assets/bg/sandou.webp'
import bgShrine from '../assets/bg/shrine.webp'
import bgVeranda from '../assets/bg/veranda.webp'
import cgBarrier from '../assets/cg/barrier.webp'
import cgExorcism from '../assets/cg/exorcism.webp'
import cgTea from '../assets/cg/tea.webp'
import cgTitle from '../assets/cg/title.webp'
import iconUrl from '../assets/icon-64.png'
import rError from '../assets/reimu/error.webp'
import rFail from '../assets/reimu/fail.webp'
import rIdle from '../assets/reimu/idle.webp'
import rPass from '../assets/reimu/pass.webp'
import rRunning from '../assets/reimu/running.webp'
import rSkip from '../assets/reimu/skip.webp'
import rWarn from '../assets/reimu/warn.webp'

export type BgId = 'sandou' | 'shrine' | 'barrier' | 'interior' | 'veranda'
export type CgId = 'title' | 'exorcism' | 'barrier' | 'tea'
export type Expression = 'idle' | 'running' | 'pass' | 'warn' | 'fail' | 'skip' | 'error'

export const BG: Record<BgId, string> = {
  sandou: bgSandou,
  shrine: bgShrine,
  barrier: bgBarrier,
  interior: bgInterior,
  veranda: bgVeranda,
}

export const CG: Record<CgId, string> = {
  title: cgTitle,
  exorcism: cgExorcism,
  barrier: cgBarrier,
  tea: cgTea,
}

export const SPRITE: Record<Expression, string> = {
  idle: rIdle,
  running: rRunning,
  pass: rPass,
  warn: rWarn,
  fail: rFail,
  skip: rSkip,
  error: rError,
}

export const EXPRESSIONS = Object.keys(SPRITE) as Expression[]

export const ICON = iconUrl
