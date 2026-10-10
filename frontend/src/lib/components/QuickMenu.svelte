<!-- the classic VN quick menu; the labels stay in English small caps, the hints are localized -->
<script lang="ts">
import type { UiKey } from '../i18n'
import { useStore } from '../state.svelte'

const store = useStore()

interface Item {
  label: UiKey
  hint: UiKey
  act: () => void
  disabled?: () => boolean
  accent?: boolean
}

const items: Item[] = [
  {
    label: 'vn.auto',
    hint: 'qm.auto',
    act: () => void store.runAll(),
    disabled: () => store.busy,
    accent: true,
  },
  { label: 'vn.stop', hint: 'qm.stop', act: () => void store.stop(), disabled: () => !store.busy },
  { label: 'vn.log', hint: 'qm.log', act: () => store.go('log') },
  {
    label: 'vn.save',
    hint: 'qm.save',
    act: () => (store.dialog = 'save'),
    disabled: () => store.results.length === 0,
  },
  { label: 'vn.config', hint: 'qm.config', act: () => store.go('config') },
  { label: 'vn.title', hint: 'qm.title', act: () => store.go('title') },
  { label: 'vn.quit', hint: 'qm.quit', act: () => store.requestQuit() },
]
</script>

<nav class="qm" aria-label={store.t('qm.label')}>
  {#each items as item (item.label)}
    <button
      type="button"
      class:accent={item.accent}
      title={store.t(item.hint)}
      aria-label={`${store.t(item.label)}. ${store.t(item.hint)}`}
      disabled={item.disabled?.() ?? false}
      onclick={item.act}
    >
      {store.t(item.label)}
    </button>
  {/each}
</nav>

<style>
  .qm {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 2px;
  }

  button {
    position: relative;
    padding: 3px 9px;
    font-variant: small-caps;
    font-size: 15px;
    letter-spacing: 0.12em;
    color: var(--gold-hi);
    transition: color 0.15s;
  }

  button::after {
    content: '';
    position: absolute;
    left: 9px;
    right: 9px;
    bottom: 2px;
    height: 1px;
    background: var(--vermilion-hi);
    transform: scaleX(0);
    transition: transform 0.2s ease;
  }

  button:hover:not(:disabled),
  button:focus-visible {
    color: var(--paper);
  }

  button:hover:not(:disabled)::after {
    transform: scaleX(1);
  }

  .accent {
    color: var(--vermilion-hi);
  }

  button:disabled {
    color: var(--paper-dim);
    opacity: 0.45;
  }
</style>
