<script lang="ts">
import { onMount } from 'svelte'
import ConfigScreen from './lib/components/ConfigScreen.svelte'
import DemoRibbon from './lib/components/DemoRibbon.svelte'
import EventCg from './lib/components/EventCg.svelte'
import InkDefs from './lib/components/InkDefs.svelte'
import LogScreen from './lib/components/LogScreen.svelte'
import QuitDialog from './lib/components/QuitDialog.svelte'
import SaveDialog from './lib/components/SaveDialog.svelte'
import SceneScreen from './lib/components/SceneScreen.svelte'
import TitleScreen from './lib/components/TitleScreen.svelte'
import Toast from './lib/components/Toast.svelte'
import { trackInput } from './lib/input'
import { type Screen, useStore } from './lib/state.svelte'

const store = useStore()

const overlay = $derived(store.screen === 'log' || store.screen === 'config')
// the screen under an open log or config stays mounted, so the text box keeps its place
const base = $derived(overlay ? store.returnTo : store.screen)
const demo = $derived(store.backend.kind === 'demo')

let shell = $state<HTMLDivElement>()
let opened: { screen: Screen; opener: HTMLElement | null } | null = null

// read before the screen under the log or the config turns inert and drops its focus
$effect.pre(() => {
  if (!overlay || opened) return
  const el = document.activeElement
  opened = { screen: store.screen, opener: el instanceof HTMLElement && el !== document.body ? el : null }
})

// the focus goes back where the screen was opened from, or else to the selected chapter or the
// title menu entry of the closed screen, never to the body where arrows and Tab start over
$effect(() => {
  if (overlay || !opened) return
  const { screen, opener } = opened
  opened = null
  const fallback = base === 'title' ? `[data-screen="${screen}"]` : `[data-check="${store.selected}"]`
  const target = opener?.isConnected ? opener : shell?.querySelector<HTMLElement>(fallback)
  target?.focus()
})

$effect(() => {
  store.persist()
})

$effect(() => {
  document.documentElement.lang = store.lang
})

onMount(() => {
  const mq =
    typeof window.matchMedia === 'function' ? window.matchMedia('(prefers-reduced-motion: reduce)') : null
  const sync = () => {
    store.reducedMotion = mq?.matches ?? false
  }
  sync()
  mq?.addEventListener('change', sync)
  const stopInput = trackInput()
  void store.init()
  return () => {
    stopInput()
    mq?.removeEventListener('change', sync)
    store.destroy()
  }
})

function onkeydown(e: KeyboardEvent): void {
  if (store.cg) {
    // any key closes an event CG; modifiers alone do not, so shortcuts keep working
    if (['Shift', 'Control', 'Alt', 'Meta'].includes(e.key)) return
    if (e.key === 'Enter' || e.key === ' ' || e.key === 'Escape') e.preventDefault()
    e.stopImmediatePropagation()
    store.closeCg()
    return
  }
  if (e.key !== 'Escape') return
  if (store.dialog) {
    if (!store.quitting) store.dialog = null
    e.preventDefault()
  } else if (overlay) {
    store.back()
    e.preventDefault()
  }
}
</script>

<svelte:window {onkeydown} />

<InkDefs />

<!-- a CG opened from a thumbnail is modal until closed, an automatic one never takes the focus -->
<div class="app" class:demo inert={store.cg?.manual === true} bind:this={shell}>
  {#if base === 'title'}
    <TitleScreen inert={overlay || store.dialog !== null} />
  {:else}
    <SceneScreen inert={overlay || store.dialog !== null} />
  {/if}
  {#if store.screen === 'log'}
    <LogScreen />
  {:else if store.screen === 'config'}
    <ConfigScreen />
  {/if}
</div>

{#if store.dialog === 'save'}
  <SaveDialog />
{:else if store.dialog === 'quit'}
  <QuitDialog />
{/if}

<EventCg />
<Toast />
<div class="sr-only" aria-live="polite">{store.announcement}</div>

{#if store.backend.kind === 'none'}
  <p class="missing" role="alert">{store.t('app.unavailable')}</p>
{/if}
{#if demo}
  <DemoRibbon />
{/if}

<style>
  .app {
    position: fixed;
    inset: 0;
    overflow: clip;
  }

  .app.demo {
    --demo-pad: 128px;
  }

  .missing {
    position: fixed;
    z-index: 60;
    left: 50%;
    bottom: 24px;
    transform: translateX(-50%);
    margin: 0;
    padding: 8px 16px;
    background: var(--blood);
    border: 1px solid var(--vermilion-hi);
    color: var(--paper);
  }
</style>
