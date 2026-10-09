<!-- a full-bleed CG that cross-fades into the next one when src changes -->
<script lang="ts">
import { untrack } from 'svelte'

let {
  src,
  drift = false,
  position = 'center',
}: { src: string; drift?: boolean; position?: string } = $props()

interface Layer {
  src: string
  key: number
}

let key = 0
let layers = $state<Layer[]>([])

// the new CG fades in over the previous one, two layers at most
$effect(() => {
  const next = src
  untrack(() => {
    if (layers.at(-1)?.src === next) return
    layers = [...layers.slice(-1), { src: next, key: ++key }]
  })
})
</script>

<div class="backdrop" aria-hidden="true">
  {#each layers as layer (layer.key)}
    <img
      class="layer"
      class:drift
      class:first={layer.key === 1}
      src={layer.src}
      alt=""
      decoding="async"
      draggable="false"
      style:object-position={position}
    />
  {/each}
</div>

<style>
  .backdrop {
    position: absolute;
    inset: 0;
    overflow: clip;
    background: var(--ink);
  }

  .layer {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: cover;
    user-select: none;
    animation: fade-in 0.9s ease both;
  }

  .layer.first {
    animation-duration: 1.4s;
  }

  .layer.drift {
    animation: fade-in 0.9s ease both;
  }

  @keyframes fade-in {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }


  @media (prefers-reduced-motion: reduce) {
    .layer,
    .layer.drift {
      animation: none;
    }
  }
</style>
