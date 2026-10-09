import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'
import { connect } from './lib/api'
import { Store, storeContext } from './lib/state.svelte'

const target = document.getElementById('app')
if (!target) throw new Error('no #app element')

const app = mount(App, { target, context: storeContext(new Store(await connect())) })

export default app
