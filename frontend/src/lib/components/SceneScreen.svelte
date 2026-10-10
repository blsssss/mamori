<!-- the main VN scene: background of the selected check, Reimu, the chapters and the text box -->
<script lang="ts">
import { BG } from '../assets'
import { meta } from '../checks'
import { useStore } from '../state.svelte'
import Backdrop from './Backdrop.svelte'
import ChapterList from './ChapterList.svelte'
import Grain from './Grain.svelte'
import Sprite from './Sprite.svelte'
import TextBox from './TextBox.svelte'
import TopBar from './TopBar.svelte'

let { inert = false }: { inert?: boolean } = $props()

const store = useStore()
let flash = $state<HTMLDivElement>()
let lastFx = 0

$effect(() => {
  const fx = store.fx
  if (fx?.kind !== 'flash' || fx.seq === lastFx || !flash) return
  lastFx = fx.seq
  if (store.reducedMotion || typeof flash.animate !== 'function') return
  flash.animate([{ opacity: 0 }, { opacity: 0.55, offset: 0.18 }, { opacity: 0 }], {
    duration: 650,
    easing: 'ease-out',
  })
})
</script>

<main class="scene" {inert}>
  <Backdrop src={BG[meta(store.selected).bg]} drift={!store.reducedMotion} />
  <Sprite />
  <Grain />
  <TopBar />
  <ChapterList />
  <TextBox />
  <div class="flash" bind:this={flash} aria-hidden="true"></div>
</main>

<style>
  .scene {
    --box-h: 200px;
    position: absolute;
    inset: 0;
    overflow: clip;
  }

  .flash {
    position: absolute;
    inset: 0;
    z-index: 6;
    pointer-events: none;
    opacity: 0;
    background: radial-gradient(ellipse at 70% 40%, var(--vermilion-hi), var(--vermilion) 55%, var(--blood));
    mix-blend-mode: screen;
  }

  @media (max-height: 700px) {
    .scene {
      --box-h: 164px;
    }
  }
</style>
