import { codeText, ui } from './i18n'
import type { CheckId, CheckState, Lang, Status } from './types'

export type LineKind = 'intro' | 'hint' | 'notice' | 'finding' | 'verdict' | 'error'

export interface Line {
  key: string
  kind: LineKind
  text: string
  status?: Status
  // the raw OS error of a finding, shown as is under the text
  raw?: string
}

// the lines Reimu says about one check: what the module does, then its findings and verdict
export function dialogue(lang: Lang, id: CheckId, st: CheckState): Line[] {
  const lines: Line[] = [{ key: 'intro', kind: 'intro', text: ui(lang, `say.intro.${id}`) }]
  if (st.phase === 'idle') {
    lines.push({ key: 'hint', kind: 'hint', text: ui(lang, 'say.hint') })
    return lines
  }
  for (const n of st.notices) {
    lines.push({ key: `notice-${n}`, kind: 'notice', text: ui(lang, `say.notice.${n}`) })
  }
  const findings = st.result?.findings ?? st.live
  findings.forEach((f, i) => {
    lines.push({
      key: `f${i}`,
      kind: 'finding',
      status: f.status,
      text: codeText(lang, f.code, f.params, 'line'),
      raw: f.params?.error || undefined,
    })
  })
  if (st.result) {
    lines.push({
      key: 'verdict',
      kind: 'verdict',
      status: st.result.status,
      text: codeText(lang, st.result.code, st.result.params, 'line'),
    })
  } else if (st.error !== null) {
    lines.push({
      key: 'error',
      kind: 'error',
      status: 'error',
      text: ui(lang, 'say.runError'),
      raw: st.error,
    })
  }
  return lines
}
