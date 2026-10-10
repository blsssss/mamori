<!-- a full-screen event CG with a caption; it closes by itself unless opened from a thumbnail.
     One opened from a thumbnail takes the focus and gives it back on close, as a dialog does -->
<script lang="ts">
import { CG } from '../assets'
import { useStore } from '../state.svelte'

const store = useStore()
const HOLD_MS = 2600

let closing = $state(false)
let button = $state<HTMLButtonElement>()
let opener: HTMLElement | null = null

// read before the scene under the CG turns inert and drops its focus
$effect.pre(() => {
  if (store.cg?.manual && !opener && document.activeElement instanceof HTMLElement) {
    opener = document.activeElement
  }
})

$effect(() => {
  if (store.cg?.manual) button?.focus()
})

$effect(() => {
  if (store.cg) return
  if (opener?.isConnected) opener.focus()
  opener = null
})

$effect(() => {
  const show = store.cg
  closing = false
  if (!show || show.manual) return
  const timer = setTimeout(close, HOLD_MS)
  return () => clearTimeout(timer)
})

// fades out first, the animation end then shows the next CG in the queue
function close(): void {
  if (store.reducedMotion) store.closeCg()
  else closing = true
}

function onanimationend(e: AnimationEvent): void {
  if (e.animationName.endsWith('cg-out')) store.closeCg()
}
</script>

{#if store.cg}
  {@const caption = store.t(`cg.${store.cg.id}`)}
  {#key store.cg.seq}
    <button
      type="button"
      class="cg"
      class:closing
      class:still={store.reducedMotion}
      aria-label={store.t('cg.aria', { caption })}
      bind:this={button}
      onclick={close}
      {onanimationend}
    >
      <img src={CG[store.cg.id]} alt="" />
      <span class="caption">
        <span class="jp brush" lang="ja">{store.t(`jp.${store.cg.id}`)}</span>
        <span class="text">{caption}</span>
        <span class="hint">{store.t('cg.close')}</span>
      </span>
    </button>
  {/key}
{/if}
<p class="sr-only" role="status">{store.cg ? store.t('cg.aria', { caption: store.t(`cg.${store.cg.id}`) }) : ''}</p>

<style>
  .cg {
    position: fixed;
    inset: 0;
    z-index: 50;
    display: block;
    overflow: clip;
    background: var(--ink);
    cursor: pointer;
    animation: cg-in 0.6s ease both;
  }

  .cg.closing {
    animation: cg-out 0.45s ease both;
  }

  .cg.still,
  .cg.still img {
    animation: none;
  }

  img {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: cover;
    animation: zoom 3.4s ease-out both;
  }

  .caption {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 80px 24px 26px;
    background: linear-gradient(180deg, transparent, rgb(14 10 12 / 0.82) 45%, rgb(14 10 12 / 0.95));
    text-align: center;
  }

  .jp {
    font-size: 46px;
    line-height: 1.1;
    color: var(--paper);
    text-shadow: 0 2px 2px rgb(8 5 6 / 0.9);
  }

  .text {
    font-size: 19px;
    color: var(--paper);
  }

  .hint {
    font-size: 12px;
    color: var(--gold-hi);
    letter-spacing: 0.06em;
  }

  @keyframes cg-in {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  @keyframes cg-out {
    from {
      opacity: 1;
    }
    to {
      opacity: 0;
    }
  }

  @keyframes zoom {
    from {
      transform: scale(1.08);
    }
    to {
      transform: scale(1);
    }
  }
</style>
