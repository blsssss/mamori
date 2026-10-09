<script lang="ts">
import { Info, Quit, Run, SaveReport, Summarize } from '../wailsjs/go/main/App'
import type { check, main, report } from '../wailsjs/go/models'
import { EventsOn } from '../wailsjs/runtime/runtime'

const checks = [
  { id: 'internet', title: '1. Подключение к интернету' },
  { id: 'inventory', title: '2. Наличие антивируса и межсетевого экрана' },
  { id: 'firewall', title: '3. Работоспособность межсетевого экрана' },
  { id: 'antivirus', title: '4. Работоспособность антивируса' },
]

let info = $state<main.AppInfo | null>(null)
let results = $state<Record<string, check.Result>>({})
let live = $state<Record<string, check.Finding[]>>({})
let running = $state<string | null>(null)
let summary = $state<report.Summary | null>(null)
let error = $state('')

Info().then((i) => (info = i))
EventsOn('finding', (e: { check: string; finding: check.Finding }) => {
  live[e.check] = [...(live[e.check] ?? []), e.finding]
})

async function run(id: string): Promise<void> {
  running = id
  error = ''
  live[id] = []
  try {
    results[id] = await Run(id, { policyTargets: [], eicarWaitSec: 15, allowElevation: true })
  } catch (e) {
    error = String(e)
  } finally {
    running = null
  }
}

async function runAll(): Promise<void> {
  for (const c of checks) await run(c.id)
}

async function summarize(): Promise<void> {
  summary = await Summarize(Object.values(results))
}

async function save(): Promise<void> {
  if (!summary) await summarize()
  await SaveReport('mamori-report.json', JSON.stringify(summary, null, 2))
}

function clear(): void {
  results = {}
  live = {}
  summary = null
}

const params = (p?: Record<string, string>) =>
  p
    ? Object.entries(p)
        .map(([k, v]) => `${k}=${v}`)
        .join('; ')
    : ''
</script>

<main>
  <header>
    <h1>mamori</h1>
    {#if info}
      <p>{info.system.product} {info.system.version}, {info.system.build}, {info.system.host}, версия {info.version}</p>
    {/if}
  </header>

  <div class="bar">
    <button onclick={runAll} disabled={running !== null}>Проверить всё</button>
    <button onclick={clear} disabled={running !== null}>Очистить</button>
    <button onclick={() => Quit()}>Выход</button>
  </div>

  {#each checks as c (c.id)}
    {@const r = results[c.id]}
    {@const list = r ? r.findings : (live[c.id] ?? [])}
    <section>
      <h2>{c.title}</h2>
      <button onclick={() => run(c.id)} disabled={running !== null}>
        {running === c.id ? 'Выполняется...' : 'Проверить'}
      </button>
      {#if r}
        <p class="verdict {r.status}">{r.status}: {r.code} ({r.elapsedMs} мс)</p>
      {/if}
      <ul>
        {#each list as f, i (i)}
          <li class={f.status}><code>{f.status}</code> {f.code} <small>{params(f.params)}</small></li>
        {/each}
      </ul>
    </section>
  {/each}

  <section>
    <h2>5. Вывод результатов</h2>
    <button onclick={summarize}>Сводка</button>
    <button onclick={save}>Сохранить отчёт</button>
    {#if summary}
      <p class="verdict {summary.status}">{summary.status}: {summary.code}</p>
    {/if}
  </section>

  {#if error}<p class="error">{error}</p>{/if}
</main>
