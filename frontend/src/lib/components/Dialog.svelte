<!-- a small modal panel: focus moves in on open, Tab stays inside, focus returns on close.
     Escape is handled once for the whole app in App.svelte -->
<script lang="ts">
import type { Snippet } from 'svelte'

let { title, children }: { title: string; children: Snippet } = $props()
const id = $props.id()
let panel = $state<HTMLDivElement>()

const FOCUSABLE =
  'button:not(:disabled), input:not(:disabled), textarea, select, [tabindex]:not([tabindex="-1"])'

$effect(() => {
  const opener = document.activeElement
  const target =
    panel?.querySelector<HTMLElement>('[data-autofocus]') ?? panel?.querySelector<HTMLElement>(FOCUSABLE)
  target?.focus()
  return () => {
    if (opener instanceof HTMLElement && opener.isConnected) opener.focus()
  }
})

function onkeydown(e: KeyboardEvent): void {
  if (e.key !== 'Tab' || !panel) return
  const items = [...panel.querySelectorAll<HTMLElement>(FOCUSABLE)]
  if (items.length === 0) return
  const first = items[0]
  const last = items[items.length - 1]
  if (e.shiftKey && document.activeElement === first) {
    e.preventDefault()
    last.focus()
  } else if (!e.shiftKey && document.activeElement === last) {
    e.preventDefault()
    first.focus()
  }
}
</script>

<div class="veil">
  <div class="dialog ink-panel" role="dialog" aria-modal="true" aria-labelledby={id} bind:this={panel} tabindex="-1" {onkeydown}>
    <h2 {id}>{title}</h2>
    {@render children()}
  </div>
</div>

<style>
  .veil {
    position: fixed;
    inset: 0;
    z-index: 40;
    display: grid;
    place-items: center;
    padding: var(--gutter);
    background: rgb(10 6 8 / 0.62);
    animation: veil 0.2s ease both;
  }

  .dialog {
    width: min(460px, 100%);
    padding: 22px 26px 22px;
    animation: rise 0.25s ease both;
  }

  h2 {
    margin: 0 0 14px;
    font-size: 21px;
    font-weight: 400;
    color: var(--paper);
    letter-spacing: 0.04em;
  }

  @keyframes veil {
    from {
      opacity: 0;
    }
  }

  @keyframes rise {
    from {
      opacity: 0;
      transform: translateY(8px);
    }
  }
</style>
