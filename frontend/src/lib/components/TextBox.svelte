<!-- the VN text window: one line at a time with a typewriter, the findings of the selected check -->
<script lang="ts">
import { onMount, untrack } from 'svelte'
import { dialogue } from '../dialogue'
import { byKeyboard } from '../input'
import { useStore } from '../state.svelte'
import QuickMenu from './QuickMenu.svelte'
import Seal from './Seal.svelte'

const store = useStore()
const FULL = Number.MAX_SAFE_INTEGER

const st = $derived(store.checks[store.selected])
const lines = $derived(dialogue(store.lang, store.selected, st))
const running = $derived(st.phase === 'running' || st.phase === 'notice')

let cursor = $state(0)
let typed = $state(0)
// follow: the cursor walks to new lines on its own while a check runs
let follow = $state(false)
// the line changed because the user asked, so a screen reader reads it out
let manual = $state(false)
let say = $state<HTMLDivElement>()

const index = $derived(Math.min(cursor, lines.length - 1))
const line = $derived(lines[index])
const shown = $derived(line.text.slice(0, typed))
const full = $derived(typed >= line.text.length)
const atEnd = $derived(index >= lines.length - 1)

// a new selection or a new run starts at the newest line: the verdict of a finished check, the
// notice or the latest finding of a running one; a check that never ran starts at its intro
$effect(() => {
  const state = st
  void store.selected
  untrack(() => {
    manual = false
    cursor = state.phase === 'idle' ? 0 : lines.length - 1
    follow = state.phase === 'running' || state.phase === 'notice'
    typed = 0
  })
})

// typewriter, time based so a slow frame does not slow the text down
$effect(() => {
  const len = line.text.length
  const ms = store.charMs
  void index
  const from = untrack(() => typed)
  if (from >= len) return
  if (ms <= 0 || typeof requestAnimationFrame !== 'function') {
    typed = FULL
    return
  }
  const t0 = performance.now() - from * ms
  let raf = 0
  const tick = (now: number) => {
    // a click or the stream catching up finished the line, the next frame must not take that back
    if (typed >= len) return
    const n = Math.min(len, Math.floor((now - t0) / ms))
    if (n !== typed) typed = n
    if (n < len) raf = requestAnimationFrame(tick)
  }
  raf = requestAnimationFrame(tick)
  return () => cancelAnimationFrame(raf)
})

// follow mode: once a line is out, move on to the next; far behind, skip the typing to catch up.
// The distance is read untracked, otherwise every streamed finding would restart the timer
$effect(() => {
  // the index is a dependency of its own: a line shown at once leaves full and atEnd unchanged
  const at = index
  if (!follow || !full || atEnd) return
  const hurry = untrack(() => lines.length - 1 - at > 2)
  const timer = setTimeout(
    () => {
      const behind = lines.length - 1 - (at + 1)
      cursor = at + 1
      typed = behind > 2 ? FULL : 0
      manual = false
    },
    hurry ? 140 : 650,
  )
  return () => clearTimeout(timer)
})

// lines are waiting: the one being typed is finished at once instead of holding the stream back
$effect(() => {
  if (!follow || full || atEnd) return
  const timer = setTimeout(() => {
    typed = FULL
  }, 250)
  return () => clearTimeout(timer)
})

// AUTO starts the next check only once the verdict of this one is on screen, typed out in full
onMount(() => store.attachReader())
// a keyboard user arrives on the text, where Enter and Space go on advancing it
onMount(() => {
  if (byKeyboard()) say?.focus()
})
$effect(() => {
  if (st.phase === 'done' && atEnd && full) store.verdictShown(store.selected)
})

// a long line scrolled down does not leave the next one scrolled
$effect(() => {
  void index
  void st
  if (say) say.scrollTop = 0
})

function next(): void {
  manual = true
  if (!full) {
    typed = FULL
    return
  }
  if (!atEnd) {
    cursor = index + 1
    typed = 0
    if (cursor === lines.length - 1 && running) follow = true
    return
  }
  if (st.phase === 'notice') store.skipNotice()
}

function prev(): void {
  if (index === 0) return
  manual = true
  follow = false
  cursor = index - 1
  typed = FULL
}

function first(): void {
  manual = true
  follow = false
  cursor = 0
  typed = FULL
}

function last(): void {
  manual = true
  cursor = lines.length - 1
  typed = FULL
  follow = running
}

function onkeydown(e: KeyboardEvent): void {
  switch (e.key) {
    case 'Enter':
    case ' ':
    case 'ArrowRight':
    case 'ArrowDown':
    case 'PageDown':
      next()
      break
    case 'ArrowLeft':
    case 'ArrowUp':
    case 'PageUp':
      prev()
      break
    case 'Home':
      first()
      break
    case 'End':
      last()
      break
    default:
      return
  }
  e.preventDefault()
}

// the wheel scrolls a line that does not fit first, and pages only from its top or bottom; the
// pause keeps one scroll gesture from running on into the next line
let wheelAt = 0
function onwheel(e: WheelEvent): void {
  if (e.deltaY === 0) return
  const now = performance.now()
  const el = e.currentTarget as HTMLElement
  const up = e.deltaY < 0
  const scrolls = up ? el.scrollTop > 0 : el.scrollTop + el.clientHeight < el.scrollHeight - 1
  if (scrolls || now - wheelAt < 180) {
    wheelAt = now
    return
  }
  wheelAt = now
  if (up) prev()
  else next()
}

// Enter or Space with nothing focused advances the text, as in any VN
function onWindowKey(e: KeyboardEvent): void {
  if (e.defaultPrevented || e.target !== document.body || store.cg || store.dialog || store.screen !== 'main')
    return
  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    next()
  }
}
</script>

<svelte:window onkeydown={onWindowKey} />

<section class="box ink-panel" aria-label={store.t('box.label')}>
  <div class="nameplate" aria-hidden="true">
    <span lang="ja">{store.t('jp.reimu')}</span><span class="dot">·</span><span>{store.t('box.name')}</span>
  </div>

  <!-- the control keeps one name, apart from the arrow's; the current line reaches it as a description
       from the live region below -->
  <div
    class="say {line.kind}"
    role="button"
    tabindex="0"
    aria-label={store.t('box.say')}
    aria-describedby="box-line box-hint"
    bind:this={say}
    onclick={next}
    {onkeydown}
    {onwheel}
  >
    {#if line.status}
      <span class="seal"><Seal status={line.status} size={line.kind === 'verdict' ? 34 : 26} /></span>
    {/if}
    <div class="lines">
      <p class="text" aria-hidden="true">
        {shown}{#if full && !atEnd}<span class="marker">▼</span>{:else if full && running}<span
            class="wait"
            ><span></span><span></span><span></span></span
          >{/if}
      </p>
      {#if line.raw && full}
        <p class="raw mono" aria-hidden="true">{line.raw}</p>
      {/if}
    </div>
  </div>
  <p id="box-line" class="sr-only live" aria-live={manual ? 'polite' : 'off'}>
    {#if line.status}{store.t('box.counter', { n: index + 1, total: lines.length })}.{/if}
    {line.text}
    {#if line.raw}{store.t('box.raw')}: {line.raw}{/if}
  </p>

  <div class="controls">
    <div class="nav">
      <button type="button" class="arrow double left" onclick={first} disabled={index === 0} title={store.t('box.first')} aria-label={store.t('box.first')}></button>
      <button type="button" class="arrow left" onclick={prev} disabled={index === 0} title={store.t('box.prev')} aria-label={store.t('box.prev')}></button>
      <span class="counter" aria-hidden="true">{store.t('box.counter', { n: index + 1, total: lines.length })}</span>
      <button type="button" class="arrow right" onclick={next} disabled={atEnd && full} title={store.t('box.next')} aria-label={store.t('box.next')}></button>
      <button type="button" class="arrow double right" onclick={last} disabled={atEnd} title={store.t('box.last')} aria-label={store.t('box.last')}></button>
      {#if running}<span class="sr-only">{store.t('box.waiting')}</span>{/if}
    </div>
    <span id="box-hint" class="sr-only">{store.t('box.hint')}</span>
    <QuickMenu />
  </div>
</section>

<style>
  .box {
    position: absolute;
    z-index: 4;
    left: var(--gutter);
    right: var(--gutter);
    bottom: var(--gutter);
    height: var(--box-h);
    display: flex;
    flex-direction: column;
    padding: 22px 26px 8px;
    background:
      linear-gradient(180deg, rgb(26 20 23 / 0.9), rgb(16 12 14 / 0.94)),
      radial-gradient(ellipse at 20% 0%, rgb(179 38 30 / 0.12), transparent 60%);
    backdrop-filter: blur(2px);
  }

  .nameplate {
    position: absolute;
    top: -17px;
    left: 22px;
    padding: 3px 16px 4px;
    background: linear-gradient(180deg, #2c1416, #1a0f11);
    border: 1px solid var(--gold);
    box-shadow:
      0 4px 14px rgb(8 5 6 / 0.6),
      inset 0 0 0 2px rgb(20 16 18 / 0.9),
      inset 0 0 0 3px rgb(176 141 87 / 0.3);
    font-size: 16px;
    letter-spacing: 0.08em;
    color: var(--paper);
  }

  .nameplate [lang='ja'] {
    color: var(--vermilion-hi);
  }

  .dot {
    margin: 0 8px;
    color: var(--gold);
  }

  .say {
    flex: 1;
    min-height: 0;
    display: flex;
    gap: 14px;
    align-items: flex-start;
    cursor: pointer;
    overflow-y: auto;
    scrollbar-width: thin;
    scrollbar-color: var(--gold) transparent;
    border-radius: 2px;
  }

  .say:focus-visible {
    outline-offset: 4px;
  }

  .seal {
    flex: none;
    display: grid;
    margin-top: 4px;
  }

  .lines {
    flex: 1;
    min-width: 0;
  }

  .text {
    margin: 0;
    font-size: 19px;
    line-height: 1.62;
    color: var(--paper);
    text-shadow: 0 1px 2px rgb(8 5 6 / 0.8);
    overflow-wrap: anywhere;
  }

  .intro .text,
  .hint .text {
    color: var(--paper-2);
  }

  .notice .text {
    color: var(--gold-hi);
  }

  .verdict .text {
    font-weight: 700;
    color: var(--paper-hi);
  }

  .raw {
    margin: 4px 0 0;
    font-size: 12.5px;
    line-height: 1.45;
    color: var(--paper-dim);
    overflow-wrap: anywhere;
  }

  .marker {
    display: inline-block;
    margin-left: 8px;
    font-size: 13px;
    color: var(--vermilion-hi);
    animation: blink 1.1s steps(1) infinite;
    vertical-align: 2px;
  }

  .wait {
    display: inline-flex;
    gap: 5px;
    margin-left: 10px;
    vertical-align: 3px;
  }

  .wait span {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: var(--gold-hi);
    animation: pulse 1.2s ease-in-out infinite;
  }

  .wait span:nth-child(2) {
    animation-delay: 0.2s;
  }

  .wait span:nth-child(3) {
    animation-delay: 0.4s;
  }

  .controls {
    flex: none;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding-top: 4px;
    border-top: 1px solid rgb(176 141 87 / 0.22);
  }

  .nav {
    display: flex;
    align-items: center;
    gap: 2px;
  }

  .counter {
    min-width: 64px;
    text-align: center;
    font-size: 13px;
    color: var(--gold-hi);
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.06em;
  }

  .arrow {
    position: relative;
    width: 26px;
    height: 26px;
    color: var(--gold-hi);
  }

  .arrow::before,
  .arrow.double::after {
    content: '';
    position: absolute;
    top: 50%;
    left: 50%;
    width: 7px;
    height: 7px;
    border-left: 1.5px solid currentColor;
    border-bottom: 1.5px solid currentColor;
    transform: translate(-30%, -50%) rotate(45deg);
  }

  .arrow.right::before,
  .arrow.right.double::after {
    transform: translate(-70%, -50%) rotate(-135deg);
  }

  .arrow.double.left::before {
    margin-left: -3px;
  }

  .arrow.double.left::after {
    margin-left: 3px;
  }

  .arrow.double.right::before {
    margin-left: -3px;
  }

  .arrow.double.right::after {
    margin-left: 3px;
  }

  .arrow:hover:not(:disabled) {
    color: var(--paper);
  }

  .arrow:disabled {
    opacity: 0.3;
  }

  @keyframes blink {
    50% {
      opacity: 0;
    }
  }

  @keyframes pulse {
    0%,
    100% {
      opacity: 0.25;
    }
    50% {
      opacity: 1;
    }
  }

  @media (max-height: 700px) {
    .box {
      padding: 18px 22px 4px;
    }

    .text {
      font-size: 17.5px;
      line-height: 1.55;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .marker,
    .wait span {
      animation: none;
    }
  }
</style>
