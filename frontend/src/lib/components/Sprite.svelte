<!-- Reimu: all expressions share one body, so the faces are stacked and cross-faded -->
<script lang="ts">
import { EXPRESSIONS, SPRITE } from '../assets'
import { useStore } from '../state.svelte'

const store = useStore()
let el = $state<HTMLDivElement>()
let lastFx = 0

$effect(() => {
  const fx = store.fx
  if (fx?.kind !== 'shake' || fx.seq === lastFx || !el) return
  lastFx = fx.seq
  if (store.reducedMotion || typeof el.animate !== 'function') return
  el.animate(
    [
      { transform: 'translateX(0)' },
      { transform: 'translateX(-14px) rotate(-0.6deg)' },
      { transform: 'translateX(11px) rotate(0.5deg)' },
      { transform: 'translateX(-8px)' },
      { transform: 'translateX(5px)' },
      { transform: 'translateX(0)' },
    ],
    { duration: 460, easing: 'ease-out' },
  )
})
</script>

<div class="sprite" class:still={store.reducedMotion} aria-hidden="true">
  <div class="shake" bind:this={el}>
    <div class="breath">
      {#each EXPRESSIONS as e (e)}
        <img src={SPRITE[e]} alt="" class:on={e === store.expression} decoding="async" draggable="false" />
      {/each}
    </div>
  </div>
</div>

<style>
  .sprite {
    position: absolute;
    top: calc(var(--topbar) + 6px);
    right: max(1vw, 8px);
    height: calc(100vh - var(--topbar) + 40px);
    aspect-ratio: 1000 / 1615;
    pointer-events: none;
    filter: drop-shadow(0 0 18px rgb(12 8 10 / 0.55));
    animation: enter 0.9s ease-out both;
  }

  .shake,
  .breath {
    position: absolute;
    inset: 0;
  }

  .breath {
    transform-origin: 50% 100%;
    animation: breathe 4.6s ease-in-out infinite;
  }

  img {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    opacity: 0;
    transition: opacity 0.22s ease;
    user-select: none;
  }

  img.on {
    opacity: 1;
  }

  .still .breath,
  .still {
    animation: none;
  }

  @keyframes breathe {
    0%,
    100% {
      transform: scale(1);
    }
    50% {
      transform: scale(1.005);
    }
  }

  @keyframes enter {
    from {
      opacity: 0;
      transform: translateX(24px);
    }
    to {
      opacity: 1;
      transform: none;
    }
  }
</style>
