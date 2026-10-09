<script lang="ts">
import { reportName } from '../report'
import { useStore } from '../state.svelte'
import type { ReportFormat } from '../types'
import Dialog from './Dialog.svelte'

const store = useStore()
const formats: ReportFormat[] = ['html', 'txt', 'json']
let format = $state<ReportFormat>(store.settings.reportFormat)
</script>

<Dialog title={store.t('save.title')}>
  <fieldset>
    <legend>{store.t('save.format')}</legend>
    {#each formats as f (f)}
      <label class:on={format === f}>
        <input type="radio" name="report-format" value={f} bind:group={format} data-autofocus={format === f ? '' : undefined} />
        <span>{store.t(`save.${f}`)}</span>
      </label>
    {/each}
  </fieldset>
  <p class="name">{store.t('save.name', { name: reportName(format) })}</p>
  <div class="actions">
    <button type="button" class="btn" onclick={() => (store.dialog = null)}>{store.t('save.cancel')}</button>
    <button type="button" class="btn primary" onclick={() => store.saveReport(format)}>{store.t('save.ok')}</button>
  </div>
</Dialog>

<style>
  fieldset {
    margin: 0;
    padding: 0;
    border: 0;
    display: grid;
    gap: 6px;
  }

  legend {
    margin-bottom: 8px;
    font-size: 13px;
    color: var(--gold-hi);
  }

  label {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 7px 10px;
    border: 1px solid rgb(176 141 87 / 0.3);
    cursor: pointer;
  }

  label.on {
    border-color: var(--gold-hi);
    background: rgb(179 38 30 / 0.16);
  }

  input {
    accent-color: var(--vermilion);
    width: 16px;
    height: 16px;
    margin: 0;
  }

  .name {
    margin: 14px 0 18px;
    font-family: var(--font-mono);
    font-size: 12.5px;
    color: var(--paper-2);
    overflow-wrap: anywhere;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }
</style>
