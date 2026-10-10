<!-- always on top in demo mode, so a demo screen can never pass for a real run -->
<script lang="ts">
import { useStore } from '../state.svelte'

const store = useStore()
</script>

<div class="ribbon" aria-hidden="true">
  <span class="word">{store.t('demo.ribbon')}</span>
  <span class="note">{store.t('demo.note', { scenario: store.backend.scenario ?? '' })}</span>
</div>
<p class="sr-only">{store.t('demo.ribbon')}: {store.t('demo.note', { scenario: store.backend.scenario ?? '' })}</p>

<style>
  .ribbon {
    position: fixed;
    z-index: 100;
    top: 26px;
    right: -58px;
    width: 230px;
    padding: 3px 0 4px;
    transform: rotate(38deg);
    display: flex;
    flex-direction: column;
    align-items: center;
    background: repeating-linear-gradient(-45deg, #b3261e 0 10px, #9c1f18 10px 20px);
    border-top: 1px solid var(--gold-hi);
    border-bottom: 1px solid var(--gold-hi);
    box-shadow: 0 4px 16px rgb(8 5 6 / 0.6);
    pointer-events: none;
    color: var(--paper);
    text-align: center;
    line-height: 1.15;
  }

  .word {
    font-size: 15px;
    letter-spacing: 0.3em;
    padding-left: 0.3em;
  }

  .note {
    font-size: 9.5px;
    letter-spacing: 0.04em;
  }
</style>
