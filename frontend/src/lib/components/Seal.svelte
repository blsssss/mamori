<!-- a hanko stamp for a status; empty frame before a run, a turning ofuda while it runs.
     Decorative: the status word next to it carries the meaning -->
<script lang="ts">
import { SEAL } from '../checks'
import type { Status } from '../types'

let {
  status,
  running = false,
  size = 36,
}: { status: Status | null; running?: boolean; size?: number } = $props()
</script>

<span
  class="seal {running ? 'running' : (status ?? 'none')}"
  style:--size="{size}px"
  aria-hidden="true"
>
  {#if running}
    <span class="ofuda"><span class="mark"></span></span>
  {:else if status}
    <span class="kanji" lang="ja">{SEAL[status]}</span>
  {/if}
</span>

<style>
  .seal {
    --c: var(--gold);
    position: relative;
    flex: none;
    display: inline-grid;
    place-items: center;
    width: var(--size);
    height: var(--size);
    border-radius: calc(var(--size) * 0.14);
    transform: rotate(-4deg);
  }

  .kanji {
    display: grid;
    place-items: center;
    width: 100%;
    height: 100%;
    border-radius: inherit;
    background: var(--c);
    color: var(--paper);
    font-size: calc(var(--size) * 0.64);
    line-height: 1;
    box-shadow:
      inset 0 0 0 calc(var(--size) * 0.06) rgb(239 230 216 / 0.82),
      inset 0 0 0 calc(var(--size) * 0.1) var(--c);
    filter: url(#ink-edge);
  }

  .pass {
    --c: var(--st-pass);
  }

  .warn {
    --c: var(--st-warn);
  }

  .warn .kanji {
    color: var(--ink);
    box-shadow:
      inset 0 0 0 calc(var(--size) * 0.06) rgb(20 16 18 / 0.75),
      inset 0 0 0 calc(var(--size) * 0.1) var(--c);
  }

  .fail {
    --c: var(--st-fail);
  }

  .skip {
    --c: var(--st-skip);
  }

  .error {
    --c: var(--st-error);
  }

  .error .kanji {
    outline: 1px solid var(--vermilion-hi);
    outline-offset: 1px;
  }

  .none {
    border: 1.5px dashed rgb(176 141 87 / 0.6);
    transform: none;
  }

  .running {
    border: 1.5px dashed rgb(212 180 124 / 0.75);
    border-radius: 50%;
    transform: none;
  }

  .ofuda {
    position: relative;
    width: calc(var(--size) * 0.3);
    height: calc(var(--size) * 0.74);
    background: var(--paper);
    border-radius: 1px;
    box-shadow: 0 1px 2px rgb(8 5 6 / 0.6);
    animation: spin 1.8s linear infinite;
  }

  .mark {
    position: absolute;
    inset: 18% 30%;
    border-top: 2px solid var(--vermilion);
    border-bottom: 2px solid var(--vermilion);
    background: linear-gradient(var(--vermilion), var(--vermilion)) center / 2px 100% no-repeat;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .ofuda {
      animation: none;
      transform: rotate(-12deg);
    }
  }
</style>
