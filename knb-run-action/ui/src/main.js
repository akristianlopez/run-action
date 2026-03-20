import { createApp } from 'vue'
import App from './App.vue'
import './style.css'


// app.mount('#app')
// Contrat de montage que le Shell utilisera pour injecter ce module
const mount = (el) => {
    const app = createApp(App)
    app.mount(el)
    return () => app.unmount()
}

// Pour le développement local (mode standalone)
if (import.meta.env.DEV) {
    const devRoot = document.querySelector('#app')
    if (devRoot) mount(devRoot)
}

// Export de la fonction mount pour le Module Federation
export { mount }