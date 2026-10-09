<!-- vignette, film grain and faint scanlines over a CG, generated, no image files -->
<script lang="ts">
let { strength = 1 }: { strength?: number } = $props()
</script>

<div class="fx" aria-hidden="true" style:--strength={strength}>
  <div class="vignette"></div>
  <svg class="grain" preserveAspectRatio="none">
    <rect width="100%" height="100%" filter="url(#film-grain)" />
  </svg>
  <div class="scanlines"></div>
</div>

<style>
  .fx {
    position: absolute;
    inset: 0;
    pointer-events: none;
    overflow: hidden;
  }

  .vignette {
    position: absolute;
    inset: 0;
    background:
      radial-gradient(ellipse 75% 70% at 50% 45%, transparent 55%, rgb(12 8 10 / calc(0.7 * var(--strength))) 100%),
      linear-gradient(180deg, rgb(12 8 10 / 0.35), transparent 18%, transparent 70%, rgb(12 8 10 / 0.5));
  }

  .grain {
    position: absolute;
    top: -50%;
    left: -50%;
    width: 200%;
    height: 200%;
    opacity: calc(0.11 * var(--strength));
    mix-blend-mode: overlay;
    will-change: transform;
    animation: grain 0.9s steps(6) infinite;
  }

  .scanlines {
    position: absolute;
    inset: 0;
    background: repeating-linear-gradient(180deg, transparent 0 2px, rgb(10 6 8 / 0.06) 2px 3px);
    mix-blend-mode: multiply;
  }

  @keyframes grain {
    0% {
      transform: translate(0, 0);
    }
    20% {
      transform: translate(-7%, 4%);
    }
    40% {
      transform: translate(5%, -6%);
    }
    60% {
      transform: translate(-4%, -3%);
    }
    80% {
      transform: translate(6%, 5%);
    }
    100% {
      transform: translate(-2%, 7%);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .grain {
      animation: none;
    }
  }
</style>
