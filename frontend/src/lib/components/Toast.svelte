<script lang="ts">
import { useStore } from '../state.svelte'

const store = useStore()
let visible = $state(false)

$effect(() => {
  if (!store.toast) return
  void store.toast.seq
  visible = true
  const timer = setTimeout(() => (visible = false), store.toast.error ? 7000 : 4500)
  return () => clearTimeout(timer)
})
</script>

<div class="toast-region" role="status" aria-live="polite">
  {#if store.toast && visible}
    {#key store.toast.seq}
      <p class="toast" class:error={store.toast.error}>{store.toast.text}</p>
    {/key}
  {/if}
</div>

<style>
  .toast-region {
    position: fixed;
    z-index: 45;
    top: calc(var(--topbar) + 12px);
    left: 50%;
    transform: translateX(-50%);
    width: min(620px, calc(100vw - 2 * var(--gutter)));
    pointer-events: none;
    display: grid;
    justify-items: center;
  }

  .toast {
    margin: 0;
    padding: 8px 18px;
    background: var(--panel);
    border: 1px solid var(--gold);
    box-shadow: var(--shadow);
    color: var(--paper);
    font-size: 14px;
    text-align: center;
    overflow-wrap: anywhere;
    animation: drop 0.25s ease both;
  }

  .toast.error {
    border-color: var(--vermilion-hi);
  }

  @keyframes drop {
    from {
      opacity: 0;
      transform: translateY(-6px);
    }
  }
</style>
