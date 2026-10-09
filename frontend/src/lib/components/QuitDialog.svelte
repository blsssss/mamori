<script lang="ts">
import { useStore } from '../state.svelte'
import Dialog from './Dialog.svelte'

const store = useStore()
</script>

<Dialog title={store.t('quit.title')}>
  <p role="status">{store.quitting ? store.t('quit.stopping') : store.t('quit.running')}</p>
  <div class="actions">
    <button type="button" class="btn" data-autofocus disabled={store.quitting} onclick={() => (store.dialog = null)}>
      {store.t('quit.cancel')}
    </button>
    <button type="button" class="btn primary" disabled={store.quitting} onclick={() => store.quit()}>
      {store.t('quit.ok')}
    </button>
  </div>
</Dialog>

<style>
  p {
    margin: 0 0 20px;
    color: var(--paper-2);
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }
</style>
