<script lang="ts">
import { ICON } from '../assets'
import { useStore } from '../state.svelte'
import LangSwitch from './LangSwitch.svelte'

const store = useStore()
const sys = $derived(store.info?.system)
const version = $derived(
  store.info ? store.info.version + (store.info.commit ? ` (${store.info.commit})` : '') : '',
)
</script>

<header class="topbar">
  <span class="brand">
    <img src={ICON} alt="" width="18" height="18" />
    <span class="name">{store.t('app.name')}</span>
    {#if version}<span class="ver">{version}</span>{/if}
  </span>
  <span class="sys" title={sys ? store.t('top.system', { ...sys }) : undefined}>
    {#if sys}
      <span>{store.t('top.system', { ...sys })}</span>
      {#if sys.host}<span class="sep" aria-hidden="true">·</span><span>{store.t('top.host', { host: sys.host })}</span>{/if}
      {#if sys.user}<span class="sep" aria-hidden="true">·</span><span>{store.t('top.user', { user: sys.user })}</span>{/if}
    {:else}
      <span class="muted">{store.t('top.noInfo')}</span>
    {/if}
  </span>
  {#if store.info?.elevated}
    <span class="admin" title={store.t('top.adminHint')}>{store.t('top.admin')}</span>
  {/if}
  <LangSwitch />
</header>

<style>
  .topbar {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    z-index: 5;
    height: var(--topbar);
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 0 calc(var(--gutter) + var(--demo-pad, 0px)) 0 var(--gutter);
    background: linear-gradient(180deg, rgb(14 10 12 / 0.92), rgb(14 10 12 / 0.75));
    border-bottom: 1px solid rgb(176 141 87 / 0.35);
    font-size: 12.5px;
    color: var(--paper-2);
  }

  .brand {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    flex: none;
  }

  .name {
    color: var(--paper);
    letter-spacing: 0.06em;
  }

  .ver {
    color: var(--gold-hi);
  }

  .sys {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  .sep {
    margin: 0 7px;
    color: var(--gold);
  }

  .muted {
    color: var(--paper-dim);
  }

  .admin {
    flex: none;
    padding: 0 8px;
    border: 1px solid var(--gold-hi);
    color: var(--gold-hi);
    font-size: 11.5px;
    letter-spacing: 0.06em;
  }
</style>
