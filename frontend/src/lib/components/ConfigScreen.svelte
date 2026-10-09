<!-- the VN config screen; every change applies and is saved at once -->
<script lang="ts">
import { BG } from '../assets'
import type { UiKey } from '../i18n'
import { DEFAULTS, EICAR_WAIT_MAX, EICAR_WAIT_MIN, policyList, TEXT_SPEED_MAX } from '../settings'
import { useStore } from '../state.svelte'
import type { Lang } from '../types'
import Backdrop from './Backdrop.svelte'
import Grain from './Grain.svelte'

const store = useStore()
const id = $props.id()
let back = $state<HTMLButtonElement>()

const sample = $derived(store.t('cfg.speedSample'))
let typed = $state(0)

// a live preview of the text speed
$effect(() => {
  const ms = store.charMs
  const len = sample.length
  typed = 0
  if (ms <= 0) {
    typed = len
    return
  }
  const timer = setInterval(() => {
    typed += 1
    if (typed >= len) clearInterval(timer)
  }, ms)
  return () => clearInterval(timer)
})

const targets = $derived(policyList(store.settings.policyTargets).length)
const langs: { id: Lang; label: UiKey }[] = [
  { id: 'ru', label: 'lang.ru' },
  { id: 'en', label: 'lang.en' },
]

$effect(() => {
  back?.focus()
})

// the language stays: a reset should not switch the screen to another language under the user
function reset(): void {
  Object.assign(store.settings, { ...structuredClone(DEFAULTS), lang: store.settings.lang })
}
</script>

<section class="config" aria-labelledby={id}>
  <Backdrop src={BG.shrine} />
  <Grain strength={0.7} />

  <div class="frame ink-panel">
    <header>
      <span class="kanji brush" lang="ja" aria-hidden="true">{store.t('jp.config')}</span>
      <h1 {id}>{store.t('cfg.title')}</h1>
      <div class="actions">
        <button type="button" class="btn" onclick={reset}>{store.t('cfg.reset')}</button>
        <button type="button" class="btn primary" bind:this={back} onclick={() => store.back()}>
          {store.t('cfg.back')}
        </button>
      </div>
    </header>

    <div class="body scroll">
      {#if !store.persistent}
        <p class="warn">{store.t('cfg.storage')}</p>
      {/if}
      <div class="columns">
        <fieldset>
          <legend>{store.t('cfg.ui')}</legend>

          <div class="field">
            <span class="label" id="{id}-lang">{store.t('cfg.lang')}</span>
            <div class="seg" role="radiogroup" aria-labelledby="{id}-lang">
              {#each langs as l (l.id)}
                <label class:on={store.settings.lang === l.id}>
                  <input type="radio" name="lang" value={l.id} bind:group={store.settings.lang} />
                  <span lang={l.id}>{store.t(l.label)}</span>
                </label>
              {/each}
            </div>
          </div>

          <div class="field">
            <label class="label" for="{id}-speed">{store.t('cfg.speed')}</label>
            <div class="range">
              <span class="end">{store.t('cfg.speedInstant')}</span>
              <input
                id="{id}-speed"
                type="range"
                min="0"
                max={TEXT_SPEED_MAX}
                step="1"
                bind:value={store.settings.textSpeed}
                aria-valuetext={store.settings.textSpeed === 0
                  ? store.t('cfg.speedInstant')
                  : store.t('cfg.speedValue', { n: store.settings.textSpeed, max: TEXT_SPEED_MAX })}
              />
              <span class="end">{store.t('cfg.speedSlow')}</span>
            </div>
            <p class="sample" aria-hidden="true">{sample.slice(0, typed)}<span class="caret">▼</span></p>
            {#if store.reducedMotion}<p class="hint">{store.t('cfg.reduced')}</p>{/if}
          </div>

          <div class="field">
            <label class="check">
              <input type="checkbox" bind:checked={store.settings.eventCgs} />
              <span>{store.t('cfg.cgs')}</span>
            </label>
            <p class="hint">{store.t('cfg.cgsHint')}</p>
          </div>

          <div class="field">
            <label class="check">
              <input type="checkbox" bind:checked={store.settings.skipTitle} />
              <span>{store.t('cfg.skipTitle')}</span>
            </label>
          </div>
        </fieldset>

        <fieldset>
          <legend>{store.t('cfg.checks')}</legend>

          <div class="field">
            <label class="label" for="{id}-policy">{store.t('cfg.policy')}</label>
            <textarea
              id="{id}-policy"
              rows="4"
              spellcheck="false"
              autocomplete="off"
              placeholder={store.t('cfg.policyPlaceholder')}
              aria-describedby="{id}-policy-hint"
              bind:value={store.settings.policyTargets}
            ></textarea>
            <p class="hint" id="{id}-policy-hint">
              {store.t('cfg.policyHint')}
              <span class="count">{store.t('cfg.policyCount', { n: targets })}</span>
            </p>
          </div>

          <div class="field">
            <label class="label" for="{id}-eicar">{store.t('cfg.eicar')}</label>
            <div class="range">
              <span class="end">{EICAR_WAIT_MIN}</span>
              <input
                id="{id}-eicar"
                type="range"
                min={EICAR_WAIT_MIN}
                max={EICAR_WAIT_MAX}
                step="1"
                bind:value={store.settings.eicarWaitSec}
                aria-valuetext={store.t('cfg.eicarValue', { n: store.settings.eicarWaitSec })}
                aria-describedby="{id}-eicar-hint"
              />
              <span class="end">{EICAR_WAIT_MAX}</span>
              <output for="{id}-eicar">{store.t('cfg.eicarValue', { n: store.settings.eicarWaitSec })}</output>
            </div>
            <p class="hint" id="{id}-eicar-hint">{store.t('cfg.eicarHint')}</p>
          </div>

          <div class="field">
            <label class="check">
              <input type="checkbox" bind:checked={store.settings.allowElevation} aria-describedby="{id}-uac-hint" />
              <span>{store.t('cfg.uac')}</span>
            </label>
            <p class="hint" id="{id}-uac-hint">{store.t('cfg.uacHint')}</p>
          </div>
        </fieldset>
      </div>
    </div>
  </div>
</section>

<style>
  .config {
    position: absolute;
    inset: 0;
    z-index: 20;
    overflow: clip;
  }

  .frame {
    position: absolute;
    inset: var(--gutter);
    display: flex;
    flex-direction: column;
    background: rgb(18 14 16 / 0.9);
  }

  header {
    flex: none;
    display: flex;
    align-items: center;
    gap: 18px;
    padding: 14px calc(22px + var(--demo-pad, 0px)) 12px 22px;
    border-bottom: 1px solid var(--line);
  }

  .kanji {
    font-size: 54px;
    line-height: 1;
    color: var(--vermilion-hi);
  }

  h1 {
    flex: 1;
    margin: 0;
    font-size: 26px;
    font-weight: 400;
    letter-spacing: 0.04em;
  }

  .actions {
    display: flex;
    gap: 8px;
  }

  .body {
    flex: 1;
    min-height: 0;
    padding: 16px 22px 22px;
  }

  .warn {
    margin: 0 0 12px;
    padding: 6px 12px;
    border-left: 3px solid var(--st-warn);
    background: rgb(199 146 50 / 0.12);
    font-size: 14px;
  }

  .columns {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 22px 32px;
  }

  fieldset {
    margin: 0;
    padding: 0;
    border: 0;
    min-width: 0;
  }

  legend {
    width: 100%;
    margin-bottom: 10px;
    padding-bottom: 4px;
    border-bottom: 1px solid rgb(176 141 87 / 0.35);
    font-size: 18px;
    color: var(--gold-hi);
    letter-spacing: 0.04em;
  }

  .field {
    margin-bottom: 16px;
  }

  .label {
    display: block;
    margin-bottom: 6px;
    font-size: 15px;
    color: var(--paper);
  }

  .seg {
    display: inline-flex;
    border: 1px solid var(--line);
  }

  .seg label {
    position: relative;
    padding: 5px 16px;
    cursor: pointer;
    color: var(--paper-2);
  }

  .seg label + label {
    border-left: 1px solid var(--line);
  }

  .seg label.on {
    background: var(--vermilion);
    color: var(--paper);
  }

  .seg input {
    position: absolute;
    opacity: 0;
    inset: 0;
    margin: 0;
    cursor: pointer;
  }

  .seg label:has(input:focus-visible) {
    outline: 2px solid var(--gold-hi);
    outline-offset: 2px;
  }

  .range {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .end,
  output {
    font-size: 12.5px;
    color: var(--paper-dim);
    white-space: nowrap;
  }

  output {
    min-width: 44px;
    color: var(--gold-hi);
    font-size: 14px;
  }

  input[type='range'] {
    flex: 1;
    min-width: 0;
    accent-color: var(--vermilion);
  }

  .sample {
    margin: 8px 0 0;
    min-height: 1.6em;
    padding: 6px 12px;
    border: 1px solid rgb(176 141 87 / 0.25);
    background: rgb(20 16 18 / 0.6);
    font-size: 16px;
  }

  .caret {
    margin-left: 6px;
    font-size: 11px;
    color: var(--vermilion-hi);
  }

  .check {
    display: flex;
    align-items: center;
    gap: 10px;
    cursor: pointer;
    font-size: 15px;
  }

  .check input {
    width: 18px;
    height: 18px;
    margin: 0;
    accent-color: var(--vermilion);
    flex: none;
  }

  .hint {
    margin: 6px 0 0;
    font-size: 13px;
    line-height: 1.5;
    color: var(--paper-dim);
  }

  .count {
    display: inline-block;
    margin-left: 6px;
    color: var(--gold-hi);
  }

  textarea {
    display: block;
    width: 100%;
    resize: vertical;
    min-height: 92px;
    padding: 8px 10px;
    background: rgb(12 9 10 / 0.75);
    border: 1px solid var(--line);
    color: var(--paper);
    font-family: var(--font-mono);
    font-size: 13px;
    line-height: 1.5;
  }

  textarea::placeholder {
    color: rgb(180 168 147 / 0.55);
  }

  textarea:focus-visible {
    outline-offset: 1px;
    border-color: var(--gold-hi);
  }

  @media (max-width: 1040px) {
    .columns {
      gap: 18px 22px;
    }
  }
</style>
