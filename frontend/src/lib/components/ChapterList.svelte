<!-- the four checks as VN chapters plus module 5, the way to the results -->
<script lang="ts">
import { BG } from '../assets'
import { CHECKS } from '../checks'
import { duration, statusWord } from '../i18n'
import { checkTitle } from '../report'
import { useStore } from '../state.svelte'
import { type CheckId, checkStatus } from '../types'
import Seal from './Seal.svelte'

const store = useStore()

function word(id: CheckId): string {
  const st = store.checks[id]
  if (st.phase === 'running' || st.phase === 'notice') return store.t('chapter.active')
  const s = checkStatus(st)
  return s ? statusWord(store.lang, s) : store.t('chapter.idle')
}

function time(id: CheckId): string {
  const ms = store.elapsed(id)
  return ms === null ? '' : duration(store.lang, ms)
}

const overall = $derived(store.overall)
const overallAria = $derived(
  `${store.t('chapter.selectAria', {
    module: store.t('overall.module'),
    title: store.t('overall.label'),
    status: overall ? statusWord(store.lang, overall.status) : store.t('overall.pending'),
  })}. ${store.t('overall.open')}`,
)
</script>

<nav class="chapters" aria-label={store.t('chapters.label')}>
  <ol>
    {#each CHECKS as c (c.id)}
      {@const st = store.checks[c.id]}
      {@const active = st.phase === 'running' || st.phase === 'notice'}
      {@const status = checkStatus(st)}
      {@const title = checkTitle(store.lang, c.id)}
      {@const module = store.t('check.module', { n: c.module })}
      <li class="chapter" class:selected={store.selected === c.id} class:active style:--thumb="url({BG[c.bg]})">
        <button
          type="button"
          class="pick"
          data-check={c.id}
          aria-current={store.selected === c.id ? 'true' : undefined}
          aria-label={store.t('chapter.selectAria', { module, title, status: word(c.id) })}
          onclick={() => store.select(c.id)}
        >
          <span class="num brush" lang="ja" aria-hidden="true">{c.numeral}</span>
          <span class="sub"><span lang="ja">{c.jp}</span><span class="dot">·</span>{module}</span>
          <span class="title">{title}</span>
          <span class="meta">
            <span class="word {active ? 'running' : (status ?? 'none')}">{word(c.id)}</span>
            {#if time(c.id)}<span class="dot">·</span><span class="time">{time(c.id)}</span>{/if}
          </span>
          <span class="seal"><Seal {status} running={active} size={36} /></span>
        </button>
        <button
          type="button"
          class="run"
          disabled={store.busy}
          aria-label={store.t(active ? 'chapter.runningAria' : status ? 'chapter.rerunAria' : 'chapter.runAria', { title })}
          onclick={() => store.run(c.id)}
        >
          {store.t(active ? 'chapter.running' : status ? 'chapter.rerun' : 'chapter.run')}
        </button>
      </li>
    {/each}
  </ol>

  <button type="button" class="overall" aria-label={overallAria} onclick={() => store.go('log')}>
    <span class="num brush" lang="ja" aria-hidden="true">{store.t('jp.five')}</span>
    <span class="sub"><span lang="ja">{store.t('jp.log')}</span><span class="dot">·</span>{store.t('overall.module')}</span>
    <span class="title">{store.t('overall.label')}</span>
    <span class="meta">
      {#if overall}
        <span class="word {overall.status}">{statusWord(store.lang, overall.status)}</span>
      {:else}
        <span class="word none">{store.t('overall.pending')}</span>
      {/if}
      <span class="open">{store.t('overall.open')}</span>
    </span>
    <span class="seal"><Seal status={overall?.status ?? null} size={30} /></span>
  </button>
</nav>

<style>
  .chapters {
    position: absolute;
    z-index: 3;
    top: calc(var(--topbar) + 14px);
    left: var(--gutter);
    width: min(400px, 36vw);
    max-height: calc(100vh - var(--topbar) - var(--box-h) - 52px);
    overflow-y: auto;
    overflow-x: hidden;
    scrollbar-width: thin;
    scrollbar-color: var(--gold) transparent;
    padding: 2px 4px 4px 2px;
  }

  ol {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 7px;
  }

  .chapter {
    position: relative;
  }

  .pick,
  .overall {
    position: relative;
    width: 100%;
    display: grid;
    grid-template-columns: 40px 1fr 40px;
    grid-template-areas:
      'num sub seal'
      'num title seal'
      'num meta meta';
    column-gap: 10px;
    align-items: center;
    padding: 8px 10px 8px 8px;
    text-align: left;
    border: 1px solid rgb(176 141 87 / 0.4);
    background:
      linear-gradient(90deg, rgb(20 16 18 / 0.94) 55%, rgb(20 16 18 / 0.7)),
      var(--thumb, none) center / cover;
    transition:
      border-color 0.2s,
      background-color 0.2s,
      transform 0.2s;
  }

  .pick::before,
  .overall::before {
    content: '';
    position: absolute;
    left: -1px;
    top: -1px;
    bottom: -1px;
    width: 3px;
    background: transparent;
    transition: background 0.2s;
  }

  .pick:hover,
  .overall:hover {
    border-color: var(--gold-hi);
  }

  .selected .pick {
    border-color: var(--gold-hi);
    background:
      linear-gradient(90deg, rgb(48 22 22 / 0.95) 50%, rgb(30 14 14 / 0.6)),
      var(--thumb, none) center / cover;
    transform: translateX(4px);
  }

  .selected .pick::before {
    background: var(--vermilion-hi);
  }

  .num {
    grid-area: num;
    align-self: start;
    justify-self: center;
    font-size: 34px;
    line-height: 1.1;
    color: var(--vermilion-hi);
  }

  .sub {
    grid-area: sub;
    font-size: 12px;
    line-height: 1.3;
    color: var(--gold-hi);
    letter-spacing: 0.04em;
  }

  .dot {
    margin: 0 6px;
    color: var(--gold);
  }

  .title {
    grid-area: title;
    font-size: 16px;
    line-height: 1.25;
    color: var(--paper);
  }

  .meta {
    grid-area: meta;
    display: flex;
    align-items: center;
    min-height: 24px;
    margin-top: 3px;
    padding-right: 92px;
    font-size: 12.5px;
    color: var(--paper-2);
  }

  .word {
    padding: 0 6px;
    border-left: 3px solid var(--c, var(--gold));
    background: rgb(239 230 216 / 0.06);
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
  .word.running {
    --c: var(--gold-hi);
  }
  .word.none {
    --c: rgb(176 141 87 / 0.5);
    color: var(--paper-dim);
  }

  .time {
    font-variant-numeric: tabular-nums;
  }

  .seal {
    grid-area: seal;
    justify-self: end;
    align-self: start;
    display: grid;
  }

  .run {
    position: absolute;
    right: 10px;
    bottom: 8px;
    min-width: 82px;
    height: 24px;
    padding: 0 10px;
    border: 1px solid var(--vermilion);
    background: var(--fill-primary);
    color: var(--paper);
    font-size: 12px;
    letter-spacing: 0.04em;
  }

  .selected .run {
    transform: translateX(4px);
  }

  .run:hover:not(:disabled) {
    background: var(--fill-primary-hover);
    border-color: var(--vermilion-hi);
  }

  .run:disabled {
    opacity: 0.5;
  }

  .active .run:disabled {
    opacity: 0.85;
    border-color: var(--gold);
    background: rgb(20 16 18 / 0.8);
  }

  .overall {
    margin-top: 10px;
    background: linear-gradient(90deg, rgb(20 16 18 / 0.94), rgb(42 39 64 / 0.75));
  }

  .overall .num {
    font-size: 28px;
  }

  .overall .meta {
    padding-right: 0;
    justify-content: space-between;
    gap: 8px;
  }

  .open {
    color: var(--gold-hi);
    font-size: 12px;
    text-decoration: underline;
    text-decoration-color: rgb(212 180 124 / 0.5);
    text-underline-offset: 3px;
  }

  .overall:hover .open {
    color: var(--paper);
  }

  @media (max-height: 700px) {
    .pick,
    .overall {
      padding-top: 5px;
      padding-bottom: 5px;
    }

    /* the module line stays, the teacher maps the card to the guide by it; the card itself opens the log */
    .overall .open {
      display: none;
    }

    .sub {
      line-height: 1.15;
    }

    .meta {
      min-height: 22px;
      margin-top: 1px;
    }

    .run {
      height: 22px;
      bottom: 5px;
    }

    .num {
      font-size: 30px;
    }

    .title {
      font-size: 15px;
    }

    ol {
      gap: 3px;
    }

    .overall {
      margin-top: 5px;
    }
  }
</style>
