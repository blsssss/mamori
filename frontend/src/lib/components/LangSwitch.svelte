<script lang="ts">
import type { UiKey } from '../i18n'
import { useStore } from '../state.svelte'
import type { Lang } from '../types'

const store = useStore()
const langs: { id: Lang; label: UiKey }[] = [
  { id: 'ru', label: 'lang.ruShort' },
  { id: 'en', label: 'lang.enShort' },
]
</script>

<div class="lang" role="group" aria-label={store.t('top.lang')}>
  {#each langs as l (l.id)}
    <button
      type="button"
      lang={l.id}
      aria-pressed={store.lang === l.id}
      onclick={() => (store.settings.lang = l.id)}>{store.t(l.label)}</button
    >
  {/each}
</div>

<style>
  .lang {
    display: inline-flex;
    border: 1px solid var(--line);
    border-radius: 2px;
    overflow: hidden;
  }

  button {
    min-width: 34px;
    padding: 1px 8px;
    font-size: 12px;
    letter-spacing: 0.08em;
    color: var(--paper-dim);
    background: rgb(20 16 18 / 0.6);
  }

  button + button {
    border-left: 1px solid var(--line);
  }

  button:hover {
    color: var(--paper);
  }

  button[aria-pressed='true'] {
    color: var(--paper);
    background: var(--vermilion);
  }
</style>
