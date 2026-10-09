<!-- module 5, results output: the overall verdict and a backlog of every check -->
<script lang="ts">
import { BG, CG } from '../assets'
import { CHECKS } from '../checks'
import { duration, statusWord } from '../i18n'
import { checkTitle, formatTime, systemLine } from '../report'
import { type EventCg, useStore } from '../state.svelte'
import { CHECK_IDS, checkStatus } from '../types'
import Backdrop from './Backdrop.svelte'
import Grain from './Grain.svelte'
import Seal from './Seal.svelte'

const store = useStore()
const id = $props.id()
let confirming = $state(false)
let back = $state<HTMLButtonElement>()

const summary = $derived(store.overall)
const hasResults = $derived(store.results.length > 0)
const gallery: EventCg[] = ['exorcism', 'barrier', 'tea']

// results that change while the log is open are summed up again, once per version
let tried = -1
$effect(() => {
  const v = store.version
  if (store.summaryVersion !== v && tried !== v) {
    tried = v
    void store.summarize()
  }
})

$effect(() => {
  back?.focus()
})

function clear(): void {
  confirming = false
  store.clearResults()
}
</script>

<section class="log" aria-labelledby={id}>
  <Backdrop src={BG.veranda} />
  <Grain strength={0.6} />

  <div class="frame ink-panel">
    <header>
      <span class="kanji brush" lang="ja" aria-hidden="true">{store.t('jp.log')}</span>
      <div class="heading">
        <h1 {id}>{store.t('log.title')}</h1>
        <p>{store.t('log.subtitle')}</p>
      </div>
      <div class="actions">
        <button type="button" class="btn primary" disabled={!hasResults} onclick={() => (store.dialog = 'save')}>
          {store.t('log.save')}
        </button>
        <button type="button" class="btn" disabled={!hasResults} onclick={() => store.copyReport()}>
          {store.t('log.copy')}
        </button>
        {#if confirming}
          <span class="confirm" role="group" aria-label={store.t('log.clearConfirm')}>
            <span>{store.t('log.clearConfirm')}</span>
            <button type="button" class="btn danger" onclick={clear}>{store.t('log.clearYes')}</button>
            <button type="button" class="btn" onclick={() => (confirming = false)}>{store.t('log.clearNo')}</button>
          </span>
        {:else}
          <button
            type="button"
            class="btn"
            disabled={!hasResults || store.busy}
            onclick={() => (confirming = true)}
          >
            {store.t('log.clear')}
          </button>
        {/if}
        <button type="button" class="btn" bind:this={back} onclick={() => store.back()}>{store.t('log.back')}</button>
      </div>
    </header>

    <div class="body scroll">
      <section class="overall {summary?.status ?? 'none'}">
        <Seal status={summary?.status ?? null} size={76} />
        <div class="verdict">
          <h2>
            {store.t('log.overall')}{#if summary}:
              <span class="word">{statusWord(store.lang, summary.status)}</span>{/if}
          </h2>
          <!-- only the verdict sentence is read out when it changes, not the gallery or the meta line -->
          <div aria-live="polite">
            {#if summary}<p class="sentence">{store.code(summary.code)}</p>{/if}
          </div>
          {#if summary}
            <p class="meta">
              <span>{systemLine(store.lang, summary.system)}</span>
              {#if summary.system.host}<span>{store.t('top.host', { host: summary.system.host })}</span>{/if}
              <span>{store.t('log.generated', { date: formatTime(summary.generated) })}</span>
              <span>{store.t('log.checksRun', { n: summary.results.length, total: CHECK_IDS.length })}</span>
            </p>
          {:else}
            <p class="sentence muted">{store.t('log.loading')}</p>
          {/if}
        </div>
        <div class="gallery" role="group" aria-label={store.t('log.cgs')}>
          {#each gallery as cg (cg)}
            {#if store.unlocked[cg]}
              <button
                type="button"
                class="cg"
                title={store.t(`cg.${cg}`)}
                aria-label={store.t('chapter.cg', { caption: store.t(`cg.${cg}`) })}
                onclick={() => store.showCg(cg, true)}
              >
                <img src={CG[cg]} alt="" />
              </button>
            {:else}
              <span class="cg locked" title={store.t('log.cgLocked')}>
                <span lang="ja" aria-hidden="true">{store.t('jp.locked')}</span>
                <span class="sr-only">{store.t('log.cgLocked')}</span>
              </span>
            {/if}
          {/each}
        </div>
      </section>

      {#each CHECKS as c (c.id)}
        {@const st = store.checks[c.id]}
        {@const status = checkStatus(st)}
        {@const findings = st.result?.findings ?? st.live}
        <section class="entry">
          <h3>
            <span class="num brush" lang="ja" aria-hidden="true">{c.numeral}</span>
            <span class="module">{store.t('check.module', { n: c.module })}.</span>
            <span>{checkTitle(store.lang, c.id)}</span>
            <span class="jp" lang="ja">{c.jp}</span>
          </h3>
          {#if st.phase === 'idle'}
            <p class="muted">{store.t('log.notRun')}</p>
          {:else}
            {#if st.result}
              <p class="line verdict-line">
                <Seal {status} size={30} />
                <span class="word {status}">{statusWord(store.lang, st.result.status)}</span>
                <span class="text">{store.code(st.result.code, st.result.params)}</span>
                <span class="time">{store.t('log.elapsed', { time: duration(store.lang, st.result.elapsedMs) })}</span>
              </p>
            {:else if st.error !== null}
              <p class="line verdict-line">
                <Seal status="error" size={30} />
                <span class="word error">{statusWord(store.lang, 'error')}</span>
                <span class="text">{store.t('say.runError')}<span class="raw mono">{st.error}</span></span>
              </p>
            {:else}
              <p class="muted">{store.t('log.running')}</p>
            {/if}
            {#if findings.length > 0}
              <ol class="findings">
                {#each findings as f, i (i)}
                  <li class="line">
                    <Seal status={f.status} size={22} />
                    <span class="word {f.status}">{statusWord(store.lang, f.status)}</span>
                    <span class="text">
                      {store.code(f.code, f.params)}
                      {#if f.params?.error}<span class="raw mono">{f.params.error}</span>{/if}
                    </span>
                  </li>
                {/each}
              </ol>
            {/if}
          {/if}
        </section>
      {/each}
    </div>
  </div>
</section>

<style>
  .log {
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

  /* at the minimum window size the buttons go to a row of their own instead of squeezing the title */
  header {
    flex: none;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 10px 18px;
    padding: 14px calc(22px + var(--demo-pad, 0px)) 12px 22px;
    border-bottom: 1px solid var(--line);
  }

  .kanji {
    font-size: 54px;
    line-height: 1;
    color: var(--vermilion-hi);
  }

  .heading {
    flex: 1 1 220px;
    min-width: 0;
  }

  h1 {
    margin: 0;
    font-size: 26px;
    font-weight: 400;
    letter-spacing: 0.04em;
  }

  .heading p {
    margin: 0;
    color: var(--gold-hi);
    font-size: 14px;
  }

  .actions {
    flex: 1 1 auto;
    min-width: 0;
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    align-items: center;
    gap: 8px;
  }

  .confirm {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    color: var(--paper);
  }

  .btn.danger {
    border-color: var(--vermilion-hi);
    background: var(--blood);
  }

  .body {
    flex: 1;
    min-height: 0;
    padding: 6px 22px 22px;
  }

  .overall {
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: center;
    gap: 22px;
    padding: 18px 6px 18px;
    border-bottom: 1px solid rgb(176 141 87 / 0.3);
  }

  h2 {
    margin: 0;
    font-size: 15px;
    font-weight: 400;
    color: var(--gold-hi);
    letter-spacing: 0.04em;
  }

  .sentence {
    margin: 4px 0 6px;
    font-size: 22px;
    line-height: 1.4;
    font-weight: 700;
    color: var(--paper-hi);
  }

  .meta {
    margin: 0;
    display: flex;
    flex-wrap: wrap;
    gap: 4px 16px;
    font-size: 12.5px;
    color: var(--paper-dim);
  }

  .gallery {
    display: flex;
    gap: 8px;
  }

  .cg {
    display: grid;
    place-items: center;
    width: 84px;
    height: 56px;
    border: 1px solid var(--gold);
    overflow: hidden;
    padding: 0;
  }

  .cg img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  button.cg:hover {
    border-color: var(--paper);
  }

  .cg.locked {
    border-style: dashed;
    border-color: rgb(176 141 87 / 0.45);
    color: rgb(176 141 87 / 0.6);
    font-size: 22px;
  }

  .entry {
    padding: 16px 6px 6px;
    border-bottom: 1px solid rgb(176 141 87 / 0.18);
  }

  h3 {
    margin: 0 0 8px;
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: 10px;
    font-size: 18px;
    font-weight: 400;
  }

  h3 .num {
    font-size: 30px;
    line-height: 1;
    color: var(--vermilion-hi);
  }

  .module {
    color: var(--gold-hi);
  }

  .jp {
    font-size: 13px;
    color: var(--gold);
    letter-spacing: 0.1em;
  }

  .findings {
    list-style: none;
    margin: 6px 0 4px;
    padding: 0 0 0 4px;
    display: grid;
    gap: 6px;
  }

  .line {
    display: grid;
    grid-template-columns: auto 104px 1fr auto;
    align-items: start;
    gap: 12px;
    margin: 0;
    font-size: 15px;
    line-height: 1.5;
  }

  .verdict-line {
    margin-bottom: 6px;
    font-weight: 700;
    align-items: center;
  }

  .verdict-line .text {
    color: var(--paper-hi);
  }

  .findings .line {
    padding-left: 4px;
    color: var(--paper-2);
  }

  .word {
    margin-top: 2px;
    font-size: 12.5px;
    font-weight: 400;
    padding: 0 6px;
    border-left: 3px solid var(--c, var(--gold));
    background: rgb(239 230 216 / 0.06);
    white-space: nowrap;
    justify-self: start;
  }

  .word.pass {
    --c: var(--st-pass);
  }
  .word.warn {
    --c: var(--st-warn);
  }
  .word.fail {
    --c: var(--st-fail);
  }
  .word.skip {
    --c: var(--st-skip);
  }
  .word.error {
    --c: var(--vermilion-hi);
  }

  .time {
    font-size: 12.5px;
    font-weight: 400;
    color: var(--paper-dim);
    white-space: nowrap;
  }

  .raw {
    display: block;
    margin-top: 2px;
    font-size: 12px;
    font-weight: 400;
    color: var(--paper-dim);
    overflow-wrap: anywhere;
  }

  .muted {
    margin: 0;
    color: var(--paper-dim);
  }
</style>
