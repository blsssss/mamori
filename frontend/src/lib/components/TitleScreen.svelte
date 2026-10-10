<!-- the title screen: key visual, logotype and the VN title menu -->
<script lang="ts">
import { onMount } from 'svelte'
import { CG, SPRITE } from '../assets'
import type { UiKey } from '../i18n'
import { type Screen, useStore } from '../state.svelte'
import Backdrop from './Backdrop.svelte'
import Grain from './Grain.svelte'
import Hanko from './Hanko.svelte'
import LangSwitch from './LangSwitch.svelte'

let { inert = false }: { inert?: boolean } = $props()

const store = useStore()
let menu = $state<HTMLElement>()

interface Item {
  jp: UiKey
  label: UiKey
  act: () => void
  screen?: Screen
}

const go = (screen: Screen) => ({ act: () => store.go(screen), screen })
const items: Item[] = [
  { jp: 'jp.start', label: 'title.start', ...go('main') },
  { jp: 'jp.log', label: 'title.log', ...go('log') },
  { jp: 'jp.config', label: 'title.config', ...go('config') },
  { jp: 'jp.quit', label: 'title.quit', act: () => store.requestQuit() },
]

const version = $derived(
  store.info
    ? `${store.t('title.version', { version: store.info.version })}${store.info.commit ? ` (${store.info.commit})` : ''}`
    : '',
)

onMount(() => {
  menu?.querySelector<HTMLButtonElement>('button')?.focus()
  // the sprite is large: decoded now, Reimu is on screen the moment the scene opens
  const sprite = new Image()
  sprite.src = SPRITE.idle
  sprite.decode?.().catch(() => {})
})

function onkeydown(e: KeyboardEvent): void {
  if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return
  const buttons = [...(menu?.querySelectorAll<HTMLButtonElement>('button') ?? [])]
  const i = buttons.indexOf(document.activeElement as HTMLButtonElement)
  const next = e.key === 'ArrowDown' ? (i + 1) % buttons.length : (i - 1 + buttons.length) % buttons.length
  buttons[next]?.focus()
  e.preventDefault()
}
</script>

<main class="title" {inert}>
  <div class="kv"><Backdrop src={CG.title} position="0% center" drift={!store.reducedMotion} /></div>
  <div class="shade" aria-hidden="true"></div>
  <Grain strength={0.8} />

  <div class="corner"><LangSwitch /></div>

  <div class="content">
    <h1 class="logo">
      <Hanko size="clamp(72px, 11vh, 104px)" />
      <span class="latin">{store.t('app.name')}</span>
    </h1>
    <p class="subtitle">{store.t('title.subtitle')}</p>

    <nav aria-label={store.t('title.menu')} bind:this={menu}>
      <ul>
        {#each items as item, i (item.label)}
          <li style:--i={i}>
            <button type="button" aria-label={store.t(item.label)} data-screen={item.screen} onclick={item.act} {onkeydown}>
              <span class="mark" aria-hidden="true"></span>
              <span class="kanji" lang="ja" aria-hidden="true">{store.t(item.jp)}</span>
              <span class="label">{store.t(item.label)}</span>
            </button>
          </li>
        {/each}
      </ul>
    </nav>
  </div>

  <footer>
    <span class="fan">{store.t('title.fan')}</span>
    {#if version}<span class="version">{version}</span>{/if}
  </footer>
</main>

<style>
  .title {
    position: absolute;
    inset: 0;
    overflow: clip;
  }

  /* the key visual stands right of the text column, so the scrim under the logo and the menu can
     stay dark without darkening Reimu; the masked edge hides where the picture starts */
  .kv {
    position: absolute;
    inset: 0 0 0 12%;
    mask-image: linear-gradient(90deg, transparent, #000 14%);
  }

  .shade {
    position: absolute;
    inset: 0;
    background:
      linear-gradient(
        90deg,
        rgb(14 10 12 / 0.95) 0%,
        rgb(14 10 12 / 0.9) 30%,
        rgb(14 10 12 / 0.86) 42%,
        rgb(14 10 12 / 0.4) 51%,
        transparent 61%
      ),
      linear-gradient(0deg, rgb(14 10 12 / 0.7), transparent 30%);
  }

  .corner {
    position: absolute;
    z-index: 2;
    top: 14px;
    right: calc(var(--gutter) + var(--demo-pad, 0px));
  }

  .content {
    position: absolute;
    z-index: 2;
    left: clamp(32px, 7vw, 96px);
    top: 50%;
    transform: translateY(-54%);
    max-width: 520px;
  }

  /* baseline alignment sits the seal on the baseline of the wordmark, as a stamp beside a signature */
  .logo {
    margin: 0;
    display: flex;
    align-items: baseline;
    gap: 18px;
    font-weight: 400;
    animation: ink-in 1.6s ease-out both;
  }

  .latin {
    font-size: clamp(64px, 10vh, 92px);
    line-height: 0.9;
    letter-spacing: 0.04em;
    color: var(--paper);
    text-shadow: 0 2px 4px rgb(8 5 6 / 0.6);
  }

  .subtitle {
    margin: 14px 0 0;
    padding-top: 12px;
    border-top: 1px solid var(--line);
    font-size: 20px;
    letter-spacing: 0.03em;
    color: var(--paper-2);
    text-shadow: 0 1px 2px rgb(8 5 6 / 0.8);
    animation: fade 1.2s 0.5s ease both;
  }

  ul {
    list-style: none;
    margin: clamp(22px, 5vh, 44px) 0 0;
    padding: 0;
    display: grid;
    gap: 4px;
  }

  li {
    animation: slide 0.7s calc(0.8s + var(--i) * 0.12s) ease both;
  }

  li button {
    position: relative;
    display: flex;
    align-items: baseline;
    gap: 16px;
    width: 300px;
    padding: 7px 14px 7px 34px;
    text-align: left;
    color: var(--paper-2);
    transition:
      color 0.15s,
      background 0.2s;
  }

  .kanji {
    font-size: 15px;
    color: var(--gold);
    letter-spacing: 0.1em;
  }

  .label {
    font-size: 25px;
    letter-spacing: 0.05em;
    text-shadow: 0 1px 2px rgb(8 5 6 / 0.8);
  }

  .mark {
    position: absolute;
    left: 10px;
    top: 50%;
    width: 9px;
    height: 24px;
    background: var(--paper);
    border-top: 3px solid var(--vermilion);
    border-bottom: 3px solid var(--vermilion);
    transform: translate(-8px, -50%) rotate(-8deg);
    opacity: 0;
    transition:
      opacity 0.15s,
      transform 0.2s;
  }

  li button:hover,
  li button:focus-visible {
    color: var(--paper);
    background: linear-gradient(90deg, rgb(179 38 30 / 0.32), transparent 85%);
  }

  li button:hover .mark,
  li button:focus-visible .mark {
    opacity: 1;
    transform: translate(0, -50%) rotate(-8deg);
  }

  li button:focus-visible {
    outline-offset: 0;
  }

  footer {
    position: absolute;
    z-index: 2;
    left: var(--gutter);
    right: var(--gutter);
    bottom: 12px;
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
    gap: 24px;
    font-size: 12px;
    color: var(--paper-dim);
  }

  .version {
    color: var(--gold-hi);
    letter-spacing: 0.04em;
    white-space: nowrap;
  }

  @media (max-width: 1100px) {
    .content {
      left: 40px;
    }
  }

  @keyframes ink-in {
    from {
      opacity: 0;
      filter: blur(6px);
      transform: translateY(6px);
    }
    to {
      opacity: 1;
      filter: none;
      transform: none;
    }
  }

  @keyframes fade {
    from {
      opacity: 0;
    }
  }

  @keyframes slide {
    from {
      opacity: 0;
      transform: translateX(-12px);
    }
  }
</style>
