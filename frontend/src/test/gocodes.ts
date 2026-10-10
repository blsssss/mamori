// the finding and verdict codes of the Go checks and the params they are emitted with, read from the
// sources so that a code added or a param renamed in Go without a translation fails the tests
const sources = import.meta.glob(
  ['../../../internal/**/*.go', '!../../../internal/**/*_test.go', '../../../*.go'],
  {
    query: '?raw',
    import: 'default',
    eager: true,
  },
) as Record<string, string>

const CODE_TEXT = String.raw`"((?:net|inv|fw|av|common|report)\.[a-z_]+(?:\.[a-z_]+)*)"`
const CODE = new RegExp(CODE_TEXT, 'g')
const CODE_ONLY = new RegExp(`^${CODE_TEXT}$`)

export function goFiles(): string[] {
  return Object.keys(sources)
}

export function goCodes(): string[] {
  const found = new Set<string>()
  for (const text of Object.values(sources)) {
    for (const m of text.matchAll(CODE)) found.add(m[1])
  }
  return [...found].sort()
}

// the arguments of a Go call or composite literal whose opening bracket is at text[open], split at
// the top-level commas; strings, runes and nested brackets are skipped over
export function splitArgs(text: string, open: number): string[] {
  const close: Record<string, string> = { '(': ')', '{': '}', '[': ']' }
  const stack = [close[text[open]]]
  const args: string[] = []
  let from = open + 1
  for (let i = open + 1; i < text.length; i++) {
    const c = text[i]
    if (c === '"' || c === "'") {
      for (i++; i < text.length && text[i] !== c; i++) if (text[i] === '\\') i++
    } else if (c === '`') {
      i = text.indexOf('`', i + 1)
    } else if (c in close) {
      stack.push(close[c])
    } else if (c === stack[stack.length - 1]) {
      stack.pop()
      if (stack.length === 0) {
        const last = text.slice(from, i).trim()
        if (last !== '') args.push(last)
        return args
      }
    } else if (c === ',' && stack.length === 1) {
      args.push(text.slice(from, i).trim())
      from = i + 1
    }
  }
  return args
}

const literal = (arg: string) => /^"([^"\\]*)"$/.exec(arg)?.[1]

// keys of a flat "key", value, ... list
function keysOf(args: string[]): string[] {
  return args.filter((_, i) => i % 2 === 0).flatMap((a) => literal(a) ?? [])
}

// the []string{...} literals a function returns or a variable is built from, then appended to
function spreadKeys(text: string, name: string): string[] {
  const keys: string[] = []
  const fn = new RegExp(String.raw`func ${name}\(`).exec(text)
  if (fn) {
    const lit = text.indexOf('[]string{', fn.index)
    if (lit >= 0) keys.push(...keysOf(splitArgs(text, lit + '[]string'.length)))
  }
  for (const m of text.matchAll(new RegExp(String.raw`\b${name}\s*:?=\s*\[\]string\{`, 'g'))) {
    keys.push(...keysOf(splitArgs(text, m.index + m[0].length - 1)))
  }
  for (const m of text.matchAll(new RegExp(String.raw`\b${name}\s*=\s*append\(${name},`, 'g'))) {
    keys.push(...keysOf(splitArgs(text, m.index + m[0].indexOf('(')).slice(1)))
  }
  return keys
}

// the params each code is emitted with, by any call site: Add, Finish, finding, single or a note{}
export function goParams(): Map<string, Set<string>> {
  const all = Object.values(sources).join('\n')
  const consts = new Map<string, string>()
  for (const m of all.matchAll(new RegExp(String.raw`(\w+)\s*=\s*${CODE_TEXT}`, 'g'))) consts.set(m[1], m[2])

  const params = new Map<string, Set<string>>()
  for (const text of Object.values(sources)) {
    for (const m of text.matchAll(/\b(?:Add|Finish|finding|single)\(|\bnote\{/g)) {
      const args = splitArgs(text, m.index + m[0].length - 1)
      const code = CODE_ONLY.exec(args[1] ?? '')?.[1] ?? consts.get(args[1] ?? '')
      if (!code) continue
      const keys = params.get(code) ?? new Set<string>()
      params.set(code, keys)
      const rest = args.slice(2)
      for (const [i, arg] of rest.entries()) {
        const spread = /^(\w+)(?:\(.*\))?\.\.\.$/s.exec(arg)
        if (spread) {
          for (const k of spreadKeys(text, spread[1])) keys.add(k)
        } else if (arg.startsWith('[]string{')) {
          for (const k of keysOf(splitArgs(text, text.indexOf(arg, m.index) + '[]string'.length))) keys.add(k)
        } else if (i % 2 === 0) {
          const k = literal(arg)
          if (k !== undefined) keys.add(k)
        }
      }
    }
  }
  return params
}
